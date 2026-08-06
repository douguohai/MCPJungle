// Package testhelpers provides common testing utilities and assertion functions
// for the MCPJungle project.
package testhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MysqlTestDSNEnvVar names the environment variable that provides a live MySQL
// DSN for database-backed tests, in the form mysql://user:pass@host:port/db.
// MySQL has no in-memory mode, so when unset, database tests are skipped. The
// database name in the URL is ignored — each test gets its own isolated
// database. The account needs CREATE/DROP DATABASE privilege.
const MysqlTestDSNEnvVar = "TEST_MYSQL_DSN"

// CreateTestDB connects to the MySQL server described by TEST_MYSQL_DSN,
// creates a uniquely-named isolated database, and returns a *gorm.DB bound to
// it. The per-test database is dropped automatically via t.Cleanup when the
// test completes, so callers do not need to drop it themselves.
//
// TEST_MYSQL_DSN must be a mysql://user:pass@host:port/db URL. When unset, the
// calling test is skipped.
func CreateTestDB(t *testing.T) (*gorm.DB, error) {
	t.Helper()

	dsn := os.Getenv(MysqlTestDSNEnvVar)
	if dsn == "" {
		t.Skipf("%s not set, skipping database test", MysqlTestDSNEnvVar)
	}

	user, pass, host, port, err := parseMySQLURL(dsn)
	if err != nil {
		return nil, err
	}

	// Connect against the built-in `mysql` system database so we can CREATE/DROP
	// the per-test database.
	adminDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql?charset=utf8mb4&parseTime=True&loc=Local",
		url.QueryEscape(user), url.QueryEscape(pass), host, port)
	adminDB, err := gorm.Open(mysql.Open(adminDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mysql server: %w", err)
	}

	testDBName := fmt.Sprintf("mcpjungle_test_%d_%d", time.Now().UnixNano(), rand.Intn(1000000))
	createSQL := fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci", testDBName)
	if err := adminDB.Exec(createSQL).Error; err != nil {
		closeQuietly(adminDB)
		return nil, fmt.Errorf("failed to create test database %q (does the account have CREATE DATABASE privilege?): %w", testDBName, err)
	}

	testDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&sql_mode=ALLOW_INVALID_DATES",
		url.QueryEscape(user), url.QueryEscape(pass), host, port, testDBName)
	testDB, err := gorm.Open(mysql.Open(testDSN), &gorm.Config{})
	if err != nil {
		_ = adminDB.Exec(fmt.Sprintf("DROP DATABASE `%s`", testDBName))
		closeQuietly(adminDB)
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	t.Cleanup(func() {
		closeQuietly(testDB)
		_ = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName))
		closeQuietly(adminDB)
	})

	return testDB, nil
}

// parseMySQLURL extracts the user/password/host/port from a mysql:// DSN.
func parseMySQLURL(dsn string) (user, pass, host, port string, err error) {
	u, perr := url.Parse(dsn)
	if perr != nil || u.User == nil || u.Host == "" {
		return "", "", "", "", fmt.Errorf("invalid %s %q: must be mysql://user:pass@host:port/db", MysqlTestDSNEnvVar, dsn)
	}
	user = u.User.Username()
	pass, _ = u.User.Password()
	host = u.Hostname()
	port = u.Port()
	if port == "" {
		port = "3306"
	}
	return user, pass, host, port, nil
}

// closeQuietly closes a gorm.DB's underlying connection, ignoring errors.
func closeQuietly(db *gorm.DB) {
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// AssertError asserts that an error occurred
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// AssertNoError asserts that no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// AssertNotNil asserts that an object is not nil
func AssertNotNil(t *testing.T, obj any) {
	t.Helper()
	if obj == nil {
		t.Error("Expected not nil, got nil")
	}
}

// AssertMCPServerInfo asserts the MCP initialize response advertises the
// expected server name and version.
func AssertMCPServerInfo(t *testing.T, srv *server.MCPServer, expectedName, expectedVersion string) {
	t.Helper()

	message := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(int64(1)),
		Request: mcp.Request{
			Method: string(mcp.MethodInitialize),
		},
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("failed to marshal initialize request: %v", err)
	}

	response := srv.HandleMessage(context.Background(), messageBytes)
	if response == nil {
		t.Fatal("expected initialize response, got nil")
	}

	resp, ok := response.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("expected JSONRPCResponse, got %T", response)
	}

	initResult, ok := resp.Result.(mcp.InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", resp.Result)
	}

	if initResult.ServerInfo.Name != expectedName {
		t.Fatalf("expected server name %q, got %q", expectedName, initResult.ServerInfo.Name)
	}
	if initResult.ServerInfo.Version != expectedVersion {
		t.Fatalf("expected server version %q, got %q", expectedVersion, initResult.ServerInfo.Version)
	}
}

