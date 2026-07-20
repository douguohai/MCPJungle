// Package migrations provides database migration functionality for the MCPJungle application.
package migrations

import (
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

// Migrate performs the database migration for the application.
func Migrate(db *gorm.DB) error {
	if err := removeLegacyIdentitySchema(db); err != nil {
		return err
	}
	for _, m := range []interface{}{
		&model.User{},
		&model.McpServer{},
		&model.PermissionGroup{},
		&model.PermissionGroupUser{},
		&model.PermissionGroupMcpServer{},
		&model.UserSession{},
		&model.DeviceToken{},
		&model.DeviceTokenService{},
		&model.Tool{},
		&model.ServerConfig{},
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

// removeLegacyIdentitySchema intentionally resets only identity-owned data
// when a pre-Hub users table is detected. Those credentials cannot be mapped
// safely to personal sessions, permission groups, or device tokens. MCP server
// registrations and permission-group service bindings are preserved.
func removeLegacyIdentitySchema(db *gorm.DB) error {
	legacyClientTable := db.NamingStrategy.TableName("McpClient")
	if db.Migrator().HasTable(legacyClientTable) {
		if err := db.Migrator().DropTable(legacyClientTable); err != nil {
			return fmt.Errorf("drop legacy MCP clients table: %w", err)
		}
	}

	legacyUsers := db.Migrator().HasColumn(&model.User{}, "access_token") ||
		db.Migrator().HasColumn(&model.User{}, "allowed_servers")
	if !legacyUsers {
		return nil
	}
	for _, table := range []interface{}{
		&model.DeviceTokenService{},
		&model.DeviceToken{},
		&model.UserSession{},
		&model.PermissionGroupUser{},
		&model.UserCallStat{},
		&model.User{},
	} {
		if !db.Migrator().HasTable(table) {
			continue
		}
		if err := db.Migrator().DropTable(table); err != nil {
			return fmt.Errorf("drop legacy identity table %T: %w", table, err)
		}
	}
	return nil
}
