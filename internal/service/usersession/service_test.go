package usersession

import (
	"testing"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupSessionTest creates a test DB and a single user.
func setupSessionTest(t *testing.T) (*Service, *testhelpers.TestDBSetup, *model.User) {
	t.Helper()

	setup := testhelpers.SetupTestDB(t)
	svc := NewService(setup.DB)
	user := setup.CreateTestUser("sessionuser", types.UserRoleMember)
	return svc, setup, user
}

// --- Create tests ----------------------------------------------------------

func TestCreateSession(t *testing.T) {
	svc, _, user := setupSessionTest(t)

	raw, sess, err := svc.Create(user.ID, "127.0.0.1", "ua-hash", 0)
	require.NoError(t, err)

	assert.NotEmpty(t, raw, "raw session id must be non-empty")
	assert.NotEmpty(t, sess.SessionIDHash, "SessionIDHash must be non-empty")
	assert.Equal(t, user.ID, sess.UserID)
	assert.Equal(t, "127.0.0.1", sess.SourceIP)
	assert.Equal(t, "ua-hash", sess.UAHash)

	// ExpiresAt should be ~8h from now (default TTL).
	assert.True(t, sess.ExpiresAt.After(time.Now().UTC().Add(7*time.Hour)),
		"ExpiresAt should be roughly 8h in the future")
	assert.Nil(t, sess.RevokedAt, "new session must not be revoked")
}

// --- Lookup tests ----------------------------------------------------------

func TestLookup(t *testing.T) {
	svc, _, user := setupSessionTest(t)

	raw, _, err := svc.Create(user.ID, "127.0.0.1", "h", 0)
	require.NoError(t, err)

	got, err := svc.Lookup(raw)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.UserID)
}

func TestLookup_InvalidID(t *testing.T) {
	svc, _, _ := setupSessionTest(t)

	_, err := svc.Lookup("non-existent-session-id")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestLookup_EmptyID(t *testing.T) {
	svc, _, _ := setupSessionTest(t)

	_, err := svc.Lookup("")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestLookup_Revoked(t *testing.T) {
	svc, _, user := setupSessionTest(t)

	raw, _, err := svc.Create(user.ID, "127.0.0.1", "h", 0)
	require.NoError(t, err)

	require.NoError(t, svc.Revoke(raw))

	_, err = svc.Lookup(raw)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "revoked session must not be found")
}

func TestLookup_Expired(t *testing.T) {
	svc, setup, user := setupSessionTest(t)

	// Create with a very short TTL.
	raw, sess, err := svc.Create(user.ID, "127.0.0.1", "h", 1*time.Second)
	require.NoError(t, err)

	// Backdate ExpiresAt to the past.
	past := time.Now().UTC().Add(-1 * time.Hour)
	require.NoError(t, setup.DB.Model(sess).Update("expires_at", past).Error)

	_, err = svc.Lookup(raw)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "expired session must not be found")
}

// --- Touch tests -----------------------------------------------------------

func TestTouch(t *testing.T) {
	svc, setup, user := setupSessionTest(t)

	raw, sess, err := svc.Create(user.ID, "127.0.0.1", "h", 0)
	require.NoError(t, err)

	// Backdate LastActivityAt to make the assertion meaningful.
	past := time.Now().UTC().Add(-1 * time.Hour)
	require.NoError(t, setup.DB.Model(sess).Update("last_activity_at", past).Error)

	svc.Touch(sess.ID)

	// Reload and verify LastActivityAt was refreshed.
	var updated model.UserSession
	require.NoError(t, setup.DB.Where("session_id_hash = ?", hashID(raw)).First(&updated).Error)
	assert.True(t, updated.LastActivityAt.After(past), "LastActivityAt should be refreshed")
}

// --- Revoke tests ----------------------------------------------------------

func TestRevoke(t *testing.T) {
	svc, setup, user := setupSessionTest(t)

	raw, _, err := svc.Create(user.ID, "127.0.0.1", "h", 0)
	require.NoError(t, err)

	require.NoError(t, svc.Revoke(raw))

	var sess model.UserSession
	require.NoError(t, setup.DB.Where("session_id_hash = ?", hashID(raw)).First(&sess).Error)
	assert.NotNil(t, sess.RevokedAt, "RevokedAt must be set after Revoke")
}

func TestRevoke_EmptyID(t *testing.T) {
	svc, _, _ := setupSessionTest(t)

	// Revoke with empty ID is a no-op, should not error.
	err := svc.Revoke("")
	assert.NoError(t, err)
}
