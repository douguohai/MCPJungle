// Package user implements the lifecycle of internal human accounts.
package user

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/auth"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const minimumPasswordLength = 12

type CreateUserInput struct {
	Username    string
	DisplayName string
	Role        types.UserRole
}

type UpdateUserInput struct {
	DisplayName *string
	Role        *types.UserRole
}

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService { return &UserService{db: db} }

// CreateAdminUser creates the initial active administrator used during server
// initialization. Ordinary accounts must be created with Create instead.
func (s *UserService) CreateAdminUser(username, password string) (*model.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		username = "admin"
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	account := &model.User{
		Username:           username,
		DisplayName:        username,
		Role:               types.UserRoleSystemAdmin,
		Status:             types.UserStatusActive,
		PasswordHash:       hash,
		MustChangePassword: false,
	}
	if err := s.db.Create(account).Error; err != nil {
		return nil, fmt.Errorf("create system administrator: %w", err)
	}
	return account, nil
}

// Create creates a pending account and returns its one-time initial password.
// The plaintext password is never stored.
func (s *UserService) Create(input CreateUserInput) (*model.User, string, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Username == "" {
		return nil, "", fmt.Errorf("username is required: %w", apierrors.ErrInvalidInput)
	}
	if input.DisplayName == "" {
		input.DisplayName = input.Username
	}
	if input.Role == "" {
		input.Role = types.UserRoleMember
	}
	if !types.IsValidUserRole(input.Role) {
		return nil, "", fmt.Errorf("invalid user role: %w", apierrors.ErrInvalidInput)
	}
	plain, err := auth.Generate("", 18)
	if err != nil {
		return nil, "", err
	}
	hash, err := hashPassword(plain)
	if err != nil {
		return nil, "", err
	}
	account := &model.User{
		Username:           input.Username,
		DisplayName:        input.DisplayName,
		Role:               input.Role,
		Status:             types.UserStatusPending,
		PasswordHash:       hash,
		MustChangePassword: true,
	}
	if err := s.db.Create(account).Error; err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}
	return account, plain, nil
}

func (s *UserService) VerifyPassword(username, password string) (*model.User, error) {
	var account model.User
	err := s.db.Where("username = ?", strings.TrimSpace(username)).First(&account).Error
	if err != nil || account.Status == types.UserStatusDisabled || account.PasswordHash == "" {
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find user: %w", err)
		}
		return nil, fmt.Errorf("authenticate user: %w", apierrors.ErrInvalidCredentials)
	}
	if bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)) != nil {
		return nil, fmt.Errorf("authenticate user: %w", apierrors.ErrInvalidCredentials)
	}
	now := time.Now().UTC()
	if err := s.db.Model(&account).Update("last_login_at", now).Error; err != nil {
		return nil, fmt.Errorf("record login: %w", err)
	}
	account.LastLoginAt = &now
	return &account, nil
}

func (s *UserService) ChangePassword(userID uint, current, next string) error {
	if err := validatePassword(next); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var account model.User
		if err := tx.First(&account, userID).Error; err != nil {
			return mapNotFound("user", err)
		}
		if account.Status == types.UserStatusDisabled ||
			bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(current)) != nil {
			return fmt.Errorf("change password: %w", apierrors.ErrInvalidCredentials)
		}
		hash, err := hashPassword(next)
		if err != nil {
			return err
		}
		updates := map[string]any{
			"password_hash":        hash,
			"must_change_password": false,
		}
		if account.Status == types.UserStatusPending {
			updates["status"] = types.UserStatusActive
		}
		if err := tx.Model(&account).Updates(updates).Error; err != nil {
			return fmt.Errorf("update password: %w", err)
		}
		revokedAt := time.Now().UTC()
		if err := tx.Model(&model.UserSession{}).
			Where("user_id = ? AND revoked_at IS NULL", account.ID).
			Update("revoked_at", revokedAt).Error; err != nil {
			return fmt.Errorf("revoke sessions after password change: %w", err)
		}
		return nil
	})
}

func (s *UserService) SetStatus(actorID, userID uint, status types.UserStatus) error {
	if status != types.UserStatusActive && status != types.UserStatusDisabled {
		return fmt.Errorf("status must be active or disabled: %w", apierrors.ErrInvalidInput)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := requireSystemAdmin(tx, actorID); err != nil {
			return err
		}
		var account model.User
		if err := tx.First(&account, userID).Error; err != nil {
			return mapNotFound("user", err)
		}
		if status == types.UserStatusDisabled && account.Role == types.UserRoleSystemAdmin && account.Status == types.UserStatusActive {
			if err := ensureAnotherActiveSystemAdmin(tx, account.ID); err != nil {
				return err
			}
		}
		if err := tx.Model(&account).Update("status", status).Error; err != nil {
			return fmt.Errorf("update user status: %w", err)
		}
		if status == types.UserStatusDisabled {
			now := time.Now().UTC()
			if err := tx.Model(&model.UserSession{}).Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", now).Error; err != nil {
				return fmt.Errorf("revoke user sessions: %w", err)
			}
			if err := tx.Model(&model.DeviceToken{}).Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", now).Error; err != nil {
				return fmt.Errorf("revoke device tokens: %w", err)
			}
		}
		return nil
	})
}

