package permission

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const userStatusActive = "active"

var ErrPermissionGroupNotFound = fmt.Errorf("permission group not found: %w", apierrors.ErrNotFound)

type Service struct {
	db *gorm.DB
}

type CreateInput struct {
	Name        string
	DisplayName string
	Description string
}

type UpdateInput struct {
	DisplayName *string
	Description *string
}

type GroupDetail struct {
	Group      model.PermissionGroup `json:"group"`
	UserIDs    []uint                `json:"user_ids"`
	ServiceIDs []uint                `json:"service_ids"`
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(input CreateInput) (*model.PermissionGroup, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Name == "" {
		return nil, fmt.Errorf("permission group name is required: %w", apierrors.ErrInvalidInput)
	}
	if input.DisplayName == "" {
		input.DisplayName = input.Name
	}
	group := &model.PermissionGroup{
		Name: input.Name, DisplayName: input.DisplayName,
		Description: strings.TrimSpace(input.Description), Enabled: true,
	}
	if err := s.db.Create(group).Error; err != nil {
		return nil, fmt.Errorf("create permission group: %w", err)
	}
	return group, nil
}

func (s *Service) List() ([]model.PermissionGroup, error) {
	var groups []model.PermissionGroup
	if err := s.db.Order("name").Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("list permission groups: %w", err)
	}
	return groups, nil
}

func (s *Service) Get(groupID uint) (*GroupDetail, error) {
	var group model.PermissionGroup
	if err := s.db.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPermissionGroupNotFound
		}
		return nil, fmt.Errorf("load permission group: %w", err)
	}
	var users []model.PermissionGroupUser
	if err := s.db.Where("permission_group_id = ?", groupID).Order("user_id").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("load permission group users: %w", err)
	}
	var services []model.PermissionGroupMcpServer
	if err := s.db.Where("permission_group_id = ?", groupID).Order("mcp_server_id").Find(&services).Error; err != nil {
		return nil, fmt.Errorf("load permission group services: %w", err)
	}
	detail := &GroupDetail{Group: group, UserIDs: make([]uint, 0, len(users)), ServiceIDs: make([]uint, 0, len(services))}
	for _, row := range users {
		detail.UserIDs = append(detail.UserIDs, row.UserID)
	}
	for _, row := range services {
		detail.ServiceIDs = append(detail.ServiceIDs, row.McpServerID)
	}
	return detail, nil
}

func (s *Service) Update(groupID uint, input UpdateInput) error {
	updates := map[string]any{}
	if input.DisplayName != nil {
		value := strings.TrimSpace(*input.DisplayName)
		if value == "" {
			return fmt.Errorf("display name is required: %w", apierrors.ErrInvalidInput)
		}
		updates["display_name"] = value
	}
	if input.Description != nil {
		updates["description"] = strings.TrimSpace(*input.Description)
	}
	if len(updates) == 0 {
		return fmt.Errorf("nothing to update: %w", apierrors.ErrInvalidInput)
	}
	result := s.db.Model(&model.PermissionGroup{}).Where("id = ?", groupID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update permission group: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPermissionGroupNotFound
	}
	return nil
}

func (s *Service) EffectiveServiceIDs(userID uint) (map[uint]struct{}, error) {
	if err := s.ensureActiveUser(userID); err != nil {
		return nil, err
	}

	var memberships []model.PermissionGroupUser
	if err := s.db.
		Where("user_id = ?", userID).
		Find(&memberships).Error; err != nil {
		return nil, fmt.Errorf("load permission group memberships: %w", err)
	}
	if len(memberships) == 0 {
		return map[uint]struct{}{}, nil
	}

	groupIDs := make([]uint, 0, len(memberships))
	for id := range uniqueUintIDsFromMemberships(memberships) {
		groupIDs = append(groupIDs, id)
	}
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })

	var groups []model.PermissionGroup
	if err := s.db.
		Where("id IN ?", groupIDs).
		Where("enabled = ?", true).
		Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("load enabled permission groups: %w", err)
	}
	if len(groups) == 0 {
		return map[uint]struct{}{}, nil
	}

	enabledGroupIDs := make([]uint, 0, len(groups))
	for _, group := range groups {
		enabledGroupIDs = append(enabledGroupIDs, group.ID)
	}

	var serviceMemberships []model.PermissionGroupMcpServer
	if err := s.db.
		Where("permission_group_id IN ?", enabledGroupIDs).
		Find(&serviceMemberships).Error; err != nil {
		return nil, fmt.Errorf("load permission group services: %w", err)
	}
	if len(serviceMemberships) == 0 {
		return map[uint]struct{}{}, nil
	}

	serviceIDs := make([]uint, 0, len(serviceMemberships))
	for id := range uniqueUintIDsFromServices(serviceMemberships) {
		serviceIDs = append(serviceIDs, id)
	}
	sort.Slice(serviceIDs, func(i, j int) bool { return serviceIDs[i] < serviceIDs[j] })

	var services []model.McpServer
	if err := s.db.
		Where("id IN ?", serviceIDs).
		Where("enabled = ?", true).
		Find(&services).Error; err != nil {
		return nil, fmt.Errorf("load enabled services: %w", err)
	}

	effective := make(map[uint]struct{}, len(services))
	for _, service := range services {
		effective[service.ID] = struct{}{}
	}
	return effective, nil
}

