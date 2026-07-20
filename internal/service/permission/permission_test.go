package permission_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	projectdb "github.com/mcpjungle/mcpjungle/internal/db"
	perm "github.com/mcpjungle/mcpjungle/internal/service/permission"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type User struct {
	gorm.Model
	Username string
	Status   string `gorm:"type:varchar(32);not null"`
}

type McpServer struct {
	gorm.Model
	Name    string
	Enabled bool `gorm:"not null;default:true"`
}

type PermissionGroup struct {
	gorm.Model
	Name    string
	Enabled bool `gorm:"not null;default:true"`
}

type PermissionGroupUser struct {
	gorm.Model
	PermissionGroupID uint `gorm:"column:permission_group_id;not null"`
	UserID            uint `gorm:"column:user_id;not null"`
}

type PermissionGroupMcpServer struct {
	gorm.Model
	PermissionGroupID uint `gorm:"column:permission_group_id;not null"`
	McpServerID       uint `gorm:"column:mcp_server_id;not null"`
}

func setupPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := requireTestDB(t)
	if err := db.AutoMigrate(
		&User{},
		&McpServer{},
		&PermissionGroup{},
		&PermissionGroupUser{},
		&PermissionGroupMcpServer{},
	); err != nil {
		t.Fatalf("migrate permission test schema: %v", err)
	}
	return db
}

func requireTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	raw := strings.TrimSpace(os.Getenv("MCPJUNGLE_TEST_DATABASE_URL"))
	if raw == "" {
		t.Skip("MCPJUNGLE_TEST_DATABASE_URL is not configured")
	}

	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatalf("generate test table prefix: %v", err)
	}
	prefix := "mcpjungle_test_" + hex.EncodeToString(suffix) + "_"

	baseDB, err := projectdb.NewDBConnection(raw)
	if err != nil {
		t.Fatalf("connect to MySQL test server: %v", err)
	}
	sqlDB, err := baseDB.DB()
	if err != nil {
		t.Fatalf("get MySQL connection pool: %v", err)
	}

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: prefix},
	})
	if err != nil {
		t.Fatalf("create isolated MySQL test namespace: %v", err)
	}

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

func TestEffectiveServiceIDsReturnsUnionForActiveUser(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := perm.NewService(db)

	user := User{Username: "active-user", Status: "active"}
	otherUser := User{Username: "other-user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create active user: %v", err)
	}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}

	server1 := McpServer{Name: "server-1", Enabled: true}
	server2 := McpServer{Name: "server-2", Enabled: true}
	server3 := McpServer{Name: "server-3", Enabled: false}
	server4 := McpServer{Name: "server-4", Enabled: true}
	for _, server := range []*McpServer{&server1, &server2, &server3, &server4} {
		if err := db.Create(server).Error; err != nil {
			t.Fatalf("create server %s: %v", server.Name, err)
		}
	}
	if err := db.Model(&server3).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable server 3: %v", err)
	}

	group1 := PermissionGroup{Name: "group-1", Enabled: true}
	group2 := PermissionGroup{Name: "group-2", Enabled: true}
	group3 := PermissionGroup{Name: "group-3", Enabled: false}
	group4 := PermissionGroup{Name: "group-4", Enabled: true}
	for _, group := range []*PermissionGroup{&group1, &group2, &group3, &group4} {
		if err := db.Create(group).Error; err != nil {
			t.Fatalf("create group %s: %v", group.Name, err)
		}
	}
	if err := db.Model(&group3).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable group 3: %v", err)
	}

	joins := []PermissionGroupUser{
		{PermissionGroupID: group1.ID, UserID: user.ID},
		{PermissionGroupID: group2.ID, UserID: user.ID},
		{PermissionGroupID: group3.ID, UserID: user.ID},
		{PermissionGroupID: group4.ID, UserID: otherUser.ID},
	}
	if err := db.Create(&joins).Error; err != nil {
		t.Fatalf("create group-user joins: %v", err)
	}

	serviceJoins := []PermissionGroupMcpServer{
		{PermissionGroupID: group1.ID, McpServerID: server1.ID},
		{PermissionGroupID: group1.ID, McpServerID: server2.ID},
		{PermissionGroupID: group2.ID, McpServerID: server1.ID},
		{PermissionGroupID: group2.ID, McpServerID: server3.ID},
		{PermissionGroupID: group3.ID, McpServerID: server4.ID},
		{PermissionGroupID: group4.ID, McpServerID: server4.ID},
	}
	if err := db.Create(&serviceJoins).Error; err != nil {
		t.Fatalf("create group-service joins: %v", err)
	}

	got, err := svc.EffectiveServiceIDs(user.ID)
	if err != nil {
		t.Fatalf("EffectiveServiceIDs returned error: %v", err)
	}

	want := map[uint]struct{}{
		server1.ID: {},
		server2.ID: {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EffectiveServiceIDs() = %v, want %v", got, want)
	}
}

