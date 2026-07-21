package model

import (
	"fmt"

	"gorm.io/gorm"
)

type ServerMode string

const (
	// ModeDev is ideal for developers running the mcpjungle locally for personal MCP workflows.
	ModeDev ServerMode = "development"

	// ModeEnterprise is ideal for team deployments where multiple users will be using mcpjungle.
	ModeEnterprise ServerMode = "enterprise"
)

// IsEnterpriseMode returns true only for the authenticated team mode.
func IsEnterpriseMode(mode ServerMode) bool {
	return mode == ModeEnterprise
}

// ServerConfig represents the configuration for the MCPJungle server.
type ServerConfig struct {
	gorm.Model

	Mode ServerMode `gorm:"type:varchar(12);not null"`

	// Initialized indicates whether the server has been initialized.
	// If this is set to false, the server is not yet ready for use and all requests to it should be rejected.
	Initialized bool `gorm:"not null;default:false"`

	// SystemDisplayName is the operator-facing product name shown in the dashboard.
	// It is intentionally presentation-only and does not affect API routing or MCP identity.
	SystemDisplayName string `gorm:"type:varchar(64);not null;default:'MCPJungle'"`
}

func (c *ServerConfig) BeforeSave(tx *gorm.DB) (err error) {
	// Make sure that the server mode is valid before saving
	switch c.Mode {
	case ModeDev:
		// valid
	case ModeEnterprise:
		// valid
	default:
		return fmt.Errorf("invalid server mode: %s", c.Mode)
	}
	return nil
}
