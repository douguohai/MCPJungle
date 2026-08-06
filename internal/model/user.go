// Package model provides data models for the MCPJungle application.
package model

import (
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
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
}
