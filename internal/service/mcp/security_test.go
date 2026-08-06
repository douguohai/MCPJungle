package mcp

import (
	"context"
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/devicetoken"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupSecurityTestDB creates a test DB with the models required for the
// security/authorization tests and returns it.
func setupSecurityTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := testhelpers.CreateTestDB(t)
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.User{},
		&model.McpServer{},
		&model.Tool{},
		&model.Prompt{},
		&model.Resource{},
		&model.UpstreamOAuthToken{},
		&model.UpstreamOAuthPendingSession{},
		&model.DeviceToken{},
		&model.DeviceTokenService{},
		&model.PermissionGroup{},
		&model.PermissionGroupMember{},
		&model.PermissionGroupService{},
	)
	require.NoError(t, err)

	return db
}

// makeEnterpriseCtx builds a context with mode=enterprise and the given
// effective_services map (server-name -> allowed).
func makeEnterpriseCtx(effective map[string]bool) context.Context {
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = context.WithValue(ctx, "effective_services", effective)
	return ctx
}

func insertTestServer(t *testing.T, db *gorm.DB, name, status string) *model.McpServer {
	t.Helper()

	srv, err := model.NewStreamableHTTPServer(name, name, "http://localhost:9999", "", nil, types.SessionModeStateless)
	require.NoError(t, err)
	srv.Status = status
	srv.Slug = name
	require.NoError(t, db.Create(srv).Error)
	return srv
}

// --- authorizeProxyServerAccess tests --------------------------------------

func TestAuthorizeProxyServerAccess_Unhealthy(t *testing.T) {
	// Unhealthy server should return a "temporarily unavailable" error.
	db := setupSecurityTestDB(t)
	svc := &MCPService{db: db}

	insertTestServer(t, db, "flaky-srv", model.StatusUnhealthy)

	ctx := makeEnterpriseCtx(map[string]bool{"flaky-srv": true})
	err := svc.authorizeProxyServerAccess(ctx, "flaky-srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "temporarily unavailable")
}

func TestAuthorizeProxyServerAccess_Disabled(t *testing.T) {
	db := setupSecurityTestDB(t)
	svc := &MCPService{db: db}

	insertTestServer(t, db, "off-srv", model.StatusDisabled)

	ctx := makeEnterpriseCtx(map[string]bool{"off-srv": true})
	err := svc.authorizeProxyServerAccess(ctx, "off-srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestAuthorizeProxyServerAccess_Archived(t *testing.T) {
	db := setupSecurityTestDB(t)
	svc := &MCPService{db: db}

	insertTestServer(t, db, "old-srv", model.StatusArchived)

	ctx := makeEnterpriseCtx(map[string]bool{"old-srv": true})
	err := svc.authorizeProxyServerAccess(ctx, "old-srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "archived")
}

func TestAuthorizeProxyServerAccess_Online(t *testing.T) {
	db := setupSecurityTestDB(t)
	svc := &MCPService{db: db}

	insertTestServer(t, db, "good-srv", model.StatusOnline)

	ctx := makeEnterpriseCtx(map[string]bool{"good-srv": true})
	err := svc.authorizeProxyServerAccess(ctx, "good-srv")
	assert.NoError(t, err)
}

func TestAuthorizeProxyServerAccess_NotInEffectiveServices(t *testing.T) {
	db := setupSecurityTestDB(t)
	svc := &MCPService{db: db}

	insertTestServer(t, db, "secret-srv", model.StatusOnline)

	// effective_services does NOT include "secret-srv".
	ctx := makeEnterpriseCtx(map[string]bool{"other-srv": true})
	err := svc.authorizeProxyServerAccess(ctx, "secret-srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestAuthorizeProxyServerAccess_DevModeSkipsCheck(t *testing.T) {
	db := setupSecurityTestDB(t)
	svc := &MCPService{db: db}

	// In dev mode the method short-circuits and returns nil regardless.
	ctx := context.WithValue(context.Background(), "mode", model.ModeDev)
	err := svc.authorizeProxyServerAccess(ctx, "any-server")
	assert.NoError(t, err)
}

// --- Security: restricted token cannot escalate permissions -----------------

func TestDeviceTokenCannotEscalatePermissions(t *testing.T) {
	db := setupSecurityTestDB(t)
	svc := &MCPService{db: db}

	// Set up a server that is online.
	insertTestServer(t, db, "prod-srv", model.StatusOnline)

	// A restricted token's effective_services map only contains the servers it
	// was explicitly granted. It must NOT be able to access "prod-srv" even if
	// a hypothetical inherit_all token would.
	restrictedEffective := map[string]bool{"limited-srv": true}
	ctx := makeEnterpriseCtx(restrictedEffective)

	err := svc.authorizeProxyServerAccess(ctx, "prod-srv")
	require.Error(t, err, "restricted token must not access servers outside its scope")
	assert.Contains(t, err.Error(), "access denied")
}

// --- Security: revoked token immediately rejected --------------------------

func TestRevokedTokenImmediatelyRejected(t *testing.T) {
	db := setupSecurityTestDB(t)

	user := &model.User{Username: "sec-user", Role: types.UserRoleMember}
	require.NoError(t, db.Create(user).Error)

	dtSvc := devicetoken.NewService(db)

	raw, tok, err := dtSvc.Create(user.ID, "temp-token", model.DeviceTokenScopeInheritAll, nil, 0)
	require.NoError(t, err)

	// Verify token works.
	got, err := dtSvc.GetByToken(raw)
	require.NoError(t, err)
	assert.Equal(t, tok.ID, got.ID)

	// Revoke.
	require.NoError(t, dtSvc.Revoke(tok.ID))

	// Must be immediately rejected.
	_, err = dtSvc.GetByToken(raw)
	assert.Error(t, err, "revoked token must fail GetByToken immediately")
}
