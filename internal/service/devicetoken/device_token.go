// Package devicetoken provides personal device-token lifecycle management.
package devicetoken

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/auth"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	deviceTokenPrefix       = "mcpdt_"
	deviceTokenRandomBytes  = 32
	deviceTokenDefaultTTL   = 90 * 24 * time.Hour
	deviceTokenMaxActive    = 10
	deviceTokenStatusActive = types.UserStatusActive
)

// EffectiveServiceIDsProvider returns the user's currently effective service IDs.
type EffectiveServiceIDsProvider interface {
	EffectiveServiceIDs(userID uint) (map[uint]struct{}, error)
}

// CreateInput describes a self-service device token request.
type CreateInput struct {
	Name               string
	Scope              types.DeviceTokenScope
	ExpiresAt          *time.Time
	ServiceIDs         []uint
	SelectedServiceIDs []uint
}

// Service manages personal device tokens.
type Service struct {
	db                *gorm.DB
	effectiveServices EffectiveServiceIDsProvider
	now               func() time.Time
}

// NewService creates a device-token service.
func NewService(db *gorm.DB, effectiveServices EffectiveServiceIDsProvider) *Service {
	return &Service{
		db:                db,
		effectiveServices: effectiveServices,
		now:               time.Now,
	}
}

// Create creates a device token for an active user. The plaintext token is
// returned exactly once; persistent storage only keeps the hash and short prefix.
func (s *Service) Create(userID uint, input CreateInput) (*model.DeviceToken, string, error) {
	if userID == 0 {
		return nil, "", fmt.Errorf("user ID must be greater than zero: %w", apierrors.ErrInvalidInput)
	}
	if s.effectiveServices == nil {
		return nil, "", fmt.Errorf("effective service provider is required: %w", apierrors.ErrInvalidInput)
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, "", fmt.Errorf("device name is required: %w", apierrors.ErrInvalidInput)
	}

	scope := input.Scope
	if scope == "" {
		scope = types.DeviceTokenScopeInheritAll
	}
	if scope != types.DeviceTokenScopeInheritAll && scope != types.DeviceTokenScopeRestricted {
		return nil, "", fmt.Errorf("invalid device token scope: %w", apierrors.ErrInvalidInput)
	}

	now := s.now().UTC()
	expiresAt, err := resolveDeviceTokenExpiry(now, input.ExpiresAt)
	if err != nil {
		return nil, "", err
	}

	selectedServiceIDs := uniqueUintIDs(selectedServiceIDs(input))
	effectiveServiceIDs, err := s.effectiveServices.EffectiveServiceIDs(userID)
	if err != nil {
		return nil, "", err
	}
	if scope == types.DeviceTokenScopeRestricted {
		for _, serviceID := range selectedServiceIDs {
			if _, ok := effectiveServiceIDs[serviceID]; !ok {
				return nil, "", fmt.Errorf("selected service %d is outside the user's effective scope: %w", serviceID, apierrors.ErrInvalidInput)
			}
		}
	}

	plain, err := auth.Generate(deviceTokenPrefix, deviceTokenRandomBytes)
	if err != nil {
		return nil, "", fmt.Errorf("generate device token: %w", err)
	}

	token := &model.DeviceToken{
		UserID:      userID,
		Name:        name,
		TokenPrefix: auth.Prefix(plain),
		TokenHash:   auth.Hash(plain),
		Scope:       scope,
		ExpiresAt:   expiresAt,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := loadActiveUserForUpdate(tx, userID); err != nil {
			return err
		}

		var activeCount int64
		if err := tx.Model(&model.DeviceToken{}).
			Where("user_id = ?", userID).
			Where("revoked_at IS NULL").
			Where("expires_at > ?", now).
			Count(&activeCount).Error; err != nil {
			return fmt.Errorf("count active device tokens: %w", err)
		}
		if activeCount >= deviceTokenMaxActive {
			return fmt.Errorf("user already has the maximum number of active device tokens: %w", apierrors.ErrInvalidInput)
		}

		var existingCount int64
		if err := tx.Model(&model.DeviceToken{}).
			Where("user_id = ? AND name = ?", userID, name).
			Count(&existingCount).Error; err != nil {
			return fmt.Errorf("check duplicate device token name: %w", err)
		}
		if existingCount > 0 {
			return fmt.Errorf("device name already exists for this user: %w", apierrors.ErrInvalidInput)
		}

		if err := tx.Create(token).Error; err != nil {
			return fmt.Errorf("create device token: %w", err)
		}

		if scope != types.DeviceTokenScopeRestricted || len(selectedServiceIDs) == 0 {
			return nil
		}

		rows := make([]model.DeviceTokenService, 0, len(selectedServiceIDs))
		for _, serviceID := range selectedServiceIDs {
			rows = append(rows, model.DeviceTokenService{
				DeviceTokenID: token.ID,
				McpServerID:   serviceID,
			})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("create restricted device-token services: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	return token, plain, nil
}

// ListForUser lists the device tokens belonging to the specified user.
func (s *Service) ListForUser(userID uint) ([]model.DeviceToken, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID must be greater than zero: %w", apierrors.ErrInvalidInput)
	}

	var tokens []model.DeviceToken
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC, id DESC").Find(&tokens).Error; err != nil {
		return nil, fmt.Errorf("list device tokens: %w", err)
	}
	return tokens, nil
}

// Revoke revokes a device token. Owners can revoke their own tokens; system
// administrators can revoke any token. Revocation is idempotent.
func (s *Service) Revoke(userID, tokenID uint, systemAdmin bool) error {
	if tokenID == 0 {
		return fmt.Errorf("token ID must be greater than zero: %w", apierrors.ErrInvalidInput)
	}

	var token model.DeviceToken
	if err := s.db.First(&token, tokenID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("device token not found: %w", apierrors.ErrNotFound)
		}
		return fmt.Errorf("load device token: %w", err)
	}

	if !systemAdmin && token.UserID != userID {
		return fmt.Errorf("device token not found: %w", apierrors.ErrNotFound)
	}
	if token.RevokedAt != nil {
		return nil
	}

	revokedAt := s.now().UTC()
	if err := s.db.Model(&token).Update("revoked_at", revokedAt).Error; err != nil {
		return fmt.Errorf("revoke device token: %w", err)
	}
	return nil
}

