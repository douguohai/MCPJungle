package model

import (
	"time"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// DeviceToken stores a personal device credential. The raw token is never
// persisted; only a short prefix plus a SHA-256 hash are stored.
type DeviceToken struct {
	gorm.Model

	UserID uint `json:"user_id" gorm:"not null;index;uniqueIndex:idx_user_device_name,priority:1"`

	Name string `json:"name" gorm:"type:varchar(191);not null;uniqueIndex:idx_user_device_name,priority:2"`

	TokenPrefix string `json:"token_prefix" gorm:"type:varchar(32);not null;index"`
	TokenHash   string `json:"-" gorm:"type:char(64);not null;uniqueIndex"`

	Scope types.DeviceTokenScope `json:"scope" gorm:"type:varchar(20);not null"`

	ExpiresAt  time.Time  `json:"expires_at" gorm:"not null"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	LastUsedIP string     `json:"last_used_ip,omitempty" gorm:"type:varchar(45)"`
	ClientInfo string     `json:"client_info,omitempty" gorm:"type:varchar(255)"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

// DeviceTokenService narrows a restricted token to a subset of the user's
// effective services.
type DeviceTokenService struct {
	DeviceTokenID uint `json:"device_token_id" gorm:"primaryKey;not null"`
	McpServerID   uint `json:"mcp_server_id" gorm:"primaryKey;not null"`
}
