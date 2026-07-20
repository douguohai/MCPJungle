package model_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	projectdb "github.com/mcpjungle/mcpjungle/internal/db"
	"github.com/mcpjungle/mcpjungle/internal/migrations"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestAccessTokenDigestsAreNeverSerialized(t *testing.T) {
	for name, value := range map[string]any{
		"session": model.UserSession{TokenHash: "session-digest"},
		"device":  model.DeviceToken{TokenHash: "device-digest"},
	} {
		payload, err := json.Marshal(value)
		require.NoError(t, err)
		require.NotContains(t, string(payload), "token_hash", name)
		require.NotContains(t, string(payload), "digest", name)
	}
}

func TestAccessControlMigrationCreatesExpectedTables(t *testing.T) {
	db := requireAccessControlTestDB(t)

	require.NoError(t, migrations.Migrate(db))

	for _, table := range []interface{}{
		&model.User{},
		&model.UserSession{},
		&model.PermissionGroup{},
		&model.PermissionGroupUser{},
		&model.PermissionGroupMcpServer{},
		&model.DeviceToken{},
		&model.DeviceTokenService{},
	} {
		require.Truef(t, db.Migrator().HasTable(table), "expected table for %T", table)
	}
	require.False(t, db.Migrator().HasTable(db.NamingStrategy.TableName("mcp_client")))
}

func TestAccessControlUniqueConstraints(t *testing.T) {
	db := requireAccessControlTestDB(t)
	require.NoError(t, migrations.Migrate(db))

	user := model.User{
		Username:           "alice",
		DisplayName:        "Alice",
		Role:               types.UserRoleMember,
		Status:             types.UserStatusActive,
		PasswordHash:       "hash",
		MustChangePassword: true,
	}
	require.NoError(t, db.Create(&user).Error)

	group := model.PermissionGroup{
		Name:    "group-a",
		Enabled: true,
	}
	require.NoError(t, db.Create(&group).Error)

	server, err := model.NewSSEServer("server-a", "Server A", "https://example.com/sse", "", types.SessionModeStateless)
	require.NoError(t, err)
	require.NoError(t, db.Create(server).Error)

	now := time.Now().UTC()
	session := model.UserSession{
		UserID:     user.ID,
		TokenHash:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ExpiresAt:  time.Now().UTC().Add(8 * time.Hour),
		LastSeenAt: &now,
		IPAddress:  "127.0.0.1",
		UserAgent:  "codex-tests",
	}
	require.NoError(t, db.Create(&session).Error)

	token := model.DeviceToken{
		UserID:      user.ID,
		Name:        "Alice Laptop",
		TokenPrefix: "mcpdt_abcd",
		TokenHash:   "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		Scope:       types.DeviceTokenScopeInheritAll,
		ExpiresAt:   time.Now().UTC().Add(90 * 24 * time.Hour),
	}
	require.NoError(t, db.Create(&token).Error)

	require.Error(t, db.Create(&model.User{
		Username:     "alice",
		DisplayName:  "Dup",
		Role:         types.UserRoleMember,
		Status:       types.UserStatusActive,
		PasswordHash: "hash2",
	}).Error)

	require.Error(t, db.Create(&model.PermissionGroup{
		Name:    "group-a",
		Enabled: true,
	}).Error)

	require.Error(t, db.Create(&model.UserSession{
		UserID:    user.ID,
		TokenHash: session.TokenHash,
		ExpiresAt: time.Now().UTC().Add(8 * time.Hour),
	}).Error)

	require.Error(t, db.Create(&model.DeviceToken{
		UserID:      user.ID,
		Name:        "Other Device",
		TokenPrefix: "mcpdt_dcba",
		TokenHash:   token.TokenHash,
		Scope:       types.DeviceTokenScopeRestricted,
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
	}).Error)

	linkUser := model.PermissionGroupUser{PermissionGroupID: group.ID, UserID: user.ID}
	require.NoError(t, db.Create(&linkUser).Error)
	require.Error(t, db.Create(&model.PermissionGroupUser{PermissionGroupID: group.ID, UserID: user.ID}).Error)

	linkServer := model.PermissionGroupMcpServer{PermissionGroupID: group.ID, McpServerID: server.ID}
	require.NoError(t, db.Create(&linkServer).Error)
	require.Error(t, db.Create(&model.PermissionGroupMcpServer{PermissionGroupID: group.ID, McpServerID: server.ID}).Error)

	linkToken := model.DeviceTokenService{DeviceTokenID: token.ID, McpServerID: server.ID}
	require.NoError(t, db.Create(&linkToken).Error)
	require.Error(t, db.Create(&model.DeviceTokenService{DeviceTokenID: token.ID, McpServerID: server.ID}).Error)
}

func requireAccessControlTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	raw := strings.TrimSpace(os.Getenv("MCPJUNGLE_TEST_DATABASE_URL"))
	if raw == "" {
		t.Skip("MCPJUNGLE_TEST_DATABASE_URL is not configured")
	}

	suffix := make([]byte, 8)
	_, err := rand.Read(suffix)
	require.NoError(t, err)
	prefix := "mcpjungle_test_" + hex.EncodeToString(suffix) + "_"

	baseDB, err := projectdb.NewDBConnection(raw)
	require.NoError(t, err)

	sqlDB, err := baseDB.DB()
	require.NoError(t, err)

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: prefix},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		ctx := context.Background()
		conn, connErr := sqlDB.Conn(ctx)
		if connErr == nil {
			rows, queryErr := conn.QueryContext(ctx, "SHOW TABLES LIKE ?", prefix+"%")
			var tables []string
			if queryErr == nil {
				for rows.Next() {
					var table string
					if scanErr := rows.Scan(&table); scanErr == nil {
						tables = append(tables, table)
					}
				}
				_ = rows.Close()
			}
			_, _ = conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0")
			for _, table := range tables {
				if strings.HasPrefix(table, prefix) {
					_, _ = conn.ExecContext(ctx, "DROP TABLE IF EXISTS `"+table+"`")
				}
			}
			_, _ = conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 1")
			_ = conn.Close()
		}
		_ = sqlDB.Close()
	})

	return db
}
