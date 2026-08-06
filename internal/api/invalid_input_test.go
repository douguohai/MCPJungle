package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/toolcollection"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func setupInvalidInputServer(t *testing.T) *Server {
	t.Helper()

	setup := testhelpers.SetupTestDB(t)
	t.Cleanup(setup.Cleanup)

	mcpProxy := mcpserver.NewMCPServer("test", "0.0.1")
	sseMcpProxy := mcpserver.NewMCPServer("test-sse", "0.0.1")
	mcpSvc, err := mcp.NewMCPService(&mcp.ServiceConfig{
		DB:                      setup.DB,
		McpProxyServer:          mcpProxy,
		SseMcpProxyServer:       sseMcpProxy,
		Metrics:                 telemetry.NewNoopCustomMetrics(),
		McpServerInitReqTimeout: 5,
	})
	if err != nil {
		t.Fatalf("failed to create MCP service: %v", err)
	}

	tcSvc, err := toolcollection.NewToolCollectionService(setup.DB, mcpSvc)
	if err != nil {
		t.Fatalf("failed to create tool collection service: %v", err)
	}

	return &Server{
		mcpService:            mcpSvc,
		toolCollectionService: tcSvc,
		userService:           user.NewUserService(setup.DB),
	}
}

func TestGetToolHandler_InvalidCanonicalNameReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := setupInvalidInputServer(t)

	router := gin.New()
	router.GET("/tool", s.getToolHandler())

	req := httptest.NewRequest(http.MethodGet, "/tool?name=invalid-name", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusBadRequest, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "does not contain a __ separator")
}

func TestGetPromptHandler_InvalidCanonicalNameReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := setupInvalidInputServer(t)

	router := gin.New()
	router.GET("/prompt", s.getPromptHandler())

	req := httptest.NewRequest(http.MethodGet, "/prompt?name=invalid-name", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusBadRequest, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "does not contain a __ separator")
}

func TestCreateToolCollectionHandler_InvalidNameReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := setupInvalidInputServer(t)

	router := gin.New()
	router.POST("/collections", s.createToolCollectionHandler())

	req := httptest.NewRequest(http.MethodPost, "/collections",
		strings.NewReader(`{"name":"-bad-collection","included_tools":["test__tool"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusBadRequest, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "invalid collection name")
}
