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
		&model.ToolGroup{},
		&model.Prompt{},
		&model.Resource{},
		&model.UpstreamOAuthPendingSession{},
		&model.UpstreamOAuthToken{},
		&model.UserCallStat{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			return fmt.Errorf("auto-migration failed for %T: %v", m, err)
		}
	}
	return nil
}
