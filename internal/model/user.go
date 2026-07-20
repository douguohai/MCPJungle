// Package model provides data models for the MCPJungle application.
package model

import (
	"time"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// User represents a human internal account. Access to MCP services is derived
// from active permission groups instead of per-user allow lists.
type User struct {
	gorm.Model

	Username    string `json:"username" gorm:"type:varchar(191);not null;uniqueIndex"`
	DisplayName string `json:"display_name" gorm:"type:varchar(255);not null"`

	Role   types.UserRole   `json:"role" gorm:"type:varchar(32);not null"`
	Status types.UserStatus `json:"status" gorm:"type:varchar(20);not null;index"`

	PasswordHash string `json:"-" gorm:"type:varchar(255);not null"`

	MustChangePassword bool       `json:"must_change_password" gorm:"not null;default:false"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
}
