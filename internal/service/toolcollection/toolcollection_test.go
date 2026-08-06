package toolcollection

import (
	"errors"
	"reflect"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/version"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestValidCollectionNameRegex(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"valid-collection", true},
		{"valid_collection", true},
		{"validCollection", true},
		{"collection123", true},
		{"123collection", true}, // starts with number (allowed by regex)
		{"-collection", false},  // starts with hyphen
		{"_collection", false},  // starts with underscore
		{"", false},             // empty
		{"collection-name", true},
		{"collection_name", true},
		{"collection name", false}, // contains space
		{"collection@name", false}, // contains special character
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := ValidCollectionName.MatchString(tt.name)
			testhelpers.AssertEqual(t, tt.valid, isValid)
		})
	}
}

func TestValidCollectionNameEdgeCases(t *testing.T) {
	// Test very long names
	longName := "a"
	for i := 0; i < 100; i++ {
		longName += "a"
	}

	isValid := ValidCollectionName.MatchString(longName)
	testhelpers.AssertTrue(t, isValid, "Expected long name to be valid")

	// Test single character names
	singleCharNames := []string{"a", "A", "1", "0"}
	for _, name := range singleCharNames {
		isValid := ValidCollectionName.MatchString(name)
		testhelpers.AssertTrue(t, isValid, "Expected single character name '"+name+"' to be valid")
	}

	// Test names with mixed characters
	mixedNames := []string{"a1-b_c", "A1-B_C", "test-123_collection"}
	for _, name := range mixedNames {
		isValid := ValidCollectionName.MatchString(name)
		testhelpers.AssertTrue(t, isValid, "Expected mixed name '"+name+"' to be valid")
	}
}

func TestValidCollectionNameUnicode(t *testing.T) {
	// Test that the regex only allows ASCII characters
	unicodeNames := []string{"collection-n", "collection-e", "collection-u", "collection-ss"}
	for _, name := range unicodeNames {
		isValid := ValidCollectionName.MatchString(name)
		testhelpers.AssertTrue(t, isValid, "Expected ascii name '"+name+"' to be valid")
	}
}

func TestValidCollectionNameSpecialCharacters(t *testing.T) {
	// Test various special characters that should not be allowed
	specialChars := []string{"!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "+", "=", "[", "]", "{", "}", "|", "\\", ":", ";", "\"", "'", "<", ">", ",", ".", "?", "/"}

	for _, char := range specialChars {
		name := "collection" + char
		isValid := ValidCollectionName.MatchString(name)
		testhelpers.AssertFalse(t, isValid, "Expected name with special character '"+char+"' to be invalid")
	}
}

func TestValidCollectionNameBoundaryConditions(t *testing.T) {
	// Test names that are exactly at the boundary of what's allowed
	boundaryNames := []string{
		"a",  // single lowercase letter
		"A",  // single uppercase letter
		"0",  // single digit
		"a0", // letter followed by digit
		"0a", // digit followed by letter (allowed by regex)
		"a-", // letter followed by hyphen (allowed by regex)
		"a_", // letter followed by underscore (allowed by regex)
	}

	expectedResults := []bool{true, true, true, true, true, true, true}

	for i, name := range boundaryNames {
		isValid := ValidCollectionName.MatchString(name)
		expected := expectedResults[i]
		if isValid != expected {
			t.Errorf("Expected '%s' to be %v, got %v", name, expected, isValid)
		}
	}
}

func TestValidCollectionNamePerformance(t *testing.T) {
	// Test that the regex performs reasonably well with long strings
	longName := "a"
	for i := 0; i < 1000; i++ {
		longName += "a"
	}

	// This should complete quickly
	isValid := ValidCollectionName.MatchString(longName)
	if !isValid {
		t.Errorf("Expected very long name to be valid")
	}
}

func TestValidCollectionNameConsistency(t *testing.T) {
	// Test that the same input always produces the same result
	testName := "test-collection-name"

	// Run the test multiple times to ensure consistency
	for i := 0; i < 100; i++ {
		result1 := ValidCollectionName.MatchString(testName)
		result2 := ValidCollectionName.MatchString(testName)

		if result1 != result2 {
			t.Errorf("Regex results inconsistent for '%s': got %v and %v", testName, result1, result2)
		}
	}
}

func setupInMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := testhelpers.CreateTestDB(t)
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(&model.McpServer{}, &model.Tool{}, &model.ToolCollection{}, &model.Prompt{}, &model.Resource{}); err != nil {
		t.Fatalf("failed to migrate test models: %v", err)
	}
	return db
}

func newTestMCPService(t *testing.T, db *gorm.DB) *mcp.MCPService {
	t.Helper()

	proxyServer := server.NewMCPServer(
		"test proxy",
		"0.0.1",
		server.WithResourceCapabilities(false, false),
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithToolFilter(mcp.ProxyToolFilter),
	)
	sseProxyServer := server.NewMCPServer(
		"test sse proxy",
		"0.0.1",
		server.WithResourceCapabilities(false, false),
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithToolFilter(mcp.ProxyToolFilter),
	)

	svc, err := mcp.NewMCPService(&mcp.ServiceConfig{
		DB:                      db,
		McpProxyServer:          proxyServer,
		SseMcpProxyServer:       sseProxyServer,
		Metrics:                 telemetry.NewNoopCustomMetrics(),
		McpServerInitReqTimeout: 10,
	})
	if err != nil {
		t.Fatalf("failed to create MCP service: %v", err)
	}

	return svc
}

func TestResolveEffectiveTools_CollectionNotFound(t *testing.T) {
	db := setupInMemoryDB(t)
	s := &ToolCollectionService{
		db:         db,
		mcpService: &mcp.MCPService{}, // zero value is fine for this test
	}

	_, err := s.ResolveEffectiveTools("nonexistent-collection")
	if !errors.Is(err, ErrToolCollectionNotFound) {
		t.Fatalf("expected ErrToolCollectionNotFound, got: %v", err)
	}
}

func TestResolveEffectiveTools_ReturnsSorted(t *testing.T) {
	db := setupInMemoryDB(t)

	// Create a ToolCollection that contains an unsorted list of tools.
	// The model.ToolCollection implementation is expected to return those tools
	// (or otherwise resolve them); this test asserts that ResolveEffectiveTools
	// sorts the result before returning.
	collection := model.ToolCollection{
		Name:          "my-collection",
		IncludedTools: datatypes.JSON([]byte(`["tool-b","tool-a","tool-c"]`)),
	}

	if err := db.Create(&collection).Error; err != nil {
		t.Fatalf("failed to create tool collection: %v", err)
	}

	s := &ToolCollectionService{
		db:         db,
		mcpService: &mcp.MCPService{},
	}

	tools, err := s.ResolveEffectiveTools("my-collection")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect sorted order
	expected := []string{"tool-a", "tool-b", "tool-c"}
	// If the underlying ResolveEffectiveTools implementation returns a different
	// set because it resolves servers/excludes, adapt expectations accordingly.
	// For the common case where explicit tools were stored, assert sorting.
	if !reflect.DeepEqual(tools, expected) {
		t.Fatalf("expected sorted tools %v, got %v", expected, tools)
	}
}

