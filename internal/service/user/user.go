// Package user provides user service functionality for the MCPJungle application.
package user

import (
	"errors"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService provides methods to manage users in the MCPJungle system.
type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// CreateAdminUser creates a system admin user with the given username and password.
// The password is bcrypt-hashed before storage; the plaintext is never persisted.
func (u *UserService) CreateAdminUser(username, password string) (*model.User, error) {
	if password == "" {
		return nil, fmt.Errorf("admin password must not be empty: %w", apierrors.ErrInvalidInput)
	}
	if username == "" {
		username = "admin"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash admin password: %w", err)
	}
	user := model.User{
		Username:     username,
		Role:         types.UserRoleSystemAdmin,
		PasswordHash: string(hash),
	}
	if err := u.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}
	return &user, nil
}

// GetUserByUsername returns the user with the given username.
func (u *UserService) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := u.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %w", apierrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

// GetUserByID returns the user with the given primary key.
func (u *UserService) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	if err := u.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %w", apierrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

// VerifyPassword authenticates a user by username+password. It returns
// apierrors.ErrInvalidCredentials for any mismatch (unknown user, unset
// password, or wrong password) so callers cannot distinguish which failed.
func (u *UserService) VerifyPassword(username, password string) (*model.User, error) {
	user, err := u.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return nil, fmt.Errorf("invalid credentials: %w", apierrors.ErrInvalidCredentials)
		}
		return nil, err
	}
	if user.PasswordHash == "" {
		// legacy/migrated user has not set a password yet
		return nil, fmt.Errorf("invalid credentials: %w", apierrors.ErrInvalidCredentials)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", apierrors.ErrInvalidCredentials)
	}
	return user, nil
}

// SetPasswordHash is a helper that bcrypt-hashes a plaintext password.
func SetPasswordHash(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password must not be empty: %w", apierrors.ErrInvalidInput)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CreateUser creates a new member user with the specified username.
func (u *UserService) CreateUser(input *model.User) (*model.User, error) {
	user := model.User{
		Username: input.Username,
		Role:     types.UserRoleMember,
	}
	if err := u.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// ListUsers retrieves all users from the database.
func (u *UserService) ListUsers() ([]model.User, error) {
	var users []model.User
	if err := u.db.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

// DeleteUser removes a user with the specified username from the database.
// If a user's role is system_admin, the deletion will be rejected.
func (u *UserService) DeleteUser(username string) error {
	var user model.User
	err := u.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user with username %s not found: %w", username, apierrors.ErrNotFound)
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	if user.Role == types.UserRoleSystemAdmin {
		return fmt.Errorf("cannot delete a system admin user: %w", apierrors.ErrInvalidInput)
	}

	err = u.db.Unscoped().Where("username = ?", username).Delete(&model.User{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// UserCallStatView is one row of per-user per-server call counts for the stats page.
type UserCallStatView struct {
	Username   string `json:"username"`
	ServerName string `json:"server_name"`
	Date       string `json:"date"`
	Count      uint64 `json:"count"`
}

// ListCallStats returns per-user per-server per-day MCP call counts, joined with usernames.
func (u *UserService) ListCallStats() ([]UserCallStatView, error) {
	var rows []UserCallStatView
	err := u.db.Table("user_call_stats").
		Select("users.username AS username, user_call_stats.server_name AS server_name, user_call_stats.date AS date, user_call_stats.count AS count").
		Joins("LEFT JOIN users ON users.id = user_call_stats.user_id").
		Where("users.deleted_at IS NULL").
		Order("user_call_stats.date DESC, users.username, user_call_stats.server_name").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load call stats: %w", err)
	}
	return rows, nil
}