// AssertEqual asserts that two values are equal
func AssertEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertTrue asserts that a condition is true
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Error(message)
	}
}

// AssertFalse asserts that a condition is false
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Error(message)
	}
}

// AssertStringContains asserts that a string contains a substring
func AssertStringContains(t *testing.T, str, substr string) {
	t.Helper()
	if !Contains(str, substr) {
		t.Errorf("Expected string '%s' to contain '%s'", str, substr)
	}
}

// AssertStringNotContains asserts that a string does not contain a substring
func AssertStringNotContains(t *testing.T, str, substr string) {
	t.Helper()
	if Contains(str, substr) {
		t.Errorf("Expected string '%s' to not contain '%s'", str, substr)
	}
}

// AssertSliceLength asserts that a slice has the expected length
func AssertSliceLength(t *testing.T, slice any, expectedLength int) {
	t.Helper()
	switch v := slice.(type) {
	case []any:
		if len(v) != expectedLength {
			t.Errorf("Expected slice length %d, got %d", expectedLength, len(v))
		}
	case []string:
		if len(v) != expectedLength {
			t.Errorf("Expected slice length %d, got %d", expectedLength, len(v))
		}
	case []int:
		if len(v) != expectedLength {
			t.Errorf("Expected slice length %d, got %d", expectedLength, len(v))
		}
	default:
		t.Error("Unsupported slice type for length assertion")
	}
}

// AssertMapContainsKey asserts that a map contains a specific key
func AssertMapContainsKey(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if _, exists := m[key]; !exists {
		t.Errorf("Expected map to contain key '%s'", key)
	}
}

// AssertMapNotContainsKey asserts that a map does not contain a specific key
func AssertMapNotContainsKey(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if _, exists := m[key]; exists {
		t.Errorf("Expected map to not contain key '%s'", key)
	}
}

// AssertPanic asserts that a function panics
func AssertPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected function to panic")
		}
	}()
	fn()
}

// AssertNoPanic asserts that a function does not panic
func AssertNoPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected no panic, but got: %v", r)
		}
	}()
	fn()
}

// CreateTestTable creates a test table for table-driven tests
func CreateTestTable[T any](tests []T) []T {
	return tests
}

// RunTableTests runs table-driven tests
func RunTableTests[T any](t *testing.T, tests []T, testFunc func(t *testing.T, test T)) {
	for i, tt := range tests {
		t.Run(testName(i, tt), func(t *testing.T) {
			testFunc(t, tt)
		})
	}
}

// Helper function to generate test names
func testName(index int, test any) string {
	if named, ok := test.(interface{ Name() string }); ok {
		return named.Name()
	}
	return fmt.Sprintf("test_%d", index)
}

// Contains checks if a string contains a substring
// This is a public function that can be used by other packages
func Contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

// containsSubstring is a helper function for Contains
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// FormatError formats error messages consistently
func FormatError(expected, actual any) string {
	return fmt.Sprintf("Expected %v, got %v", expected, actual)
}

// FormatSliceError formats slice error messages
func FormatSliceError(expected, actual any) string {
	return fmt.Sprintf("Expected %v, got %v", expected, actual)
}

// FormatMapError formats map error messages
func FormatMapError(expected, actual any) string {
	return fmt.Sprintf("Expected %v, got %v", expected, actual)
}

// TestDBSetup represents a test database setup with common models
type TestDBSetup struct {
	DB *gorm.DB
}

// SetupTestDB creates a test database with all common models migrated
func SetupTestDB(t *testing.T) *TestDBSetup {
	t.Helper()

	db, err := CreateTestDB(t)
	AssertNoError(t, err)

	// Migrate all common models
	err = db.AutoMigrate(
		&model.User{},
		&model.McpServer{},
		&model.Tool{},
		&model.ServerConfig{},
		&model.ToolGroup{},
		&model.Prompt{},
		&model.Resource{},
		&model.UpstreamOAuthPendingSession{},
		&model.UpstreamOAuthToken{},
		&model.UserCallStat{},
		// New identity & authorization models (stage 1).
		&model.UserSession{},
		&model.DeviceToken{},
		&model.DeviceTokenService{},
		&model.PermissionGroup{},
		&model.PermissionGroupMember{},
		&model.PermissionGroupService{},
	)
	AssertNoError(t, err)

	return &TestDBSetup{DB: db}
}

