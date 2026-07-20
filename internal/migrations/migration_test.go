package migrations

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestMigrateRemovesLegacyIdentitySchema(t *testing.T) {
	db := testhelpers.RequireTestDB(t)
	userTable := db.NamingStrategy.TableName("User")
	clientTable := db.NamingStrategy.TableName("McpClient")

	if err := db.Exec("CREATE TABLE `" + userTable + "` (`id` bigint unsigned AUTO_INCREMENT PRIMARY KEY, `username` varchar(255) NOT NULL, `access_token` varchar(255) NOT NULL, `allowed_servers` json NULL)").Error; err != nil {
		t.Fatalf("create legacy users table: %v", err)
	}
	if err := db.Exec("CREATE TABLE `" + clientTable + "` (`id` bigint unsigned AUTO_INCREMENT PRIMARY KEY, `name` varchar(255) NOT NULL)").Error; err != nil {
		t.Fatalf("create legacy MCP clients table: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if db.Migrator().HasColumn(&model.User{}, "access_token") {
		t.Fatal("legacy users.access_token column still exists")
	}
	if db.Migrator().HasColumn(&model.User{}, "allowed_servers") {
		t.Fatal("legacy users.allowed_servers column still exists")
	}
	if db.Migrator().HasTable(clientTable) {
		t.Fatal("legacy mcp_clients table still exists")
	}
	if !db.Migrator().HasTable(&model.User{}) {
		t.Fatal("current users table was not recreated")
	}
}
