// Package migrations provides database migration functionality for the MCPJungle application.
package migrations

import (
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

// Migrate performs the database migration for the application.
func Migrate(db *gorm.DB) error {
	for _, m := range []interface{}{
		&model.McpServer{},
		&model.Tool{},
		&model.ServerConfig{},
		&model.User{},
		&model.UserSession{},
		&model.DeviceToken{},
		&model.DeviceTokenService{},
		&model.PermissionGroup{},
		&model.PermissionGroupMember{},
		&model.PermissionGroupService{},
		&model.ToolCollection{},
		&model.Prompt{},
		&model.Resource{},
		&model.UpstreamOAuthPendingSession{},
		&model.UpstreamOAuthToken{},
		&model.UserCallStat{},
		&model.McpServiceManager{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			return fmt.Errorf("auto-migration failed for %T: %v", m, err)
		}
	}
	// AutoMigrate drops the legacy 'enabled' column from mcp_servers because
	// the McpServer struct no longer carries it. Status is now the single
	// source of truth for server lifecycle state.

	return nil
}
