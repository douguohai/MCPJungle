// Package api provides HTTP API functionality for the MCPJungle server.
package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/dashboardui"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/config"
	"github.com/mcpjungle/mcpjungle/internal/service/dashboard"
	"github.com/mcpjungle/mcpjungle/internal/service/devicetoken"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/permission"
	sessionservice "github.com/mcpjungle/mcpjungle/internal/service/session"
	"github.com/mcpjungle/mcpjungle/internal/service/toolgroup"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/mcpjungle/mcpjungle/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

const (
	V0PathPrefix    = "/v0"
	V0ApiPathPrefix = "/api" + V0PathPrefix
)

type ServerOptions struct {
	// MCPProxyServer is the MCP proxy server instance that contains tools for all MCP servers
	// using the stdio or streamable http transport.
	MCPProxyServer *server.MCPServer
	// SseMcpProxyServer is the MCP proxy server instance that contains tools for all MCP servers
	// using the SSE transport.
	// sse tools are kept separate because SSE is supported for backward compatibility reasons, and
	// we don't want it to interfere with the usual mcp proxy server.
	// Both sse & streamable http use http, and we don't want to mix them up either.
	SseMcpProxyServer *server.MCPServer

	MCPService         *mcp.MCPService
	ConfigService      *config.ServerConfigService
	UserService        *user.UserService
	ToolGroupService   *toolgroup.ToolGroupService
	DashboardService   *dashboard.Service
	SessionService     *sessionservice.Service
	PermissionService  *permission.Service
	DeviceTokenService *devicetoken.Service

	OtelProviders *telemetry.Providers
	Metrics       telemetry.CustomMetrics
}

// Server represents the MCPJungle registry server that handles MCP proxy and API requests
type Server struct {
	router *gin.Engine

	mcpProxyServer    *server.MCPServer
	sseMcpProxyServer *server.MCPServer

	mcpService *mcp.MCPService

	configService      *config.ServerConfigService
	userService        *user.UserService
	toolGroupService   *toolgroup.ToolGroupService
	dashboardService   *dashboard.Service
	sessionService     *sessionservice.Service
	permissionService  *permission.Service
	deviceTokenService *devicetoken.Service

	otelProviders *telemetry.Providers
	metrics       telemetry.CustomMetrics

	// groupMcpServers keeps track of mcp-go's server.SSEServer instances created for each tool group.
	// These instances serve the requests made to tool groups' SSE tools.
	// We need to maintain one instance for each group for sse to work correctly.
	groupSseServers sync.Map

	// dashboardOAuthMu guards dashboardOAuthResults, which is a short-lived
	// in-memory cache used by the browser-based dashboard OAuth flow.
	dashboardOAuthMu sync.Mutex
	// dashboardOAuthResults stores terminal dashboard-facing OAuth status for a
	// session ID (completed/failed/expired) so the frontend can poll for
	// progress after opening the upstream authorization URL.
	dashboardOAuthResults map[string]dashboardOAuthSessionResult
}

// dashboardOAuthSessionResult is the dashboard-facing terminal state for an
// upstream OAuth registration attempt. This is not the source of truth for the
// pending OAuth session itself; it is a lightweight cache used to coordinate
// callback completion and frontend polling.
type dashboardOAuthSessionResult struct {
	Status     string
	Error      string
	ServerName string
	ExpiresAt  time.Time
	UpdatedAt  time.Time
}

// NewServer initializes a new Gin server for MCPJungle registry and MCP proxy
func NewServer(opts *ServerOptions) (*Server, error) {
	s := &Server{
		mcpProxyServer:        opts.MCPProxyServer,
		sseMcpProxyServer:     opts.SseMcpProxyServer,
		mcpService:            opts.MCPService,
		configService:         opts.ConfigService,
		userService:           opts.UserService,
		toolGroupService:      opts.ToolGroupService,
		dashboardService:      opts.DashboardService,
		sessionService:        opts.SessionService,
		permissionService:     opts.PermissionService,
		deviceTokenService:    opts.DeviceTokenService,
		otelProviders:         opts.OtelProviders,
		metrics:               opts.Metrics,
		dashboardOAuthResults: make(map[string]dashboardOAuthSessionResult),
	}

	// Set up the router after the server is fully initialized
	r, err := s.setupRouter()
	if err != nil {
		return nil, err
	}
	s.router = r

	return s, nil
}