func (s *Service) ReplaceUsers(groupID uint, userIDs []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := ensurePermissionGroupExists(tx, groupID); err != nil {
			return err
		}

		uniqueIDs := uniqueUintIDs(userIDs)
		if err := ensureExistingUsers(tx, uniqueIDs); err != nil {
			return err
		}

		if err := tx.Where("permission_group_id = ?", groupID).Delete(&model.PermissionGroupUser{}).Error; err != nil {
			return fmt.Errorf("clear permission group users: %w", err)
		}

		if len(uniqueIDs) == 0 {
			return nil
		}

		rows := make([]model.PermissionGroupUser, 0, len(uniqueIDs))
		for _, userID := range uniqueIDs {
			rows = append(rows, model.PermissionGroupUser{
				PermissionGroupID: groupID,
				UserID:            userID,
			})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("create permission group users: %w", err)
		}
		return nil
	})
}

func (s *Service) ReplaceServices(groupID uint, serviceIDs []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := ensurePermissionGroupExists(tx, groupID); err != nil {
			return err
		}

		uniqueIDs := uniqueUintIDs(serviceIDs)
		if err := ensureExistingServices(tx, uniqueIDs); err != nil {
			return err
		}

		if err := tx.Where("permission_group_id = ?", groupID).Delete(&model.PermissionGroupMcpServer{}).Error; err != nil {
			return fmt.Errorf("clear permission group services: %w", err)
		}

		if len(uniqueIDs) == 0 {
			return nil
		}

		rows := make([]model.PermissionGroupMcpServer, 0, len(uniqueIDs))
		for _, serviceID := range uniqueIDs {
			rows = append(rows, model.PermissionGroupMcpServer{
				PermissionGroupID: groupID,
				McpServerID:       serviceID,
			})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("create permission group services: %w", err)
		}
		return nil
	})
}

func (s *Service) SetEnabled(groupID uint, enabled bool) error {
	result := s.db.Model(&model.PermissionGroup{}).
		Where("id = ?", groupID).
		Update("enabled", enabled)
	if result.Error != nil {
		return fmt.Errorf("set permission group enabled state: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPermissionGroupNotFound
	}
	return nil
}

func (s *Service) ensureActiveUser(userID uint) error {
	var count int64
	if err := s.db.Model(&model.User{}).
		Where("id = ?", userID).
		Where("status = ?", userStatusActive).
		Count(&count).Error; err != nil {
		return fmt.Errorf("check active user: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("active user %d not found: %w", userID, apierrors.ErrNotFound)
	}
	return nil
}

func ensurePermissionGroupExists(tx *gorm.DB, groupID uint) error {
	var group model.PermissionGroup
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPermissionGroupNotFound
		}
		return fmt.Errorf("check permission group: %w", err)
	}
	return nil
}

func ensureExistingUsers(tx *gorm.DB, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	var count int64
	if err := tx.Model(&model.User{}).
		Distinct("id").
		Where("id IN ?", ids).
		Count(&count).Error; err != nil {
		return fmt.Errorf("validate users: %w", err)
	}
	if int(count) != len(ids) {
		return fmt.Errorf("one or more users do not exist: %w", apierrors.ErrInvalidInput)
	}
	return nil
}

func ensureExistingServices(tx *gorm.DB, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	var count int64
	if err := tx.Model(&model.McpServer{}).
		Distinct("id").
		Where("id IN ?", ids).
		Count(&count).Error; err != nil {
		return fmt.Errorf("validate services: %w", err)
	}
	if int(count) != len(ids) {
		return fmt.Errorf("one or more services do not exist: %w", apierrors.ErrInvalidInput)
	}
	return nil
}

func uniqueUintIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[uint]struct{}, len(ids))
	unique := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
	return unique
}

func uniqueUintIDsFromMemberships(rows []model.PermissionGroupUser) map[uint]struct{} {
	unique := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		unique[row.PermissionGroupID] = struct{}{}
	}
	return unique
}

func uniqueUintIDsFromServices(rows []model.PermissionGroupMcpServer) map[uint]struct{} {
	unique := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		unique[row.McpServerID] = struct{}{}
	}
	return unique
}