func TestEffectiveServiceIDsReturnsEmptyWhenUserHasNoGroups(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := perm.NewService(db)

	user := User{Username: "lonely-user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := svc.EffectiveServiceIDs(user.ID)
	if err != nil {
		t.Fatalf("EffectiveServiceIDs returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("EffectiveServiceIDs() = %v, want empty", got)
	}
}

func TestEffectiveServiceIDsRejectsMissingOrInactiveUser(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := perm.NewService(db)

	inactive := User{Username: "inactive-user", Status: "disabled"}
	if err := db.Create(&inactive).Error; err != nil {
		t.Fatalf("create inactive user: %v", err)
	}

	_, err := svc.EffectiveServiceIDs(inactive.ID)
	if !errors.Is(err, apierrors.ErrNotFound) {
		t.Fatalf("expected inactive user to return ErrNotFound, got %v", err)
	}

	_, err = svc.EffectiveServiceIDs(999999)
	if !errors.Is(err, apierrors.ErrNotFound) {
		t.Fatalf("expected missing user to return ErrNotFound, got %v", err)
	}
}

func TestReplaceUsersReplacesDedupesAndRejectsMissingIDs(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := perm.NewService(db)

	group := PermissionGroup{Name: "group-1", Enabled: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	user1 := User{Username: "user-1", Status: "active"}
	user2 := User{Username: "user-2", Status: "active"}
	user3 := User{Username: "user-3", Status: "disabled"}
	for _, user := range []*User{&user1, &user2, &user3} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Username, err)
		}
	}

	initial := []PermissionGroupUser{
		{PermissionGroupID: group.ID, UserID: user1.ID},
		{PermissionGroupID: group.ID, UserID: user2.ID},
	}
	if err := db.Create(&initial).Error; err != nil {
		t.Fatalf("create initial joins: %v", err)
	}

	if err := svc.ReplaceUsers(group.ID, []uint{user2.ID, user2.ID, user3.ID}); err != nil {
		t.Fatalf("ReplaceUsers returned error: %v", err)
	}

	assertGroupUsers(t, db, group.ID, []uint{user2.ID, user3.ID})

	err := svc.ReplaceUsers(group.ID, []uint{user1.ID, 999999})
	if !errors.Is(err, apierrors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	assertGroupUsers(t, db, group.ID, []uint{user2.ID, user3.ID})
}

func TestReplaceServicesReplacesDedupesAndRejectsMissingIDs(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := perm.NewService(db)

	group := PermissionGroup{Name: "group-1", Enabled: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	server1 := McpServer{Name: "server-1", Enabled: true}
	server2 := McpServer{Name: "server-2", Enabled: false}
	server3 := McpServer{Name: "server-3", Enabled: true}
	for _, server := range []*McpServer{&server1, &server2, &server3} {
		if err := db.Create(server).Error; err != nil {
			t.Fatalf("create server %s: %v", server.Name, err)
		}
	}

	initial := []PermissionGroupMcpServer{
		{PermissionGroupID: group.ID, McpServerID: server1.ID},
		{PermissionGroupID: group.ID, McpServerID: server2.ID},
	}
	if err := db.Create(&initial).Error; err != nil {
		t.Fatalf("create initial joins: %v", err)
	}

	if err := svc.ReplaceServices(group.ID, []uint{server2.ID, server2.ID, server3.ID}); err != nil {
		t.Fatalf("ReplaceServices returned error: %v", err)
	}

	assertGroupServices(t, db, group.ID, []uint{server2.ID, server3.ID})

	err := svc.ReplaceServices(group.ID, []uint{server1.ID, 999999})
	if !errors.Is(err, apierrors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	assertGroupServices(t, db, group.ID, []uint{server2.ID, server3.ID})
}

func TestSetEnabledUpdatesGroupState(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := perm.NewService(db)

	group := PermissionGroup{Name: "group-1", Enabled: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	if err := svc.SetEnabled(group.ID, false); err != nil {
		t.Fatalf("SetEnabled returned error: %v", err)
	}

	var stored PermissionGroup
	if err := db.First(&stored, group.ID).Error; err != nil {
		t.Fatalf("reload group: %v", err)
	}
	if stored.Enabled {
		t.Fatal("expected group to be disabled")
	}
}

func TestGroupMutationsRejectMissingGroup(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := perm.NewService(db)

	if err := svc.ReplaceUsers(999999, []uint{}); !errors.Is(err, apierrors.ErrNotFound) {
		t.Fatalf("ReplaceUsers missing group error = %v, want ErrNotFound", err)
	}
	if err := svc.ReplaceServices(999999, []uint{}); !errors.Is(err, apierrors.ErrNotFound) {
		t.Fatalf("ReplaceServices missing group error = %v, want ErrNotFound", err)
	}
	if err := svc.SetEnabled(999999, true); !errors.Is(err, apierrors.ErrNotFound) {
		t.Fatalf("SetEnabled missing group error = %v, want ErrNotFound", err)
	}
}

func assertGroupUsers(t *testing.T, db *gorm.DB, groupID uint, want []uint) {
	t.Helper()

	var rows []PermissionGroupUser
	if err := db.Where("permission_group_id = ?", groupID).Order("user_id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("load group users: %v", err)
	}

	got := make([]uint, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.UserID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("group users = %v, want %v", got, want)
	}
}

func assertGroupServices(t *testing.T, db *gorm.DB, groupID uint, want []uint) {
	t.Helper()

	var rows []PermissionGroupMcpServer
	if err := db.Where("permission_group_id = ?", groupID).Order("mcp_server_id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("load group services: %v", err)
	}

	got := make([]uint, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.McpServerID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("group services = %v, want %v", got, want)
	}
}
