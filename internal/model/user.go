// Package model provides data models for the MCPJungle application.
package model

import (
	"time"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// User status constants (design doc §6.2 / §18.2).
const (
	UserStatusPendingActivation = "pending_activation"
	UserStatusActive            = "active"
	UserStatusDisabled          = "disabled"
)

// User represents an authenticated, human user in enterprise mode.
// There are no users if mcpjungle is running in development mode.
type User struct {
	gorm.Model

	Username     string         `json:"username" gorm:"uniqueIndex;not null;type:varchar(255)"`
	Role         types.UserRole `json:"role" gorm:"not null"`

	// PasswordHash stores the bcrypt hash of the user's password and is the
	// primary credential for dashboard login (exchanged for a short-lived session).
	PasswordHash string `json:"-" gorm:"type:varchar(255)"`

	// Status tracks the lifecycle of the user account:
	// pending_activation / active / disabled.
	Status string `json:"status" gorm:"type:varchar(20);not null;default:active"`

	// MustChangePassword forces the user to pick a new password on next login.
	MustChangePassword bool `json:"must_change_password" gorm:"default:false"`

	// LastLoginAt records the time of the most recent successful login.
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}
