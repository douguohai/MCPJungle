package devicetoken

import (
	"errors"
	"sort"
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

type fakeEffectiveServices struct {
	services map[uint]struct{}
	err      error
}

func (f fakeEffectiveServices) EffectiveServiceIDs(userID uint) (map[uint]struct{}, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[uint]struct{}, len(f.services))
	for id := range f.services {
		out[id] = struct{}{}
	}
	return out, nil
}

func setupDeviceTokenTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testhelpers.RequireTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.McpServer{},
		&model.DeviceToken{},
		&model.DeviceTokenService{},
	))
	return db
}

func createDeviceTokenTestUser(t *testing.T, db *gorm.DB, username string, status types.UserStatus) *model.User {
	t.Helper()

	user := &model.User{
		Username:     username,
		DisplayName:  username,
		Role:         types.UserRoleMember,
		Status:       status,
		PasswordHash: "unused",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func createDeviceTokenTestServer(t *testing.T, db *gorm.DB, name string) *model.McpServer {
	t.Helper()

	server, err := model.NewSSEServer(name, name, "https://example.com/"+name, "", types.SessionModeStateless)
	require.NoError(t, err)
	require.NoError(t, db.Create(server).Error)
	return server
}

func TestCreateStoresOnlyHashedTokenAndDefaultExpiry(t *testing.T) {
	db := setupDeviceTokenTestDB(t)
	user := createDeviceTokenTestUser(t, db, "alice", types.UserStatusActive)
	server1 := createDeviceTokenTestServer(t, db, "svc-a")
	server2 := createDeviceTokenTestServer(t, db, "svc-b")
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)

	svc := &Service{
		db: db,
		effectiveServices: fakeEffectiveServices{
			services: map[uint]struct{}{server1.ID: {}, server2.ID: {}},
		},
		now: func() time.Time { return now },
	}

	token, plain, err := svc.Create(user.ID, CreateInput{Name: "Alice Laptop"})
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotEmpty(t, plain)

	assert.Equal(t, user.ID, token.UserID)
	assert.Equal(t, "Alice Laptop", token.Name)
	assert.Equal(t, types.DeviceTokenScopeInheritAll, token.Scope)
	assert.Equal(t, now.Add(deviceTokenDefaultTTL), token.ExpiresAt)
	assert.Equal(t, auth.Hash(plain), token.TokenHash)
	assert.NotEqual(t, plain, token.TokenHash)
	assert.Equal(t, auth.Prefix(plain), token.TokenPrefix)

	var stored model.DeviceToken
	require.NoError(t, db.First(&stored, token.ID).Error)
	assert.Equal(t, auth.Hash(plain), stored.TokenHash)
	assert.NotEqual(t, plain, stored.TokenHash)
	assert.Equal(t, auth.Prefix(plain), stored.TokenPrefix)
	assert.Equal(t, now.Add(deviceTokenDefaultTTL), stored.ExpiresAt)
}

func TestCreateRejectsInactiveUsersAndExpiryPastMax(t *testing.T) {
	db := setupDeviceTokenTestDB(t)
	pending := createDeviceTokenTestUser(t, db, "pending-user", types.UserStatusPending)
	disabled := createDeviceTokenTestUser(t, db, "disabled-user", types.UserStatusDisabled)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)

	svc := &Service{
		db:                db,
		effectiveServices: fakeEffectiveServices{},
		now:               func() time.Time { return now },
	}

	_, _, err := svc.Create(pending.ID, CreateInput{Name: "Pending Device"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrNotFound))

	_, _, err = svc.Create(disabled.ID, CreateInput{Name: "Disabled Device"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrNotFound))

	tooLate := now.Add(deviceTokenDefaultTTL + time.Minute)
	active := createDeviceTokenTestUser(t, db, "active-user", types.UserStatusActive)
	_, _, err = svc.Create(active.ID, CreateInput{Name: "Too Long", ExpiresAt: &tooLate})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrInvalidInput))
}

