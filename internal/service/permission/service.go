// Package permission provides permission-group CRUD and the user effective
// services computation used by MCP proxy authorization (design doc §7).
package permission

import (
	"errors"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"gorm.io/gorm"
)

// Service manages PermissionGroups, their members and service grants.
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateGroup creates a new active permission group owned by createdByID.
func (s *Service) CreateGroup(name, displayName, description string, createdByID uint) (*model.PermissionGroup, error) {
	if name == "" {
		return nil, fmt.Errorf("group name must not be empty: %w", apierrors.ErrInvalidInput)
	}
	g := &model.PermissionGroup{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Status:      model.PermissionGroupStatusActive,
		CreatedByID: createdByID,
	}
	if err := s.db.Create(g).Error; err != nil {
		return nil, err
	}
	return g, nil
}

// GetGroup retrieves a group by id.
func (s *Service) GetGroup(id uint) (*model.PermissionGroup, error) {
	var g model.PermissionGroup
	if err := s.db.First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("permission group not found: %w", apierrors.ErrNotFound)
		}
		return nil, err
	}
	return &g, nil
}

// ListGroups returns all non-deleted permission groups.
func (s *Service) ListGroups() ([]model.PermissionGroup, error) {
	var groups []model.PermissionGroup
	if err := s.db.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// UpdateGroup updates a group's display name and description.
func (s *Service) UpdateGroup(id uint, displayName, description string) error {
	return s.db.Model(&model.PermissionGroup{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"display_name": displayName,
			"description":  description,
		}).Error
}

// DisableGroup sets a group's status to disabled (soft-disable, preserves
// membership and grants for audit history).
func (s *Service) DisableGroup(id uint) error {
	return s.db.Model(&model.PermissionGroup{}).Where("id = ?", id).
		Update("status", model.PermissionGroupStatusDisabled).Error
}

// AddMember adds userID to groupID. Returns nil if the membership already
// exists (idempotent).
func (s *Service) AddMember(groupID, userID, assignedByID uint) error {
	m := &model.PermissionGroupMember{
		PermissionGroupID: groupID,
		UserID:            userID,
		AssignedByID:      assignedByID,
	}
	if err := s.db.Where("permission_group_id = ? AND user_id = ?", groupID, userID).
		Assign(map[string]interface{}{"assigned_by_id": assignedByID}).
		FirstOrCreate(m).Error; err != nil {
		return err
	}
	return nil
}

// RemoveMember removes userID from groupID. Returns gorm.ErrRecordNotFound if
// the membership does not exist.
func (s *Service) RemoveMember(groupID, userID uint) error {
	result := s.db.Where("permission_group_id = ? AND user_id = ?", groupID, userID).
		Delete(&model.PermissionGroupMember{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("membership not found: %w", apierrors.ErrNotFound)
	}
	return result.Error
}

// AddService grants groupID access to mcpServerID. Idempotent.
func (s *Service) AddService(groupID, mcpServerID, assignedByID uint) error {
	gs := &model.PermissionGroupService{
		PermissionGroupID: groupID,
		McpServerID:       mcpServerID,
		AssignedByID:      assignedByID,
	}
	if err := s.db.Where("permission_group_id = ? AND mcp_server_id = ?", groupID, mcpServerID).
		Assign(map[string]interface{}{"assigned_by_id": assignedByID}).
		FirstOrCreate(gs).Error; err != nil {
		return err
	}
	return nil
}

// RemoveService revokes groupID's access to mcpServerID.
func (s *Service) RemoveService(groupID, mcpServerID uint) error {
	result := s.db.Where("permission_group_id = ? AND mcp_server_id = ?", groupID, mcpServerID).
		Delete(&model.PermissionGroupService{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("grant not found: %w", apierrors.ErrNotFound)
	}
	return result.Error
}

// UserEffectiveServices returns the set of MCP server IDs that userID is
// permitted to call, computed as the union of all active permission groups the
// user belongs to (design doc §7.1).
func (s *Service) UserEffectiveServices(userID uint) ([]uint, error) {
	var ids []uint
	err := s.db.Table("permission_group_services pgs").
		Select("DISTINCT pgs.mcp_server_id").
		Joins("JOIN permission_groups pg ON pg.id = pgs.permission_group_id").
		Joins("JOIN permission_group_members pgm ON pgm.permission_group_id = pgs.permission_group_id").
		Where("pgm.user_id = ? AND pg.status = ? AND pg.deleted_at IS NULL", userID, model.PermissionGroupStatusActive).
		Pluck("pgs.mcp_server_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// ListMembers returns all current members of a group.
func (s *Service) ListMembers(groupID uint) ([]model.PermissionGroupMember, error) {
	var members []model.PermissionGroupMember
	if err := s.db.Where("permission_group_id = ?", groupID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// ListServices returns all service grants of a group.
func (s *Service) ListServices(groupID uint) ([]model.PermissionGroupService, error) {
	var services []model.PermissionGroupService
	if err := s.db.Where("permission_group_id = ?", groupID).Find(&services).Error; err != nil {
		return nil, err
	}
	return services, nil
}
