// Package user provides user service functionality for the MCPJungle application.
package user

import (
	"errors"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal"
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

// CreateAdminUser creates an admin user with the given username and password.
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
		Role:         types.UserRoleAdmin,
		PasswordHash: string(hash),
	}
	if err := u.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}
	return &user, nil
}

// GetUserByAccessToken returns a user associated with the provided (legacy)
// access token. Retained for backward compatibility during the migration to
// password+JWT auth; new logins go through VerifyPassword.
func (u *UserService) GetUserByAccessToken(token string) (*model.User, error) {
	var user model.User
	if err := u.db.Where("access_token = ?", token).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %w", apierrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to verify token: %w", err)
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

// CreateUser creates a new user with the specified username.
// This method currently only supports creating a standard user, ie, user with the "user" role.
func (u *UserService) CreateUser(input *model.User) (*model.User, error) {
	user := model.User{
		Username: input.Username,
		Role:     types.UserRoleUser,
	}
	if input.AccessToken == "" {
		// no custom access token provided, generate a new one
		token, err := internal.GenerateAccessToken()
		if err != nil {
			return nil, err
		}
		user.AccessToken = token
	} else {
		// validate the user-provided custom access token
		if err := internal.ValidateAccessToken(input.AccessToken); err != nil {
			return nil, fmt.Errorf("invalid access token: %v: %w", err, apierrors.ErrInvalidInput)
		}
		user.AccessToken = input.AccessToken
	}
	if err := u.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// UpdateUser updates an existing user's information based on the provided input.
// Currently it only supports updating the user's access token.
func (u *UserService) UpdateUser(input *model.User) (*model.User, error) {
	var user model.User
	err := u.db.Where("username = ?", input.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user with username %s not found: %w", input.Username, apierrors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if input.AccessToken == "" && input.AllowedServers == nil {
		return nil, fmt.Errorf("nothing to update: %w", apierrors.ErrInvalidInput)
	}
	// validate the user-provided custom access token (if supplied)
	if input.AccessToken != "" {
		if err := internal.ValidateAccessToken(input.AccessToken); err != nil {
			return nil, fmt.Errorf("invalid access token: %v: %w", err, apierrors.ErrInvalidInput)
		}
		user.AccessToken = input.AccessToken
	}
	if input.AllowedServers != nil {
		user.AllowedServers = input.AllowedServers
	}

	err = u.db.Save(&user).Error
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
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
// If a user's role is admin, the deletion will be rejected.
func (u *UserService) DeleteUser(username string) error {
	var user model.User
	err := u.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user with username %s not found: %w", username, apierrors.ErrNotFound)
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	if user.Role == types.UserRoleAdmin {
		return fmt.Errorf("cannot delete an admin user: %w", apierrors.ErrInvalidInput)
	}

	err = u.db.Unscoped().Where("username = ?", username).Delete(&model.User{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