func TestCreateEnforcesLimitAndPerUserUniqueName(t *testing.T) {
	db := setupDeviceTokenTestDB(t)
	user := createDeviceTokenTestUser(t, db, "bob", types.UserStatusActive)
	other := createDeviceTokenTestUser(t, db, "carol", types.UserStatusActive)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)

	svc := &Service{
		db:                db,
		effectiveServices: fakeEffectiveServices{},
		now:               func() time.Time { return now },
	}

	for i := 0; i < deviceTokenMaxActive; i++ {
		_, _, err := svc.Create(user.ID, CreateInput{Name: "device-" + string(rune('a'+i))})
		require.NoError(t, err)
	}

	_, _, err := svc.Create(user.ID, CreateInput{Name: "overflow"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrInvalidInput))

	_, _, err = svc.Create(other.ID, CreateInput{Name: "shared-name"})
	require.NoError(t, err)
	_, _, err = svc.Create(other.ID, CreateInput{Name: "shared-name"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrInvalidInput))
}

func TestCreateRestrictedTokenRequiresSubsetOfEffectiveServices(t *testing.T) {
	db := setupDeviceTokenTestDB(t)
	user := createDeviceTokenTestUser(t, db, "dave", types.UserStatusActive)
	server1 := createDeviceTokenTestServer(t, db, "svc-a")
	server2 := createDeviceTokenTestServer(t, db, "svc-b")
	server3 := createDeviceTokenTestServer(t, db, "svc-c")

	svc := &Service{
		db: db,
		effectiveServices: fakeEffectiveServices{
			services: map[uint]struct{}{server1.ID: {}, server2.ID: {}},
		},
		now: func() time.Time { return time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC) },
	}

	token, _, err := svc.Create(user.ID, CreateInput{
		Name:       "Restricted",
		Scope:      types.DeviceTokenScopeRestricted,
		ServiceIDs: []uint{server1.ID, server1.ID, server2.ID},
	})
	require.NoError(t, err)

	var rows []model.DeviceTokenService
	require.NoError(t, db.Where("device_token_id = ?", token.ID).Order("mcp_server_id ASC").Find(&rows).Error)
	require.Len(t, rows, 2)
	assert.Equal(t, server1.ID, rows[0].McpServerID)
	assert.Equal(t, server2.ID, rows[1].McpServerID)

	_, _, err = svc.Create(user.ID, CreateInput{
		Name:       "Outside Scope",
		Scope:      types.DeviceTokenScopeRestricted,
		ServiceIDs: []uint{server3.ID},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrInvalidInput))
}

func TestListForUserReturnsOnlyOwnedTokens(t *testing.T) {
	db := setupDeviceTokenTestDB(t)
	alice := createDeviceTokenTestUser(t, db, "alice", types.UserStatusActive)
	bob := createDeviceTokenTestUser(t, db, "bob", types.UserStatusActive)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)

	svc := &Service{
		db:                db,
		effectiveServices: fakeEffectiveServices{},
		now:               func() time.Time { return now },
	}

	first, _, err := svc.Create(alice.ID, CreateInput{Name: "alice-1"})
	require.NoError(t, err)
	second, _, err := svc.Create(alice.ID, CreateInput{Name: "alice-2"})
	require.NoError(t, err)
	_, _, err = svc.Create(bob.ID, CreateInput{Name: "bob-1"})
	require.NoError(t, err)

	got, err := svc.ListForUser(alice.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)

	gotIDs := []uint{got[0].ID, got[1].ID}
	sort.Slice(gotIDs, func(i, j int) bool { return gotIDs[i] < gotIDs[j] })
	assert.Equal(t, []uint{first.ID, second.ID}, gotIDs)
}

func TestRevokeIsIdempotentAndChecksOwnership(t *testing.T) {
	db := setupDeviceTokenTestDB(t)
	alice := createDeviceTokenTestUser(t, db, "alice", types.UserStatusActive)
	bob := createDeviceTokenTestUser(t, db, "bob", types.UserStatusActive)
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)

	svc := &Service{
		db:                db,
		effectiveServices: fakeEffectiveServices{},
		now:               func() time.Time { return now },
	}

	token, _, err := svc.Create(alice.ID, CreateInput{Name: "alice-device"})
	require.NoError(t, err)

	err = svc.Revoke(bob.ID, token.ID, false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apierrors.ErrNotFound))

	require.NoError(t, svc.Revoke(alice.ID, token.ID, false))
	require.NoError(t, svc.Revoke(alice.ID, token.ID, false))

	var stored model.DeviceToken
	require.NoError(t, db.First(&stored, token.ID).Error)
	require.NotNil(t, stored.RevokedAt)
	assert.Equal(t, now, stored.RevokedAt.UTC())
}

