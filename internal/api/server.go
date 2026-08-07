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
	"github.com/mcpjungle/mcpjungle/internal/middleware"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/auditevent"
	"github.com/mcpjungle/mcpjungle/internal/service/callevent"
	"github.com/mcpjungle/mcpjungle/internal/service/config"
	"github.com/mcpjungle/mcpjungle/internal/service/dashboard"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/devicetoken"
	"github.com/mcpjungle/mcpjungle/internal/service/permission"
	"github.com/mcpjungle/mcpjungle/internal/service/toolcollection"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/internal/service/usersession"
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

	MCPService       *mcp.MCPService
	ConfigService    *config.ServerConfigService
	UserService      *user.UserService
	ToolCollectionService *toolcollection.ToolCollectionService
	DashboardService *dashboard.Service

	OtelProviders *telemetry.Providers
	Metrics       telemetry.CustomMetrics

	// UserSessionService manages web dashboard sessions (cookie-based auth).
	UserSessionService *usersession.Service

	// PermissionService manages permission groups and user effective services.
	PermissionService *permission.Service
	DeviceTokenService *devicetoken.Service

	// CallEventService records detailed MCP call events for analytics.
	CallEventService *callevent.Service

	// AuditEventService records administrative audit events.
	AuditEventService *auditevent.Service
}

