package devicetoken

import (
	"strings"
	"testing"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createServerWithSlug inserts an McpServer with an explicit Slug to avoid
// unique-index collisions (the test helper CreateTestMcpServer leaves Slug
// empty, which breaks when the table has a UNIQUE on slug).
func createServerWithSlug(t *testing.T, setup *testhelpers.TestDBSetup, name string) *model.McpServer {
	t.Helper()
	srv, err := model.NewStreamableHTTPServer(name, name, "http://localhost:1", "", nil, types.SessionModeStateless)
	require.NoError(t, err)
	srv.Slug = name
	require.NoError(t, setup.DB.Create(srv).Error)
	return srv
}

// setupDeviceTokenTest creates a test DB, a member user, and two MCP servers.
func setupDeviceTokenTest(t *testing.T) (*Service, *testhelpers.TestDBSetup, *model.User, *model.McpServer, *model.McpServer) {
	t.Helper()

	setup := testhelpers.SetupTestDB(t)
	svc := NewService(setup.DB)

	user := setup.CreateTestUser("tokenuser", types.UserRoleMember)
	srv1 := createServerWithSlug(t, setup, "svc-a")
	srv2 := createServerWithSlug(t, setup, "svc-b")

	return svc, setup, user, srv1, srv2
}

// --- Create tests ----------------------------------------------------------

func TestCreateDeviceToken(t *testing.T) {
	svc, _, user, _, _ := setupDeviceTokenTest(t)

	raw, tok, err := svc.Create(user.ID, "test-token", model.DeviceTokenScopeInheritAll, nil, 0)
	require.NoError(t, err)

	assert.NotEmpty(t, raw, "raw_token must be non-empty")
	assert.True(t, strings.HasPrefix(raw, "mcpdt_"), "raw_token must start with mcpdt_")
	assert.NotEmpty(t, tok.TokenPrefix, "TokenPrefix must be non-empty")
	assert.NotEmpty(t, tok.TokenHash, "TokenHash must be non-empty")
	assert.Equal(t, model.DeviceTokenScopeInheritAll, tok.ScopeMode)
	assert.Equal(t, model.DeviceTokenStatusActive, tok.Status)
	assert.Equal(t, user.ID, tok.UserID)
}

func TestCreateRestrictedToken(t *testing.T) {
	svc, _, user, srv1, srv2 := setupDeviceTokenTest(t)

	raw, tok, err := svc.Create(user.ID, "restricted", model.DeviceTokenScopeRestricted, []uint{srv1.ID, srv2.ID}, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, raw)
	assert.Equal(t, model.DeviceTokenScopeRestricted, tok.ScopeMode)

	svcIDs, err := svc.GetRestrictedServices(tok.ID)
	require.NoError(t, err)
	assert.Len(t, svcIDs, 2)
	// Sort for deterministic comparison.
	seen := map[uint]bool{}
	for _, id := range svcIDs {
		seen[id] = true
	}
	assert.True(t, seen[srv1.ID])
	assert.True(t, seen[srv2.ID])
}

// --- GetByToken tests ------------------------------------------------------

func TestGetByToken(t *testing.T) {
	svc, _, user, _, _ := setupDeviceTokenTest(t)

	raw, _, err := svc.Create(user.ID, "lookup-token", "", nil, 0)
	require.NoError(t, err)

	got, err := svc.GetByToken(raw)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.UserID)
	assert.Equal(t, model.DeviceTokenStatusActive, got.Status)
}

func TestGetByToken_InvalidToken(t *testing.T) {
	svc, _, _, _, _ := setupDeviceTokenTest(t)

	_, err := svc.GetByToken("not-a-valid-token")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestGetByToken_TamperedSecret(t *testing.T) {
	// Tamper with the last hex char of the secret to exercise constant-time
	// comparison rejection.
	svc, _, user, _, _ := setupDeviceTokenTest(t)

	raw, _, err := svc.Create(user.ID, "tamper-test", "", nil, 0)
	require.NoError(t, err)

	// Flip the last character.
	tampered := raw[:len(raw)-1]
	last := raw[len(raw)-1]
	if last == 'a' {
		tampered += "b"
	} else {
		tampered += "a"
	}

	_, err = svc.GetByToken(tampered)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "tampered secret must be rejected")
}

