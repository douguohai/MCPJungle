package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	mcpserver "github.com/mark3labs/mcp-go/server"
	mcpSvc "github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/toolcollection"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

// setupToolCollectionServer creates a Server with a real ToolCollectionService backed by an in-memory DB.
func setupToolCollectionServer(t *testing.T) *Server {
	t.Helper()
	setup := testhelpers.SetupTestDB(t)
	t.Cleanup(setup.Cleanup)

	mcpProxy := mcpserver.NewMCPServer("test", "0.0.1")
	sseMcpProxy := mcpserver.NewMCPServer("test-sse", "0.0.1")
	svc, err := mcpSvc.NewMCPService(&mcpSvc.ServiceConfig{
		DB:                      setup.DB,
		McpProxyServer:          mcpProxy,
		SseMcpProxyServer:       sseMcpProxy,
		Metrics:                 telemetry.NewNoopCustomMetrics(),
		McpServerInitReqTimeout: 5,
	})
	if err != nil {
		t.Fatalf("failed to create MCP service: %v", err)
	}

	tcSvc, err := toolcollection.NewToolCollectionService(setup.DB, svc)
	if err != nil {
		t.Fatalf("failed to create tool collection service: %v", err)
	}

	return &Server{toolCollectionService: tcSvc}
}

func TestGetToolCollectionHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := setupToolCollectionServer(t)

	router := gin.New()
	router.GET("/collections/:name", s.getToolCollectionHandler())

	req := httptest.NewRequest(http.MethodGet, "/collections/ghost-collection", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusNotFound, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "not found")
}

func TestUpdateToolCollectionHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := setupToolCollectionServer(t)

	router := gin.New()
	router.PUT("/collections/:name", s.updateToolCollectionHandler())

	req := httptest.NewRequest(http.MethodPut, "/collections/ghost-collection",
		strings.NewReader(`{"name":"ghost-collection","description":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusNotFound, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "not found")
}
