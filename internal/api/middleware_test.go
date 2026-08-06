package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/config"
	"github.com/mcpjungle/mcpjungle/internal/service/devicetoken"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/permission"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/internal/service/usersession"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

func TestRequireInitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupConfig    func(*gorm.DB) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "server is initialized",
			setupConfig: func(testDB *gorm.DB) error {
				configService := config.NewServerConfigService(testDB)
				_, err := configService.Init(model.ModeDev)
				return err
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name: "server is not initialized",
			setupConfig: func(testDB *gorm.DB) error {
				cfg := model.ServerConfig{
					Initialized: false,
					Mode:        model.ModeDev,
				}
				return testDB.Create(&cfg).Error
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"server is not initialized"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := testhelpers.SetupTestDB(t)
			defer setup.Cleanup()
			testDB := setup.DB
			configService := config.NewServerConfigService(testDB)

			err := tt.setupConfig(testDB)
			if err != nil {
				t.Fatalf("Setup config failed: %v", err)
			}

			server := &Server{configService: configService}
			router := gin.New()
			router.Use(server.requireInitialized())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestVerifyUserAuthForAPIAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()
	testDB := setup.DB

	userService := user.NewUserService(testDB)
	sessionService := usersession.NewService(testDB)

	// Create admin user for session-based auth tests
	adminUser, err := userService.CreateAdminUser("admin", "test-password-123")
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	// Create a valid session for the admin user
	rawSessionID, _, err := sessionService.Create(adminUser.ID, "", "", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	tests := []struct {
		name           string
		mode           model.ServerMode
		sessionToken   string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "dev mode - no auth required",
			mode:           model.ModeDev,
			sessionToken:   "",
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "enterprise mode - valid session",
			mode:           model.ModeEnterprise,
			sessionToken:   rawSessionID,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "enterprise mode - missing session",
			mode:           model.ModeEnterprise,
			sessionToken:   "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid credentials"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.mode != "" {
					c.Set("mode", tt.mode)
				}
			})
			server := &Server{
				userService:        userService,
				userSessionService: sessionService,
			}
			router.Use(server.verifyUserAuthForAPIAccess())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.sessionToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.sessionToken)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestRequireAdminUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testDB := testhelpers.SetupTestDB(t).DB
	userService := user.NewUserService(testDB)

	tests := []struct {
		name           string
		mode           model.ServerMode
		user           any
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "dev mode - no admin check required",
			mode:           model.ModeDev,
			user:           nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name: "enterprise mode - system admin user",
			mode: model.ModeEnterprise,
			user: &model.User{
				Model:    gorm.Model{ID: 1},
				Username: "admin",
				Role:     types.UserRoleSystemAdmin,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name: "enterprise mode - regular member user",
			mode: model.ModeEnterprise,
			user: &model.User{
				Model:    gorm.Model{ID: 1},
				Username: "user",
				Role:     types.UserRoleMember,
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"user is not authorized to perform this action"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.mode != "" {
					c.Set("mode", tt.mode)
				}
				if tt.user != nil {
					c.Set("user", tt.user)
				}
			})
			server := &Server{userService: userService}
			router.Use(server.requireAdminUser())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestRequireServerMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		contextMode    model.ServerMode
		requiredMode   model.ServerMode
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "matching mode - dev",
			contextMode:    model.ModeDev,
			requiredMode:   model.ModeDev,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "non-matching mode - dev required, enterprise context",
			contextMode:    model.ModeEnterprise,
			requiredMode:   model.ModeDev,
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"this request is only allowed in development mode"}`,
		},
		{
			name:           "non-matching mode - dev required, prod context",
			contextMode:    model.ModeProd,
			requiredMode:   model.ModeDev,
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"this request is only allowed in development mode"}`,
		},
		{
			name:           "enterprise required, prod context (deprecated)",
			contextMode:    model.ModeProd,
			requiredMode:   model.ModeEnterprise,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "prod required, enterprise context (deprecated)",
			contextMode:    model.ModeEnterprise,
			requiredMode:   model.ModeProd,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "prod required, prod context (deprecated)",
			contextMode:    model.ModeProd,
			requiredMode:   model.ModeProd,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "enterprise required, enterprise context",
			contextMode:    model.ModeEnterprise,
			requiredMode:   model.ModeEnterprise,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.contextMode != "" {
					c.Set("mode", tt.contextMode)
				}
			})
			server := &Server{}
			router.Use(server.requireServerMode(tt.requiredMode))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestCheckAuthForMcpProxyAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()
	testDB := setup.DB

	dtService := devicetoken.NewService(testDB)
	permService := permission.NewService(testDB)
	mcpSvc, err := mcp.NewMCPService(&mcp.ServiceConfig{
		DB:                      testDB,
		McpProxyServer:          mcpserver.NewMCPServer("test", "0.0.1"),
		SseMcpProxyServer:       mcpserver.NewMCPServer("test-sse", "0.0.1"),
		Metrics:                 telemetry.NewNoopCustomMetrics(),
		McpServerInitReqTimeout: 5,
	})
	if err != nil {
		t.Fatalf("failed to create MCP service: %v", err)
	}

	// Create a test user for the device token.
	testUser := &model.User{
		Username: "testuser",
		Role:     types.UserRoleMember,
	}
	if err := testDB.Create(testUser).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create a valid device token for the test user.
	rawToken, _, err := dtService.Create(testUser.ID, "test-device", model.DeviceTokenScopeInheritAll, nil, 0)
	if err != nil {
		t.Fatalf("failed to create device token: %v", err)
	}

	tests := []struct {
		name           string
		mode           model.ServerMode
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "dev mode - no auth required",
			mode:           model.ModeDev,
			authHeader:     "",
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "enterprise mode - valid token",
			mode:           model.ModeEnterprise,
			authHeader:     "Bearer " + rawToken,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "enterprise mode - missing token",
			mode:           model.ModeEnterprise,
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"missing device token"}`,
		},
		{
			name:           "enterprise mode - invalid token",
			mode:           model.ModeEnterprise,
			authHeader:     "Bearer mcpdt_999_badtoken",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid device token"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.mode != "" {
					c.Set("mode", tt.mode)
				}
			})
			server := &Server{
				deviceTokenService: dtService,
				permissionService:  permService,
				mcpService:         mcpSvc,
				userService:        user.NewUserService(testDB),
			}
			router.Use(server.checkAuthForMcpProxyAccess())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestMiddlewareIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()
	testDB := setup.DB

	configService := config.NewServerConfigService(testDB)
	userService := user.NewUserService(testDB)
	sessionService := usersession.NewService(testDB)

	// Setup config
	_, err := configService.Init(model.ModeEnterprise)
	if err != nil {
		t.Fatalf("Setup config failed: %v", err)
	}

	// Setup system admin user
	adminUser, err := userService.CreateAdminUser("admin", "test-password-123")
	if err != nil {
		t.Fatalf("Setup user failed: %v", err)
	}

	// Create a valid session for the admin user
	rawSessionID, _, err := sessionService.Create(adminUser.ID, "", "", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	server := &Server{
		configService:      configService,
		userService:        userService,
		userSessionService: sessionService,
	}
	router := gin.New()
	router.Use(server.requireInitialized())
	router.Use(server.verifyUserAuthForAPIAccess())
	router.Use(server.requireAdminUser())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "admin access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+rawSessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	expectedBody := `{"status":"admin access granted"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}
