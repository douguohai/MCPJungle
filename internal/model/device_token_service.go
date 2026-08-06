package model

import (
	"gorm.io/gorm"
)

// DeviceTokenService records a single MCP service that a restricted-scope
// DeviceToken may access. Only used when the token's ScopeMode is
// DeviceTokenScopeRestricted; inherit_all tokens derive their scope entirely
// from the owning user's effective services.
type DeviceTokenService struct {
	gorm.Model

	DeviceTokenID uint `gorm:"not null;uniqueIndex:idx_device_token_service,priority:1"`
	McpServerID   uint `gorm:"not null;uniqueIndex:idx_device_token_service,priority:2"`
}