// Authenticate resolves a plaintext device token to its user, token metadata,
// and effective service set. Successful authentication only updates usage
// metadata and never refreshes expiry.
func (s *Service) Authenticate(plain, ip, clientInfo string) (*model.User, *model.DeviceToken, map[uint]struct{}, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return nil, nil, nil, fmt.Errorf("invalid device token: %w", apierrors.ErrInvalidCredentials)
	}
	if s.effectiveServices == nil {
		return nil, nil, nil, fmt.Errorf("effective service provider is required: %w", apierrors.ErrInvalidInput)
	}

	var token model.DeviceToken
	if err := s.db.Where("token_hash = ?", auth.Hash(plain)).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, fmt.Errorf("invalid device token: %w", apierrors.ErrInvalidCredentials)
		}
		return nil, nil, nil, fmt.Errorf("load device token: %w", err)
	}

	now := s.now().UTC()
	if token.RevokedAt != nil {
		return nil, nil, nil, fmt.Errorf("device token revoked: %w", apierrors.ErrInvalidCredentials)
	}
	if !token.ExpiresAt.After(now) {
		return nil, nil, nil, fmt.Errorf("device token expired: %w", apierrors.ErrInvalidCredentials)
	}

	user, err := loadActiveUser(s.db, token.UserID)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return nil, nil, nil, fmt.Errorf("invalid device token user: %w", apierrors.ErrInvalidCredentials)
		}
		return nil, nil, nil, err
	}

	effectiveServiceIDs, err := s.effectiveServices.EffectiveServiceIDs(user.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	if token.Scope == types.DeviceTokenScopeRestricted {
		effectiveServiceIDs, err = s.intersectRestrictedServices(token.ID, effectiveServiceIDs)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	lastUsedAt := now
	updates := map[string]any{
		"last_used_at": lastUsedAt,
		"last_used_ip": strings.TrimSpace(ip),
		"client_info":  strings.TrimSpace(clientInfo),
	}
	if err := s.db.Model(&token).Updates(updates).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("update device token usage: %w", err)
	}
	token.LastUsedAt = &lastUsedAt
	token.LastUsedIP = updates["last_used_ip"].(string)
	token.ClientInfo = updates["client_info"].(string)

	return user, &token, effectiveServiceIDs, nil
}

func (s *Service) intersectRestrictedServices(tokenID uint, userEffective map[uint]struct{}) (map[uint]struct{}, error) {
	if len(userEffective) == 0 {
		return map[uint]struct{}{}, nil
	}

	var rows []model.DeviceTokenService
	if err := s.db.Where("device_token_id = ?", tokenID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load restricted device-token services: %w", err)
	}

	result := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := userEffective[row.McpServerID]; ok {
			result[row.McpServerID] = struct{}{}
		}
	}
	return result, nil
}

func loadActiveUser(db *gorm.DB, userID uint) (*model.User, error) {
	return loadActiveUserWithQuery(db, userID)
}

func loadActiveUserForUpdate(db *gorm.DB, userID uint) (*model.User, error) {
	return loadActiveUserWithQuery(db.Clauses(clause.Locking{Strength: "UPDATE"}), userID)
}

func loadActiveUserWithQuery(query *gorm.DB, userID uint) (*model.User, error) {
	var user model.User
	if err := query.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %w", apierrors.ErrNotFound)
		}
		return nil, fmt.Errorf("load user: %w", err)
	}
	if user.Status != deviceTokenStatusActive {
		return nil, fmt.Errorf("active user required: %w", apierrors.ErrNotFound)
	}
	return &user, nil
}

func resolveDeviceTokenExpiry(now time.Time, requested *time.Time) (time.Time, error) {
	if requested == nil {
		return now.Add(deviceTokenDefaultTTL), nil
	}

	expiresAt := requested.UTC()
	if !expiresAt.After(now) {
		return time.Time{}, fmt.Errorf("device token expiry must be in the future: %w", apierrors.ErrInvalidInput)
	}
	if expiresAt.Sub(now) > deviceTokenDefaultTTL {
		return time.Time{}, fmt.Errorf("device token expiry cannot exceed 90 days: %w", apierrors.ErrInvalidInput)
	}
	return expiresAt, nil
}

func selectedServiceIDs(input CreateInput) []uint {
	if len(input.ServiceIDs) > 0 {
		return input.ServiceIDs
	}
	return input.SelectedServiceIDs
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
	return unique
}
