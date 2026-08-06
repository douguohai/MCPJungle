package model

import "gorm.io/gorm"

// McpServiceManager links a user to an MCP server as a responsible person
// (primary or backup). Design doc §5.2.
type McpServiceManager struct {
	gorm.Model

	McpServerID uint   `gorm:"not null;uniqueIndex:idx_mcp_service_manager,priority:1"`
	UserID      uint   `gorm:"not null;uniqueIndex:idx_mcp_service_manager,priority:2"`
	RoleType    string `gorm:"type:varchar(20);not null"` // primary | backup
}