// SetupUserTest creates a test database with user-related models and a basic test user
func SetupUserTest(t *testing.T) (*TestDBSetup, *model.User) {
	t.Helper()

	setup := SetupTestDB(t)

	// Create a basic test user
	testUser := &model.User{
		Username: "testuser",
		Role:     types.UserRoleMember,
	}

	err := setup.DB.Create(testUser).Error
	AssertNoError(t, err)

	return setup, testUser
}

// SetupAdminTest creates a test database with user-related models and a basic admin user
func SetupAdminTest(t *testing.T) (*TestDBSetup, *model.User) {
	t.Helper()

	setup := SetupTestDB(t)

	// Create a basic test admin user
	testAdmin := &model.User{
		Username: "testadmin",
		Role:     types.UserRoleSystemAdmin,
	}

	err := setup.DB.Create(testAdmin).Error
	AssertNoError(t, err)

	return setup, testAdmin
}

// SetupMCPTest creates a test database with MCP-related models
func SetupMCPTest(t *testing.T) *TestDBSetup {
	t.Helper()

	setup := SetupTestDB(t)

	// Additional MCP-specific setup can be added here
	// For example, creating test MCP servers, tools, etc.

	return setup
}

// SetupServerConfigTest creates a test database with server config models
func SetupServerConfigTest(t *testing.T) *TestDBSetup {
	t.Helper()

	setup := SetupTestDB(t)

	// Additional server config setup can be added here

	return setup
}

// CreateTestUser creates a test user with the given parameters
func (s *TestDBSetup) CreateTestUser(username string, role types.UserRole) *model.User {
	user := &model.User{
		Username: username,
		Role:     role,
	}

	err := s.DB.Create(user).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to create test user: %v", err))
	}

	return user
}

// CreateTestMcpServer creates a test MCP server with the given parameters
func (s *TestDBSetup) CreateTestMcpServer(name, description string, transport types.McpServerTransport, config []byte) *model.McpServer {
	server := &model.McpServer{
		Name:        name,
		Description: description,
		Transport:   transport,
		Config:      config,
	}

	err := s.DB.Create(server).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to create test MCP server: %v", err))
	}

	return server
}

// CreateTestTool creates a test tool with the given parameters
func (s *TestDBSetup) CreateTestTool(name, description string, serverID uint, enabled bool, inputSchema []byte) *model.Tool {
	tool := &model.Tool{
		Name:        name,
		Description: description,
		ServerID:    serverID,
		Enabled:     enabled,
		InputSchema: inputSchema,
	}

	err := s.DB.Create(tool).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to create test tool: %v", err))
	}

	return tool
}

// CreateTestServerConfig creates a test server config with the given parameters
func (s *TestDBSetup) CreateTestServerConfig(mode model.ServerMode, initialized bool) *model.ServerConfig {
	config := &model.ServerConfig{
		Mode:        mode,
		Initialized: initialized,
	}

	err := s.DB.Create(config).Error
	if err != nil {
		panic(fmt.Sprintf("Failed to create test server config: %v", err))
	}

	return config
}

// Cleanup closes the database connection
func (s *TestDBSetup) Cleanup() {
	if s.DB != nil {
		if sqlDB, err := s.DB.DB(); err == nil {
			sqlDB.Close()
		}
	}
}

// CommandAnnotationTest represents test data for command annotation testing
type CommandAnnotationTest struct {
	Key      string
	Expected string
}

// TestCommandAnnotations tests command annotations using table-driven approach
func TestCommandAnnotations(t *testing.T, annotations map[string]string, tests []CommandAnnotationTest) {
	t.Helper()

	AssertNotNil(t, annotations)

	for _, tt := range tests {
		t.Run(tt.Key, func(t *testing.T) {
			value, exists := annotations[tt.Key]
			AssertTrue(t, exists, "Missing '"+tt.Key+"' annotation")
			AssertEqual(t, tt.Expected, value)
		})
	}
}

// TestCommandProperties tests basic command properties
func TestCommandProperties(t *testing.T, actualUse, expectedUse, actualShort, expectedShort string) {
	t.Helper()

	AssertEqual(t, expectedUse, actualUse)
	AssertEqual(t, expectedShort, actualShort)
}

// SubcommandTestData represents test data for subcommand testing
type SubcommandTestData struct {
	Use   string
	Short string
	Long  string
}

// TestSubcommandStructure tests a subcommand's basic structure
func TestSubcommandStructure(t *testing.T, actualUse, expectedUse, actualShort, expectedShort, actualLong string) {
	t.Helper()

	AssertEqual(t, expectedUse, actualUse)
	AssertEqual(t, expectedShort, actualShort)
	if actualLong != "" {
		AssertTrue(t, len(actualLong) > 0, "Long description should not be empty")
	}
}

// FlagTestData represents test data for flag testing
type FlagTestData struct {
	Name        string
	Description string
}