func TestAuthenticateRejectsExpiredRevokedAndDisabledUser(t *testing.T) {
	db := setupDeviceTokenTestDB(t)
	active := createDeviceTokenTestUser(t, db, "active", types.UserStatusActive)
	disabled := createDeviceTokenTestUser(t, db, "disabled", types.UserStatusDisabled)
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	revokedAt := now.Add(-time.Minute)

	require.NoError(t, db.Create(&model.DeviceToken{
		UserID:      active.ID,
		Name:        "expired",
		TokenPrefix: "mcpdt_expire",
		TokenHash:   auth.Hash("expired-token"),
		Scope:       types.DeviceTokenScopeInheritAll,
		ExpiresAt:   now.Add(-time.Second),
	}).Error)
	require.NoError(t, db.Create(&model.DeviceToken{
		UserID:      active.ID,
		Name:        "revoked",
		TokenPrefix: "mcpdt_revoked",
		TokenHash:   auth.Hash("revoked-token"),
		Scope:       types.DeviceTokenScopeInheritAll,
		ExpiresAt:   now.Add(time.Hour),
		RevokedAt:   &revokedAt,
	}).Error)
	require.NoError(t, db.Create(&model.DeviceToken{
		UserID:      disabled.ID,
		Name:        "disabled",
		TokenPrefix: "mcpdt_disable",
		TokenHash:   auth.Hash("disabled-token"),
		Scope:       types.DeviceTokenScopeInheritAll,
		ExpiresAt:   now.Add(time.Hour),
	}).Error)

	svc := &Service{
		db:                db,
		effectiveServices: fakeEffectiveServices{},
		now:               func() time.Time { return now },
	}

	for _, token := range []string{"expired-token", "revoked-token", "disabled-token", "", "missing-token"} {
		user, deviceToken, effective, err := svc.Authenticate(token, "127.0.0.1", "tester")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apierrors.ErrInvalidCredentials))
		assert.Nil(t, user)
		assert.Nil(t, deviceToken)
		assert.Nil(t, effective)
	}
}

func TestAuthenticateRestrictedTokenIntersectsAndUpdatesUsage(t *testing.T) {
	db := setupDeviceTokenTestDB(t)
	user := createDeviceTokenTestUser(t, db, "erin", types.UserStatusActive)
	server1 := createDeviceTokenTestServer(t, db, "svc-a")
	server2 := createDeviceTokenTestServer(t, db, "svc-b")
	server3 := createDeviceTokenTestServer(t, db, "svc-c")
	now := time.Date(2026, 7, 20, 11, 0, 0, 0, time.UTC)

	token := &model.DeviceToken{
		UserID:      user.ID,
		Name:        "restricted",
		TokenPrefix: "mcpdt_restri",
		TokenHash:   auth.Hash("restricted-token"),
		Scope:       types.DeviceTokenScopeRestricted,
		ExpiresAt:   now.Add(time.Hour),
	}
	require.NoError(t, db.Create(token).Error)
	require.NoError(t, db.Create(&[]model.DeviceTokenService{
		{DeviceTokenID: token.ID, McpServerID: server1.ID},
		{DeviceTokenID: token.ID, McpServerID: server3.ID},
	}).Error)

	svc := &Service{
		db: db,
		effectiveServices: fakeEffectiveServices{
			services: map[uint]struct{}{server1.ID: {}, server2.ID: {}},
		},
		now: func() time.Time { return now },
	}

	gotUser, gotToken, effective, err := svc.Authenticate("restricted-token", " 10.0.0.8 ", " claude-desktop ")
	require.NoError(t, err)
	require.NotNil(t, gotUser)
	require.NotNil(t, gotToken)
	assert.Equal(t, user.ID, gotUser.ID)
	assert.Equal(t, token.ID, gotToken.ID)
	assert.Equal(t, map[uint]struct{}{server1.ID: {}}, effective)
	require.NotNil(t, gotToken.LastUsedAt)
	assert.Equal(t, now, gotToken.LastUsedAt.UTC())
	assert.Equal(t, "10.0.0.8", gotToken.LastUsedIP)
	assert.Equal(t, "claude-desktop", gotToken.ClientInfo)

	var stored model.DeviceToken
	require.NoError(t, db.First(&stored, token.ID).Error)
	require.NotNil(t, stored.LastUsedAt)
	assert.Equal(t, now, stored.LastUsedAt.UTC())
	assert.Equal(t, "10.0.0.8", stored.LastUsedIP)
	assert.Equal(t, "claude-desktop", stored.ClientInfo)
}