// IsInitialized returns true if the server is initialized
func (s *Server) IsInitialized() (bool, error) {
	c, err := s.configService.GetConfig()
	if err != nil {
		return false, fmt.Errorf("failed to get server config: %w", err)
	}
	return c.Initialized, nil
}

// GetMode returns the server mode if the server is initialized, otherwise an error
func (s *Server) GetMode() (model.ServerMode, error) {
	ok, err := s.IsInitialized()
	if err != nil {
		return "", fmt.Errorf("failed to check if server is initialized: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("server is not initialized")
	}
	c, err := s.configService.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get server config: %w", err)
	}
	return c.Mode, nil
}

// InitDev initializes the server configuration in the Development mode.
// This method does not create an admin user because that is irrelevant in dev mode.
func (s *Server) InitDev() error {
	_, err := s.configService.Init(model.ModeDev)
	if err != nil {
		return fmt.Errorf("failed to initialize server config in dev mode: %w", err)
	}
	return nil
}

// Router returns the underlying HTTP handler for use with a custom HTTP server.
// This is useful for graceful shutdown support.
func (s *Server) Router() http.Handler {
	return s.router
}

// setupRouter sets up the Gin router with the MCP proxy server and API endpoints.
func (s *Server) setupRouter() (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// if otel is enabled, setup prometheus metrics endpoint
	if s.otelProviders != nil && s.otelProviders.IsEnabled() {
		// instrument gin
		r.Use(otelgin.Middleware(s.otelProviders.ServiceName()))

		// expose prometheus metrics endpoint
		r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	r.GET(
		"/health",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	r.GET(
		"/metadata",
		func(c *gin.Context) {
			m := &types.ServerMetadata{
				Version: version.GetVersion(),
			}
			c.JSON(http.StatusOK, m)
		},
	)

	r.POST("/init", s.registerInitServerHandler())

	requireDashboardMode := s.requireDashboardMode()

	if s.sessionService != nil && s.permissionService != nil && s.deviceTokenService != nil {
		apiV1 := r.Group("/api/v1", s.requireInitialized())
		apiV1.POST("/auth/login", s.v1Login)
		authenticated := apiV1.Group("/", s.requireV1Session())
		authenticated.POST("/auth/logout", s.v1Logout)
		authenticated.GET("/auth/me", s.v1Me)
		authenticated.PUT("/auth/password", s.v1ChangePassword)

		active := authenticated.Group("/", s.requireActiveV1User())
		active.POST("/device-tokens", s.v1CreateDeviceToken)
		active.GET("/device-tokens", s.v1ListDeviceTokens)
		active.DELETE("/device-tokens/:id", s.v1RevokeDeviceToken)

		admin := active.Group("/", s.requireV1SystemAdmin())
		admin.POST("/users", s.v1CreateUser)
		admin.GET("/users", s.v1ListUsers)
		admin.PATCH("/users/:id", s.v1UpdateUser)
		admin.POST("/users/:id/enable", s.v1EnableUser)
		admin.POST("/users/:id/disable", s.v1DisableUser)
		admin.GET("/users/:id/device-tokens", s.v1ListUserDeviceTokens)
		admin.POST("/permission-groups", s.v1CreatePermissionGroup)
		admin.GET("/permission-groups", s.v1ListPermissionGroups)
		admin.GET("/permission-groups/:id", s.v1GetPermissionGroup)
		admin.PATCH("/permission-groups/:id", s.v1UpdatePermissionGroup)
		admin.PUT("/permission-groups/:id/users", s.v1ReplacePermissionGroupUsers)
		admin.PUT("/permission-groups/:id/services", s.v1ReplacePermissionGroupServices)
		admin.POST("/permission-groups/:id/enable", s.v1EnablePermissionGroup)
		admin.POST("/permission-groups/:id/disable", s.v1DisablePermissionGroup)
	}

	if s.dashboardService != nil {
		dashboardFileServer, err := dashboardui.FileServer()
		if err != nil {
			return nil, err
		}
		r.GET("/", s.requireInitialized(), requireDashboardMode, gin.WrapH(dashboardFileServer))
		r.GET("/index.html", s.requireInitialized(), requireDashboardMode, gin.WrapH(dashboardFileServer))
		r.GET("/assets/*filepath", s.requireInitialized(), requireDashboardMode, gin.WrapH(dashboardFileServer))

		// SPA fallback: serve index.html for any unknown non-API route so that
		// client-side routes (e.g. /servers) survive a hard refresh. API and MCP
		// paths still return a proper 404 JSON.
		r.NoRoute(func(c *gin.Context) {
			p := c.Request.URL.Path
			if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/v0/") ||
				strings.HasPrefix(p, "/mcp") || strings.HasPrefix(p, "/sse") ||
				strings.HasPrefix(p, "/message") || strings.HasPrefix(p, "/health") ||
				strings.HasPrefix(p, "/metrics") || strings.HasPrefix(p, "/metadata") ||
				strings.HasPrefix(p, "/init") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			cfg, err := s.configService.GetConfig()
			if err != nil || !cfg.Initialized {
				c.JSON(http.StatusForbidden, gin.H{"error": "server is not initialized"})
				return
			}
			if cfg.Mode != model.ModeDev && !model.IsEnterpriseMode(cfg.Mode) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			// Rewrite to "/" so the embedded file server returns index.html.
			c.Request.URL.Path = "/"
			dashboardFileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	// Set up the MCP proxy server on /mcp
	streamableHTTPServer := server.NewStreamableHTTPServer(s.mcpProxyServer)
	r.Any(
		"/mcp",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		gin.WrapH(streamableHTTPServer),
	)

	r.Any(
		V0PathPrefix+"/groups/:name/mcp",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		s.toolGroupMCPServerCallHandler(),
	)

	// Set up the SSE transport-based MCP proxy server for the global /sse endpoint
	sseServer := server.NewSSEServer(s.sseMcpProxyServer)
	r.Any(
		"/sse",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		gin.WrapH(sseServer.SSEHandler()),
	)
	r.Any(
		"/message",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		gin.WrapH(sseServer.MessageHandler()),
	)

	r.Any(
		V0PathPrefix+"/groups/:name/sse",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		s.toolGroupSseMCPServerCallHandler(),
	)
	r.Any(
		V0PathPrefix+"/groups/:name/message",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		s.toolGroupSseMCPServerCallMessageHandler(),
	)

	// Setup /v0 API endpoints
	apiV0 := r.Group(
		V0ApiPathPrefix,
		s.requireInitialized(),
		s.verifyUserAuthForAPIAccess(),
	)

	// The legacy human invocation/read endpoints are development-only. In
	// enterprise mode, MCP access must use a personal device token through the
	// MCP transport so the permission-group decision cannot be bypassed.
	userAPI := apiV0.Group("/", s.requireServerMode(model.ModeDev))
	{
		userAPI.GET("/servers", s.listServersHandler())

		userAPI.GET("/tools", s.listToolsHandler())
		userAPI.POST("/tools/invoke", s.invokeToolHandler())
		userAPI.GET("/tool", s.getToolHandler())

		userAPI.GET("/resources", s.listResourcesHandler())
		userAPI.POST("/resources/get", s.getResourceHandler())
		userAPI.POST("/resources/read", s.readResourceHandler())

		// Prompt endpoints
		userAPI.GET("/prompts", s.listPromptsHandler())
		userAPI.GET("/prompt", s.getPromptHandler())
		userAPI.POST("/prompts/render", s.getPromptWithArgsHandler())

	}

	// endpoints only accessible by an admin user in enterprise mode or anyone in development mode
	adminAPI := apiV0.Group("/", s.requireAdminUser())
	{
		adminAPI.POST("/servers", s.registerServerHandler())
		adminAPI.POST("/upstream_oauth/sessions/:id/complete", s.completeUpstreamOAuthSessionHandler())
		adminAPI.DELETE("/servers/:name", s.deregisterServerHandler())
		adminAPI.POST("/servers/:name/enable", s.enableServerHandler())
		adminAPI.POST("/servers/:name/disable", s.disableServerHandler())

		// this endpoint is restricted to admins only because it can potentially expose sensitive information
		// like bearer tokens.
		adminAPI.GET("/server_configs", s.getServerConfigsHandler())

		adminAPI.POST("/tools/enable", s.enableToolsHandler())
		adminAPI.POST("/tools/disable", s.disableToolsHandler())

		adminAPI.POST("/prompts/enable", s.enablePromptsHandler())
		adminAPI.POST("/prompts/disable", s.disablePromptsHandler())

		// endpoints for managing tool groups
		adminAPI.POST("/tool-groups", s.createToolGroupHandler())
		adminAPI.GET("/tool-groups/:name", s.getToolGroupHandler())
		adminAPI.GET("/tool-groups/:name/effective-tools", s.getToolGroupEffectiveToolsHandler())
		adminAPI.GET("/tool-groups", s.listToolGroupsHandler())
		adminAPI.DELETE("/tool-groups/:name", s.deleteToolGroupHandler())
		adminAPI.PUT("/tool-groups/:name", s.updateToolGroupHandler())
	}

	if s.dashboardService != nil {
		// Public dashboard endpoints: reachable without a user token so the login
		// page can verify credentials before the user is authenticated.
		dashboardPublicAPI := r.Group(
			"/api/dashboard",
			s.requireInitialized(),
			requireDashboardMode,
		)
		{
			dashboardPublicAPI.GET("/settings", s.dashboardSettingsHandler())
			dashboardPublicAPI.POST("/auth/verify", s.dashboardVerifyTokenHandler())
			dashboardPublicAPI.POST("/auth/login", s.dashboardLoginHandler())
		}

		// Protected dashboard endpoints: require a valid user access token in
		// enterprise mode (development mode is auto-allowed by the middleware).
		dashboardAPI := r.Group(
			"/api/dashboard",
			s.requireInitialized(),
			requireDashboardMode,
			s.verifyUserAuthForAPIAccess(),
		)
		{
			// Read endpoints: accessible to any authenticated dashboard user.
			dashboardAPI.GET("/overview", s.dashboardOverviewHandler())
			dashboardAPI.GET("/servers", s.dashboardServersHandler())
			dashboardAPI.GET("/oauth/callback", s.dashboardOAuthCallbackHandler())
			dashboardAPI.GET("/oauth/session/:id", s.dashboardOAuthSessionHandler())
			dashboardAPI.GET("/tools", s.dashboardToolsHandler())
			dashboardAPI.GET("/tool-groups", s.dashboardToolGroupsHandler())
			dashboardAPI.GET("/tool-groups/:name", s.dashboardGetToolGroupHandler())
			dashboardAPI.GET("/prompts", s.dashboardPromptsHandler())
			dashboardAPI.GET("/resources", s.dashboardResourcesHandler())
			dashboardAPI.GET("/diagnostics", s.dashboardDiagnosticsHandler())
		}

		// Mutation endpoints: restricted to admin users in enterprise mode, mirroring
		// the requireAdminUser guard applied to the equivalent /v0 admin API routes.
		// Without this, any authenticated user could delete servers or manage tool
		// groups via the dashboard API while the same action is forbidden via /v0.
		dashboardAdminAPI := dashboardAPI.Group("/", s.requireAdminUser())
		{
			dashboardAdminAPI.POST("/servers", s.dashboardRegisterServerHandler())
			dashboardAdminAPI.DELETE("/servers/:name", s.dashboardDeleteServerHandler())
			dashboardAdminAPI.PATCH("/servers/:name/enabled", s.dashboardSetServerEnabledHandler())
			dashboardAdminAPI.PATCH("/tools/:name/enabled", s.dashboardSetToolEnabledHandler())
			dashboardAdminAPI.POST("/tool-groups", s.dashboardCreateToolGroupHandler())
			dashboardAdminAPI.DELETE("/tool-groups/:name", s.dashboardDeleteToolGroupHandler())
			dashboardAdminAPI.PATCH("/prompts/:name/enabled", s.dashboardSetPromptEnabledHandler())

			// Analytics: per-user per-server call counts (admin only).
			dashboardAdminAPI.GET("/stats", s.dashboardCallStatsHandler())
			dashboardAdminAPI.PUT("/settings", s.dashboardUpdateSettingsHandler())
		}
	}

	return r, nil
}
