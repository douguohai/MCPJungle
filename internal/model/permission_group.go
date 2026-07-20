package model

import "gorm.io/gorm"

// PermissionGroup defines a reusable set of MCP services that can be granted
// to many internal users.
type PermissionGroup struct {
	gorm.Model

	Name        string `json:"name" gorm:"type:varchar(191);not null;uniqueIndex"`
	DisplayName string `json:"display_name" gorm:"type:varchar(255);not null"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled" gorm:"not null;default:true"`
}

// PermissionGroupUser links users to permission groups using an explicit
// composite primary key so the same membership cannot be inserted twice.
type PermissionGroupUser struct {
	PermissionGroupID uint `json:"permission_group_id" gorm:"primaryKey;not null"`
	UserID            uint `json:"user_id" gorm:"primaryKey;not null"`
}

// PermissionGroupMcpServer links permission groups to MCP services.
type PermissionGroupMcpServer struct {
	PermissionGroupID uint `json:"permission_group_id" gorm:"primaryKey;not null"`
	McpServerID       uint `json:"mcp_server_id" gorm:"primaryKey;not null"`
}