func TestGetByToken_RevokedToken(t *testing.T) {
	svc, _, user, _, _ := setupDeviceTokenTest(t)

	raw, tok, err := svc.Create(user.ID, "revoke-me", "", nil, 0)
	require.NoError(t, err)

	require.NoError(t, svc.Revoke(tok.ID))

	_, err = svc.GetByToken(raw)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "revoked token must not be found")
}

func TestGetByToken_ExpiredToken(t *testing.T) {
	svc, setup, user, _, _ := setupDeviceTokenTest(t)

	// Create with a 1-second TTL, then manually backdate the expiry.
	raw, tok, err := svc.Create(user.ID, "expires-soon", "", nil, 1*time.Second)
	require.NoError(t, err)

	// Backdate ExpiresAt to the past.
	past := time.Now().UTC().Add(-1 * time.Hour)
	require.NoError(t, setup.DB.Model(tok).Update("expires_at", past).Error)

	_, err = svc.GetByToken(raw)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "expired token must not be found")
}

// --- Revoke tests ----------------------------------------------------------

func TestRevokeToken(t *testing.T) {
	svc, setup, user, _, _ := setupDeviceTokenTest(t)

	_, tok, err := svc.Create(user.ID, "revocable", "", nil, 0)
	require.NoError(t, err)

	require.NoError(t, svc.Revoke(tok.ID))

	var updated model.DeviceToken
	require.NoError(t, setup.DB.First(&updated, tok.ID).Error)
	assert.Equal(t, model.DeviceTokenStatusRevoked, updated.Status)
	assert.NotNil(t, updated.RevokedAt)
}

// --- List tests ------------------------------------------------------------

func TestListTokens_AdminSeesAll(t *testing.T) {
	svc, _, user, _, _ := setupDeviceTokenTest(t)

	// Create two tokens for the member user.
	_, _, err := svc.Create(user.ID, "tok1", "", nil, 0)
	require.NoError(t, err)
	_, _, err = svc.Create(user.ID, "tok2", "", nil, 0)
	require.NoError(t, err)

	adminID := uint(0) // admin pseudo-user; List uses role, not userID.
	tokens, err := svc.List(adminID, types.UserRoleSystemAdmin)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tokens), 2)
}

func TestListTokens_NonAdminSeesOwnOnly(t *testing.T) {
	svc, _, user, _, _ := setupDeviceTokenTest(t)

	_, _, err := svc.Create(user.ID, "my-tok", "", nil, 0)
	require.NoError(t, err)

	tokens, err := svc.List(user.ID, types.UserRoleMember)
	require.NoError(t, err)
	assert.Len(t, tokens, 1)
	assert.Equal(t, "my-tok", tokens[0].Name)
}

// --- Delete tests ----------------------------------------------------------

func TestDeleteToken(t *testing.T) {
	svc, _, user, _, _ := setupDeviceTokenTest(t)

	_, tok, err := svc.Create(user.ID, "deletable", "", nil, 0)
	require.NoError(t, err)

	err = svc.Delete(tok.ID, user.ID, types.UserRoleMember)
	require.NoError(t, err)

	// Verify hard-deleted.
	_, err = svc.GetByID(tok.ID)
	assert.Error(t, err)
}

func TestDeleteToken_NonAdminCannotDeleteOthers(t *testing.T) {
	svc, setup, user, _, _ := setupDeviceTokenTest(t)

	other := setup.CreateTestUser("other", types.UserRoleMember)
	_, tok, err := svc.Create(other.ID, "other-tok", "", nil, 0)
	require.NoError(t, err)

	// user (non-admin) tries to delete other's token.
	err = svc.Delete(tok.ID, user.ID, types.UserRoleMember)
	assert.Error(t, err, "non-admin should not delete another user's token")
}
