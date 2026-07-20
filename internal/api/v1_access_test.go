package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/devicetoken"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/permission"
	"github.com/mcpjungle/mcpjungle/internal/service/session"
	userservice "github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

func setupV1AccessTest(t *testing.T) (*Server, *gin.Engine, *model.User, *gorm.DB) {
	t.Helper()
	db := testhelpers.RequireTestDB(t)
	if err := db.AutoMigrate(
		&model.User{}, &model.UserSession{}, &model.McpServer{},
		&model.Tool{}, &model.Prompt{}, &model.Resource{},
		&model.PermissionGroup{}, &model.PermissionGroupUser{}, &model.PermissionGroupMcpServer{},
		&model.DeviceToken{}, &model.DeviceTokenService{},
	); err != nil {
		t.Fatalf("migrate v1 access models: %v", err)
	}
	userSvc := userservice.NewUserService(db)
	admin, err := userSvc.CreateAdminUser("admin", "admin-password-123")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	permissionSvc := permission.NewService(db)
	sessionSvc := session.NewService(db)
	deviceSvc := devicetoken.NewService(db, permissionSvc)
	apiServer := &Server{userService: userSvc, permissionService: permissionSvc, sessionService: sessionSvc, deviceTokenService: deviceSvc}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/auth/login", apiServer.v1Login)
	authenticated := r.Group("/api/v1", apiServer.requireV1Session())
	authenticated.GET("/auth/me", apiServer.v1Me)
	authenticated.PUT("/auth/password", apiServer.v1ChangePassword)
	active := authenticated.Group("/", apiServer.requireActiveV1User())
	active.POST("/device-tokens", apiServer.v1CreateDeviceToken)
	adminRoutes := active.Group("/", apiServer.requireV1SystemAdmin())
	adminRoutes.POST("/users", apiServer.v1CreateUser)
	adminRoutes.GET("/users/:id/device-tokens", apiServer.v1ListUserDeviceTokens)
	return apiServer, r, admin, db
}

func TestV1LoginAndFirstPasswordChange(t *testing.T) {
	_, router, _, _ := setupV1AccessTest(t)
	adminCookie := loginV1(t, router, "admin", "admin-password-123")
	if !adminCookie.HttpOnly || adminCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unsafe session cookie: %+v", adminCookie)
	}

	create := requestV1(t, router, http.MethodPost, "/api/v1/users", map[string]any{"username": "alice", "display_name": "Alice", "role": "member"}, adminCookie)
	if create.Code != http.StatusCreated {
		t.Fatalf("create user: %d %s", create.Code, create.Body.String())
	}
	if strings.Contains(create.Body.String(), "password_hash") {
		t.Fatal("user response leaked password hash")
	}
	var created struct {
		InitialPassword string `json:"initial_password"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil || created.InitialPassword == "" {
		t.Fatalf("decode initial password: %v", err)
	}

	aliceCookie := loginV1(t, router, "alice", created.InitialPassword)
	denied := requestV1(t, router, http.MethodPost, "/api/v1/device-tokens", map[string]any{"name": "laptop"}, aliceCookie)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("pending user should be gated, got %d", denied.Code)
	}
	changed := requestV1(t, router, http.MethodPut, "/api/v1/auth/password", map[string]any{"current_password": created.InitialPassword, "new_password": "alice-password-123"}, aliceCookie)
	if changed.Code != http.StatusOK {
		t.Fatalf("change password: %d %s", changed.Code, changed.Body.String())
	}
	if stale := requestV1(t, router, http.MethodPost, "/api/v1/device-tokens", map[string]any{"name": "stale"}, aliceCookie); stale.Code != http.StatusUnauthorized {
		t.Fatalf("pre-change session should be revoked, got %d", stale.Code)
	}
	changedCookies := changed.Result().Cookies()
	if len(changedCookies) == 0 || changedCookies[0].Value == aliceCookie.Value {
		t.Fatal("password change did not rotate the session cookie")
	}
	aliceCookie = changedCookies[0]
	createdToken := requestV1(t, router, http.MethodPost, "/api/v1/device-tokens", map[string]any{"name": "laptop"}, aliceCookie)
	if createdToken.Code != http.StatusCreated {
		t.Fatalf("create device token: %d %s", createdToken.Code, createdToken.Body.String())
	}
	if !strings.Contains(createdToken.Body.String(), `"token":"mcpdt_`) {
		t.Fatalf("creation response did not show token once: %s", createdToken.Body.String())
	}
	if strings.Contains(createdToken.Body.String(), "token_hash") {
		t.Fatalf("device token response leaked token hash: %s", createdToken.Body.String())
	}

	var createPayload struct {
		User struct {
			ID uint `json:"ID"`
		} `json:"user"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &createPayload); err != nil || createPayload.User.ID == 0 {
		t.Fatalf("decode created user: %v", err)
	}
	listed := requestV1(t, router, http.MethodGet, "/api/v1/users/"+strconv.FormatUint(uint64(createPayload.User.ID), 10)+"/device-tokens", nil, adminCookie)
	if listed.Code != http.StatusOK {
		t.Fatalf("admin list user tokens: %d %s", listed.Code, listed.Body.String())
	}
	if strings.Contains(listed.Body.String(), "token_hash") || strings.Contains(listed.Body.String(), `"token":"mcpdt_`) {
		t.Fatalf("admin token metadata leaked a credential: %s", listed.Body.String())
	}
}