// Server represents the MCPJungle registry server that handles MCP proxy and API requests
type Server struct {
	router *gin.Engine

	mcpProxyServer    *server.MCPServer
	sseMcpProxyServer *server.MCPServer

	mcpService       *mcp.MCPService

	configService    *config.ServerConfigService
	userService      *user.UserService
	toolCollectionService *toolcollection.ToolCollectionService
	dashboardService *dashboard.Service

	otelProviders *telemetry.Providers
	metrics       telemetry.CustomMetrics

	userSessionService *usersession.Service
	permissionService  *permission.Service
	deviceTokenService  *devicetoken.Service

	callEventService   *callevent.Service
	auditEventService  *auditevent.Service

	// collectionSseServers keeps track of mcp-go's server.SSEServer instances created for each tool collection.
	// These instances serve the requests made to tool collections' SSE tools.
	// We need to maintain one instance for each collection for sse to work correctly.
	collectionSseServers sync.Map

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
		toolCollectionService: opts.ToolCollectionService,
		dashboardService:      opts.DashboardService,
		otelProviders:         opts.OtelProviders,
		metrics:               opts.Metrics,
		userSessionService:   opts.UserSessionService,
		permissionService:   opts.PermissionService,
		deviceTokenService:  opts.DeviceTokenService,
		callEventService:    opts.CallEventService,
		auditEventService:   opts.AuditEventService,
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
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// --- Production security middleware ---
	// CORS: allow cross-origin requests (configurable per deployment).
	r.Use(middleware.CORSMiddleware(middleware.DefaultCORSConfig()))
	// Body size limit: prevent memory exhaustion from oversized requests (10 MB default).
	r.Use(middleware.BodySizeLimitMiddleware(middleware.DefaultMaxBodySize))
	// Global rate limit: 30 req/s per IP, burst 60. Auth endpoints get tighter limits below.
	globalRL := middleware.NewRateLimiter(30, 60)
	r.Use(middleware.RateLimitMiddleware(globalRL))

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
			// Check database connectivity.
			dbStatus := "ok"
			if s.configService != nil {
				if err := s.configService.PingDB(); err != nil {
					dbStatus = "unavailable"
				}
			}
			status := "ok"
			code := http.StatusOK
			if dbStatus != "ok" {
				status = "degraded"
				code = http.StatusServiceUnavailable
			}
			c.JSON(code, gin.H{
				"status": status,
				"db":     dbStatus,
			})
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

	requireEnterpriseMode := s.requireServerMode(model.ModeEnterprise)
	requireDashboardMode := s.requireDashboardMode()

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
		V0PathPrefix+"/collections/:name/mcp",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		s.toolCollectionMCPServerCallHandler(),
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
		V0PathPrefix+"/collections/:name/sse",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		s.toolCollectionSseMCPServerCallHandler(),
	)
	r.Any(
		V0PathPrefix+"/collections/:name/message",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		s.toolCollectionSseMCPServerCallMessageHandler(),
	)

	// Setup /v0 API endpoints
	apiV0 := r.Group(
		V0ApiPathPrefix,
		s.requireInitialized(),
		s.verifyUserAuthForAPIAccess(),
	)

	// endpoints accessible by a standard user in enterprise mode or anyone in development mode
	userAPI := apiV0.Group("/")
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

		userAPI.GET("/users/whoami", requireEnterpriseMode, s.whoAmIHandler())

		// Analytics: call event details, daily aggregates, and summaries.
		// Handlers apply role-based filtering internally.
		userAPI.GET("/analytics/events", s.listCallEventsHandler())
		userAPI.GET("/analytics/daily", s.listDailyAggregatesHandler())
		userAPI.GET("/analytics/summary", s.callSummaryHandler())
	}

	// Audit events: only system_admin and auditor roles (handler enforces).
	auditAPI := apiV0.Group("/")
	{
		auditAPI.GET("/audit-events", s.listAuditEventsHandler())
	}

	// endpoints only accessible by an admin user in enterprise mode or anyone in development mode
	adminAPI := apiV0.Group("/", s.requireAdminUser())
	{
		adminAPI.POST("/servers", s.registerServerHandler())
		adminAPI.POST("/upstream_oauth/sessions/:id/complete", s.completeUpstreamOAuthSessionHandler())
		adminAPI.DELETE("/servers/:name", s.deregisterServerHandler())
		adminAPI.POST("/servers/:name/enable", s.enableServerHandler())
		adminAPI.POST("/servers/:name/disable", s.disableServerHandler())
		adminAPI.POST("/servers/:name/validate", s.validateServerHandler())
		adminAPI.POST("/servers/:name/publish", s.publishServerHandler())
		adminAPI.POST("/servers/:name/archive", s.archiveServerHandler())

		// Server manager CRUD
		adminAPI.POST("/servers/:name/managers", s.addServerManagerHandler())
		adminAPI.DELETE("/servers/:name/managers/:user_id", s.removeServerManagerHandler())
		adminAPI.GET("/servers/:name/managers", s.listServerManagersHandler())

		// this endpoint is restricted to admins only because it can potentially expose sensitive information
		// like bearer tokens.
		adminAPI.GET("/server_configs", s.getServerConfigsHandler())

		adminAPI.POST("/tools/enable", s.enableToolsHandler())
		adminAPI.POST("/tools/disable", s.disableToolsHandler())

		adminAPI.POST("/prompts/enable", s.enablePromptsHandler())
		adminAPI.POST("/prompts/disable", s.disablePromptsHandler())

		// endpoints for managing human users (enterprise mode only)
		adminAPI.POST(
			"/users",
			requireEnterpriseMode,
			s.createUserHandler(),
		)
		adminAPI.GET(
			"/users",
			requireEnterpriseMode,
			s.listUsersHandler(),
		)
		adminAPI.DELETE(
			"/users/:username",
			requireEnterpriseMode,
			s.deleteUserHandler(),
		)
		adminAPI.POST(
			"/users/:username/disable",
			requireEnterpriseMode,
			s.disableUserHandler(),
		)

		// endpoints for managing tool collections
		adminAPI.POST("/tool-collections", s.createToolCollectionHandler())
		adminAPI.GET("/tool-collections/:name", s.getToolCollectionHandler())
		adminAPI.GET("/tool-collections/:name/effective-tools", s.getToolCollectionEffectiveToolsHandler())
		adminAPI.GET("/tool-collections", s.listToolCollectionsHandler())
		adminAPI.DELETE("/tool-collections/:name", s.deleteToolCollectionHandler())
		adminAPI.PUT("/tool-collections/:name", s.updateToolCollectionHandler())
	}

	if s.dashboardService != nil {
		// Public dashboard endpoints: reachable without a user token so the login
		// page can verify credentials before the user is authenticated.
		// Auth endpoints get a stricter rate limit (5 req/s per IP) to prevent brute-force attacks.
		authRL := middleware.NewRateLimiter(5, 10)
		dashboardPublicAPI := r.Group(
			"/api/dashboard",
			s.requireInitialized(),
			requireDashboardMode,
			middleware.RateLimitMiddleware(authRL),
		)
		{
			dashboardPublicAPI.POST("/auth/verify", s.dashboardVerifyTokenHandler())
			dashboardPublicAPI.POST("/auth/login", s.dashboardLoginHandler())
			dashboardPublicAPI.POST("/auth/logout", s.dashboardLogoutHandler())
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
			dashboardAPI.GET("/tool-collections", s.dashboardToolCollectionsHandler())
			dashboardAPI.GET("/tool-collections/:name", s.dashboardGetToolCollectionHandler())
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
			dashboardAdminAPI.POST("/tool-collections", s.dashboardCreateToolCollectionHandler())
			dashboardAdminAPI.DELETE("/tool-collections/:name", s.dashboardDeleteToolCollectionHandler())
			dashboardAdminAPI.PATCH("/prompts/:name/enabled", s.dashboardSetPromptEnabledHandler())

			// Analytics: per-user per-server call counts (admin only).
			dashboardAdminAPI.GET("/stats", s.dashboardCallStatsHandler())

			// Permission groups (admin only).
			dashboardAdminAPI.GET("/permission-groups", s.listPermissionGroupsHandler())
			dashboardAdminAPI.POST("/permission-groups", s.createPermissionGroupHandler())
			dashboardAdminAPI.GET("/permission-groups/:id", s.getPermissionGroupHandler())
			dashboardAdminAPI.PATCH("/permission-groups/:id", s.updatePermissionGroupHandler())
			dashboardAdminAPI.POST("/permission-groups/:id/disable", s.disablePermissionGroupHandler())
			dashboardAdminAPI.POST("/permission-groups/:id/members", s.addPermissionGroupMemberHandler())
			dashboardAdminAPI.DELETE("/permission-groups/:id/members/:user_id", s.removePermissionGroupMemberHandler())
			dashboardAdminAPI.POST("/permission-groups/:id/services", s.addPermissionGroupServiceHandler())
			dashboardAdminAPI.DELETE("/permission-groups/:id/services/:server_id", s.removePermissionGroupServiceHandler())

			// Device tokens (admin only).
			dashboardAdminAPI.GET("/device-tokens", s.listDeviceTokensHandler())
			dashboardAdminAPI.POST("/device-tokens", s.createDeviceTokenHandler())
			dashboardAdminAPI.GET("/device-tokens/:id", s.getDeviceTokenHandler())
			dashboardAdminAPI.POST("/device-tokens/:id/revoke", s.revokeDeviceTokenHandler())
			dashboardAdminAPI.DELETE("/device-tokens/:id", s.deleteDeviceTokenHandler())
		}
	}

	return r, nil
}
