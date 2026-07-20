package session

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/auth"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSessionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testhelpers.RequireTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}))
	return db
}

func createSessionTestUser(t *testing.T, db *gorm.DB, username string, status types.UserStatus) *model.User {
	t.Helper()

	user := &model.User{
		Username:     username,
		DisplayName:  username,
		Role:         "member",
		Status:       status,
		PasswordHash: "unused",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func createStoredSession(t *testing.T, db *gorm.DB, userID uint, token string, expiresAt time.Time, revokedAt *time.Time) *model.UserSession {
	t.Helper()

	session := &model.UserSession{
		UserID:    userID,
		TokenHash: auth.Hash(token),
		ExpiresAt: expiresAt,
		RevokedAt: revokedAt,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}
	require.NoError(t, db.Create(session).Error)
	return session
}

func TestCreateStoresHashedSession(t *testing.T) {
	db := setupSessionTestDB(t)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	user := createSessionTestUser(t, db, "alice", activeUserStatus)

	svc := &Service{
		db:       db,
		now:      func() time.Time { return now },
		lifetime: sessionLifetime,
	}

	plain, session, err := svc.Create(user.ID, " 127.0.0.1 ", " browser/1.0 ")
	require.NoError(t, err)
	require.NotNil(t, session)
	require.NotEmpty(t, plain)

	assert.Equal(t, user.ID, session.UserID)
	assert.Equal(t, auth.Hash(plain), session.TokenHash)
	assert.NotEqual(t, plain, session.TokenHash)
	assert.Equal(t, now.Add(8*time.Hour), session.ExpiresAt)
	assert.Equal(t, "127.0.0.1", session.IPAddress)
	assert.Equal(t, "browser/1.0", session.UserAgent)
	assert.Nil(t, session.RevokedAt)
	assert.Nil(t, session.LastSeenAt)
	assert.True(t, strings.HasPrefix(plain, sessionTokenPrefix))

	var stored model.UserSession
	require.NoError(t, db.First(&stored, session.ID).Error)
	assert.Equal(t, auth.Hash(plain), stored.TokenHash)
	assert.NotEqual(t, plain, stored.TokenHash)
	assert.Equal(t, now.Add(8*time.Hour), stored.ExpiresAt)
}

func TestAuthenticateReturnsUserAndUpdatesLastSeen(t *testing.T) {
	db := setupSessionTestDB(t)
	createdAt := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	authenticatedAt := createdAt.Add(45 * time.Minute)
	user := createSessionTestUser(t, db, "bob", activeUserStatus)

	svc := &Service{
		db:       db,
		now:      func() time.Time { return createdAt },
		lifetime: sessionLifetime,
	}
	plain, createdSession, err := svc.Create(user.ID, "127.0.0.1", "browser")
	require.NoError(t, err)

	svc.now = func() time.Time { return authenticatedAt }

	gotUser, gotSession, err := svc.Authenticate(plain)
	require.NoError(t, err)
	require.NotNil(t, gotUser)
	require.NotNil(t, gotSession)

	assert.Equal(t, user.ID, gotUser.ID)
	assert.Equal(t, createdSession.ID, gotSession.ID)
	require.NotNil(t, gotSession.LastSeenAt)
	assert.Equal(t, authenticatedAt, gotSession.LastSeenAt.UTC())

	var stored model.UserSession
	require.NoError(t, db.First(&stored, createdSession.ID).Error)
	require.NotNil(t, stored.LastSeenAt)
	assert.Equal(t, authenticatedAt, stored.LastSeenAt.UTC())
}

func TestAuthenticateRejectsInvalidSessions(t *testing.T) {
	db := setupSessionTestDB(t)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	activeUser := createSessionTestUser(t, db, "carol", activeUserStatus)
	disabledUser := createSessionTestUser(t, db, "dave", types.UserStatusDisabled)

	revokedAt := now.Add(-10 * time.Minute)
	createStoredSession(t, db, activeUser.ID, "expired-token", now.Add(-time.Minute), nil)
	createStoredSession(t, db, activeUser.ID, "revoked-token", now.Add(time.Hour), &revokedAt)
	createStoredSession(t, db, disabledUser.ID, "disabled-token", now.Add(time.Hour), nil)

	svc := &Service{
		db:       db,
		now:      func() time.Time { return now },
		lifetime: sessionLifetime,
	}

	tests := []struct {
		name  string
		token string
	}{
		{name: "missing", token: "missing-token"},
		{name: "expired", token: "expired-token"},
		{name: "revoked", token: "revoked-token"},
		{name: "disabled-user", token: "disabled-token"},
		{name: "empty", token: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, session, err := svc.Authenticate(tt.token)
			require.Error(t, err)
			assert.True(t, errors.Is(err, apierrors.ErrInvalidCredentials))
			assert.Nil(t, user)
			assert.Nil(t, session)
		})
	}
}

func TestRevokeOnlyRevokesTargetSession(t *testing.T) {
	db := setupSessionTestDB(t)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	user := createSessionTestUser(t, db, "erin", activeUserStatus)

	first := createStoredSession(t, db, user.ID, "first-token", now.Add(time.Hour), nil)
	second := createStoredSession(t, db, user.ID, "second-token", now.Add(time.Hour), nil)

	svc := &Service{
		db:       db,
		now:      func() time.Time { return now },
		lifetime: sessionLifetime,
	}

	require.NoError(t, svc.Revoke(first.ID))

	var firstStored model.UserSession
	require.NoError(t, db.First(&firstStored, first.ID).Error)
	require.NotNil(t, firstStored.RevokedAt)
	assert.Equal(t, now, firstStored.RevokedAt.UTC())

	var secondStored model.UserSession
	require.NoError(t, db.First(&secondStored, second.ID).Error)
	assert.Nil(t, secondStored.RevokedAt)
}

func TestRevokeAllForUserRevokesOnlyThatUsersSessions(t *testing.T) {
	db := setupSessionTestDB(t)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	targetUser := createSessionTestUser(t, db, "frank", activeUserStatus)
	otherUser := createSessionTestUser(t, db, "grace", activeUserStatus)

	first := createStoredSession(t, db, targetUser.ID, "first-token", now.Add(time.Hour), nil)
	second := createStoredSession(t, db, targetUser.ID, "second-token", now.Add(time.Hour), nil)
	third := createStoredSession(t, db, otherUser.ID, "third-token", now.Add(time.Hour), nil)

	svc := &Service{
		db:       db,
		now:      func() time.Time { return now },
		lifetime: sessionLifetime,
	}

	require.NoError(t, svc.RevokeAllForUser(targetUser.ID))

	var firstStored model.UserSession
	require.NoError(t, db.First(&firstStored, first.ID).Error)
	require.NotNil(t, firstStored.RevokedAt)
	assert.Equal(t, now, firstStored.RevokedAt.UTC())

	var secondStored model.UserSession
	require.NoError(t, db.First(&secondStored, second.ID).Error)
	require.NotNil(t, secondStored.RevokedAt)
	assert.Equal(t, now, secondStored.RevokedAt.UTC())

	var thirdStored model.UserSession
	require.NoError(t, db.First(&thirdStored, third.ID).Error)
	assert.Nil(t, thirdStored.RevokedAt)
}