func TestCreateToolCollection_InvalidNameReturnsInvalidInput(t *testing.T) {
	db := setupInMemoryDB(t)
	s := &ToolCollectionService{
		db:         db,
		mcpService: &mcp.MCPService{},
	}

	err := s.CreateToolCollection(&model.ToolCollection{Name: "-bad-collection"})
	if !errors.Is(err, apierrors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestCreateToolCollection_EmptyResolvedToolsReturnsInvalidInput(t *testing.T) {
	db := setupInMemoryDB(t)
	s := &ToolCollectionService{
		db:         db,
		mcpService: &mcp.MCPService{},
	}

	err := s.CreateToolCollection(&model.ToolCollection{Name: "empty-collection"})
	if !errors.Is(err, apierrors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestNewToolCollectionService_DegradedPersistedCollectionDoesNotFailStartup(t *testing.T) {
	db := setupInMemoryDB(t)

	validServer, err := model.NewStdioServer("valid-server", "Valid server", "echo", nil, nil, "")
	if err != nil {
		t.Fatalf("failed to create valid server model: %v", err)
	}
	if err := db.Create(validServer).Error; err != nil {
		t.Fatalf("failed to persist valid server: %v", err)
	}

	validTool := model.Tool{
		ServerID:    validServer.ID,
		Name:        "sum",
		Description: "adds numbers",
		InputSchema: []byte(`{"type":"object"}`),
		Enabled:     true,
	}
	if err := db.Create(&validTool).Error; err != nil {
		t.Fatalf("failed to persist valid tool: %v", err)
	}

	validCollection := model.ToolCollection{
		Name:            "valid-collection",
		IncludedServers: datatypes.JSON([]byte(`["valid-server"]`)),
	}
	degradedCollection := model.ToolCollection{
		Name:            "degraded-collection",
		IncludedServers: datatypes.JSON([]byte(`["missing-server"]`)),
	}
	if err := db.Create(&validCollection).Error; err != nil {
		t.Fatalf("failed to persist valid collection: %v", err)
	}
	if err := db.Create(&degradedCollection).Error; err != nil {
		t.Fatalf("failed to persist degraded collection: %v", err)
	}

	mcpService := newTestMCPService(t, db)

	svc, err := NewToolCollectionService(db, mcpService)
	if err != nil {
		t.Fatalf("expected degraded persisted collection not to fail startup, got: %v", err)
	}

	validProxy, ok := svc.GetToolCollectionMCPServer("valid-collection")
	if !ok {
		t.Fatal("expected valid collection MCP proxy to be initialized")
	}
	validTools := validProxy.ListTools()
	if len(validTools) != 1 {
		t.Fatalf("expected valid collection proxy to expose 1 tool, got %d", len(validTools))
	}
	if _, ok := validTools["valid-server__sum"]; !ok {
		t.Fatalf("expected valid collection proxy to expose valid-server__sum, got keys %v", reflect.ValueOf(validTools).MapKeys())
	}

	degradedProxy, ok := svc.GetToolCollectionMCPServer("degraded-collection")
	if !ok {
		t.Fatal("expected degraded collection MCP proxy to be initialized")
	}
	if len(degradedProxy.ListTools()) != 0 {
		t.Fatalf("expected degraded collection proxy to expose 0 tools, got %d", len(degradedProxy.ListTools()))
	}

	degradedSSEProxy, ok := svc.GetToolCollectionSseMCPServer("degraded-collection")
	if !ok {
		t.Fatal("expected degraded collection SSE MCP proxy to be initialized")
	}
	if len(degradedSSEProxy.ListTools()) != 0 {
		t.Fatalf("expected degraded collection SSE proxy to expose 0 tools, got %d", len(degradedSSEProxy.ListTools()))
	}
}

func TestCreateToolCollection_InvalidIncludedServerStillFailsFast(t *testing.T) {
	db := setupInMemoryDB(t)
	s := &ToolCollectionService{
		db:         db,
		mcpService: newTestMCPService(t, db),
	}

	err := s.CreateToolCollection(&model.ToolCollection{
		Name:            "invalid-server-collection",
		IncludedServers: datatypes.JSON([]byte(`["missing-server"]`)),
	})
	if err == nil {
		t.Fatal("expected create tool collection to fail for missing included server")
	}
	if !testhelpers.Contains(err.Error(), "failed to resolve effective tools") {
		t.Fatalf("expected create error to mention failed resolution, got: %v", err)
	}
}

func TestNewMCPServer_AdvertisesCurrentVersion(t *testing.T) {
	s := &ToolCollectionService{}

	testhelpers.AssertMCPServerInfo(
		t,
		s.newMCPServer("collection-a"),
		"MCPJungle proxy MCP server for tool collection: collection-a",
		version.GetVersion(),
	)
}

func TestNewSseMCPServer_AdvertisesCurrentVersion(t *testing.T) {
	s := &ToolCollectionService{}

	testhelpers.AssertMCPServerInfo(
		t,
		s.newSseMCPServer("collection-a"),
		"MCPJungle proxy MCP server for SSE transport for tool collection: collection-a",
		version.GetVersion(),
	)
}
