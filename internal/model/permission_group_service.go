package model

import (
	"time"

	"gorm.io/gorm"
)

// PermissionGroupService grants a permission group access to a single MCP
// server. McpServerID references model.McpServer.ID.
type PermissionGroupService struct {
	gorm.Model

	PermissionGroupID uint `gorm:"not null;uniqueIndex:idx_permission_group_service,priority:1"`
	McpServerID       uint `gorm:"not null;uniqueIndex:idx_permission_group_service,priority:2"`
	AssignedByID      uint `gorm:"not null"`
	AssignedAt        time.Time `gorm:"not null"`
}