func TestMCPMiddlewareAuthenticatesPersonalDeviceToken(t *testing.T) {
	apiServer, _, admin, db := setupV1AccessTest(t)
	mcpProxy := server.NewMCPServer("test", "1")
	sseProxy := server.NewMCPServer("test-sse", "1")
	mcpSvc, err := mcp.NewMCPService(&mcp.ServiceConfig{DB: db, McpProxyServer: mcpProxy, SseMcpProxyServer: sseProxy, Metrics: telemetry.NewNoopCustomMetrics(), McpServerInitReqTimeout: 5})
	if err != nil {
		t.Fatal(err)
	}
	apiServer.mcpService = mcpSvc
	upstream, err := model.NewSSEServer("allowed", "", "https://example.com/sse", "", types.SessionModeStateless)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(upstream).Error; err != nil {
		t.Fatal(err)
	}
	group, err := apiServer.permissionService.Create(permission.CreateInput{Name: "members", DisplayName: "Members"})
	if err != nil {
		t.Fatal(err)
	}
	if err := apiServer.permissionService.ReplaceUsers(group.ID, []uint{admin.ID}); err != nil {
		t.Fatal(err)
	}
	if err := apiServer.permissionService.ReplaceServices(group.ID, []uint{upstream.ID}); err != nil {
		t.Fatal(err)
	}
	_, plain, err := apiServer.deviceTokenService.Create(admin.ID, devicetoken.CreateInput{Name: "test-client"})
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("mode", model.ModeEnterprise); c.Next() })
	r.GET("/probe", apiServer.checkAuthForMcpProxyAccess(), func(c *gin.Context) {
		access, ok := mcp.AccessFromContext(c.Request.Context())
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		_, allowed := access.EffectiveServiceIDs[upstream.ID]
		c.JSON(http.StatusOK, gin.H{"user_id": access.UserID, "allowed": allowed})
	})
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"allowed":true`) {
		t.Fatalf("device token auth failed: %d %s", w.Code, w.Body.String())
	}
}

func loginV1(t *testing.T, router http.Handler, username, password string) *http.Cookie {
	t.Helper()
	w := requestV1(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"username": username, "password": password}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("login %s: %d %s", username, w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not set session cookie")
	}
	return cookies[0]
}

func requestV1(t *testing.T, router http.Handler, method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
