package model

import (
	"time"

	"gorm.io/gorm"
)

// PermissionGroupService grants a permission group access to a single MCP
// server. McpServerID references model.McpServer.ID.
type PermissionGroupService struct {
	gorm.Model

	// Override gorm.Model fields with json tags for API serialization.
	ID        uint `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	PermissionGroupID uint `json:"permission_group_id" gorm:"not null;uniqueIndex:idx_permission_group_service,priority:1"`
	McpServerID       uint `json:"mcp_server_id" gorm:"not null;uniqueIndex:idx_permission_group_service,priority:2"`
	AssignedByID      uint `json:"assigned_by_id" gorm:"not null"`
	AssignedAt        time.Time `json:"assigned_at" gorm:"not null"`
}
