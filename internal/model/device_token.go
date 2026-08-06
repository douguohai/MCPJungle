package model

import (
	"time"

	"gorm.io/gorm"
)

// DeviceToken scope modes.
const (
	DeviceTokenScopeInheritAll = "inherit_all" // effective services = user's effective services
	DeviceTokenScopeRestricted = "restricted"  // effective services = user's services ∩ token services
)

// DeviceToken status values.
const (
	DeviceTokenStatusActive  = "active"
	DeviceTokenStatusRevoked = "revoked"
	DeviceTokenStatusExpired = "expired"
)

// DeviceToken is a per-device credential used to authenticate MCP proxy
// requests. It replaces the legacy McpClient machine identity.
//
// The raw token has the form `mcpdt_<id>_<secret>`. Only TokenPrefix
// (`mcpdt_<id>`) and the SHA-256 of <secret> (TokenHash) are persisted, so a
// leaked database cannot be used to call MCP. The raw secret is returned to the
// user exactly once at creation time.
type DeviceToken struct {
	gorm.Model

	UserID uint `gorm:"not null;index"`

	// Name is a user-readable label, e.g. "Cursor-办公电脑".
	Name string `gorm:"type:varchar(255);not null"`

	// TokenPrefix is the `mcpdt_<id>` portion used to locate the row on the hot
	// path before the constant-time hash comparison.
	TokenPrefix string `gorm:"type:varchar(64);index;not null"`

	// TokenHash is the hex-encoded SHA-256 of the <secret> portion.
	TokenHash string `gorm:"type:varchar(64);not null"`

	// ScopeMode is DeviceTokenScopeInheritAll or DeviceTokenScopeRestricted.
	ScopeMode string `gorm:"type:varchar(20);not null"`

	// Status is DeviceTokenStatusActive|Revoked|Expired.
	Status string `gorm:"type:varchar(20);not null;default:active"`

	// ExpiresAt is optional; nil means no expiry.
	ExpiresAt *time.Time `gorm:"index"`

	LastUsedAt *time.Time
	LastIP     string `gorm:"type:varchar(64)"`

	// ClientID is an optional client-provided identifier (e.g. "claude-desktop").
	ClientID string `gorm:"type:varchar(255)"`

	RevokedAt *time.Time `gorm:"index"`
}