func (s *UserService) UpdateRole(actorID, userID uint, role types.UserRole) error {
	_, err := s.UpdateUser(actorID, userID, UpdateUserInput{Role: &role})
	return err
}

func (s *UserService) UpdateDisplayName(actorID, userID uint, displayName string) error {
	_, err := s.UpdateUser(actorID, userID, UpdateUserInput{DisplayName: &displayName})
	return err
}

// UpdateUser validates and applies all requested profile and role changes in a
// single transaction so a rejected role change cannot leave a partial update.
func (s *UserService) UpdateUser(actorID, userID uint, input UpdateUserInput) (*model.User, error) {
	var updated model.User
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := requireSystemAdmin(tx, actorID); err != nil {
			return err
		}
		var account model.User
		if err := tx.First(&account, userID).Error; err != nil {
			return mapNotFound("user", err)
		}
		updates := map[string]any{}
		if input.DisplayName != nil {
			displayName := strings.TrimSpace(*input.DisplayName)
			if displayName == "" {
				return fmt.Errorf("display name is required: %w", apierrors.ErrInvalidInput)
			}
			updates["display_name"] = displayName
		}
		if input.Role != nil {
			if !types.IsValidUserRole(*input.Role) {
				return fmt.Errorf("invalid user role: %w", apierrors.ErrInvalidInput)
			}
			if account.Role == types.UserRoleSystemAdmin && *input.Role != types.UserRoleSystemAdmin && account.Status == types.UserStatusActive {
				if err := ensureAnotherActiveSystemAdmin(tx, account.ID); err != nil {
					return err
				}
			}
			updates["role"] = *input.Role
		}
		if len(updates) == 0 {
			return fmt.Errorf("nothing to update: %w", apierrors.ErrInvalidInput)
		}
		if err := tx.Model(&account).Updates(updates).Error; err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		return tx.First(&updated, account.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	var account model.User
	if err := s.db.First(&account, id).Error; err != nil {
		return nil, mapNotFound("user", err)
	}
	return &account, nil
}

func (s *UserService) GetUserByUsername(username string) (*model.User, error) {
	var account model.User
	if err := s.db.Where("username = ?", strings.TrimSpace(username)).First(&account).Error; err != nil {
		return nil, mapNotFound("user", err)
	}
	return &account, nil
}

func (s *UserService) ListUsers() ([]model.User, error) {
	var accounts []model.User
	if err := s.db.Order("username").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return accounts, nil
}

func validatePassword(password string) error {
	if len(password) < minimumPasswordLength {
		return fmt.Errorf("password must contain at least %d characters: %w", minimumPasswordLength, apierrors.ErrInvalidInput)
	}
	return nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func requireSystemAdmin(tx *gorm.DB, userID uint) error {
	var actor model.User
	if err := tx.First(&actor, userID).Error; err != nil {
		return mapNotFound("actor", err)
	}
	if actor.Status != types.UserStatusActive || actor.Role != types.UserRoleSystemAdmin {
		return fmt.Errorf("system administrator required: %w", apierrors.ErrInvalidInput)
	}
	return nil
}

func ensureAnotherActiveSystemAdmin(tx *gorm.DB, excludedID uint) error {
	var count int64
	if err := tx.Model(&model.User{}).
		Where("role = ? AND status = ? AND id <> ?", types.UserRoleSystemAdmin, types.UserStatusActive, excludedID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("count system administrators: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("cannot disable or demote the last active system administrator: %w", apierrors.ErrInvalidInput)
	}
	return nil
}

func mapNotFound(resource string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%s not found: %w", resource, apierrors.ErrNotFound)
	}
	return fmt.Errorf("load %s: %w", resource, err)
}

// UserCallStatView is retained as a read model until the observability phase
// replaces the current daily aggregate table with immutable call events.
type UserCallStatView struct {
	Username   string `json:"username"`
	ServerName string `json:"server_name"`
	Date       string `json:"date"`
	Count      uint64 `json:"count"`
}

func (s *UserService) ListCallStats() ([]UserCallStatView, error) {
	var rows []UserCallStatView
	err := s.db.Table("user_call_stats").
		Select("users.username AS username, user_call_stats.server_name AS server_name, user_call_stats.date AS date, user_call_stats.count AS count").
		Joins("LEFT JOIN users ON users.id = user_call_stats.user_id").
		Where("users.deleted_at IS NULL").
		Order("user_call_stats.date DESC, users.username, user_call_stats.server_name").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load call stats: %w", err)
	}
	return rows, nil
}
