package user

import (
	"errors"
	"testing"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/auth"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func userServiceTestDB(t *testing.T) *UserService {
	db := testhelpers.RequireTestDB(t)
	if err := db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.DeviceToken{}); err != nil {
		t.Fatalf("migrate user service models: %v", err)
	}
	return NewUserService(db)
}

func TestCreateUserReturnsOneTimePassword(t *testing.T) {
	svc := userServiceTestDB(t)
	account, plain, err := svc.Create(CreateUserInput{Username: "alice", DisplayName: "Alice"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if account.Status != types.UserStatusPending || account.Role != types.UserRoleMember || !account.MustChangePassword {
		t.Fatalf("unexpected new account state: %+v", account)
	}
	if plain == "" || account.PasswordHash == "" || plain == account.PasswordHash {
		t.Fatal("initial password must be returned once and stored only as a hash")
	}
	if _, err := svc.VerifyPassword("alice", plain); err != nil {
		t.Fatalf("verify initial password: %v", err)
	}
}

func TestChangePasswordActivatesPendingUser(t *testing.T) {
	svc := userServiceTestDB(t)
	account, initial, err := svc.Create(CreateUserInput{Username: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	oldSession := model.UserSession{UserID: account.ID, TokenHash: auth.Hash("old-session"), ExpiresAt: now.Add(time.Hour)}
	if err := svc.db.Create(&oldSession).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangePassword(account.ID, initial, "a-new-password-123"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	updated, err := svc.GetUserByID(account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != types.UserStatusActive || updated.MustChangePassword {
		t.Fatalf("pending account was not activated: %+v", updated)
	}
	if _, err := svc.VerifyPassword("bob", initial); !errors.Is(err, apierrors.ErrInvalidCredentials) {
		t.Fatalf("old password should be rejected, got %v", err)
	}
	if err := svc.db.First(&oldSession, oldSession.ID).Error; err != nil || oldSession.RevokedAt == nil {
		t.Fatalf("password change did not revoke old session: %+v, %v", oldSession, err)
	}
}

func TestDisableUserRevokesAllCredentials(t *testing.T) {
	svc := userServiceTestDB(t)
	admin, err := svc.CreateAdminUser("admin", "admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	account, initial, err := svc.Create(CreateUserInput{Username: "carol"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangePassword(account.ID, initial, "carol-password-123"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	session := model.UserSession{UserID: account.ID, TokenHash: auth.Hash("session"), ExpiresAt: now.Add(time.Hour), LastSeenAt: &now}
	token := model.DeviceToken{UserID: account.ID, Name: "laptop", TokenHash: auth.Hash("device"), TokenPrefix: "mjd_example", Scope: types.DeviceTokenScopeInheritAll, ExpiresAt: now.Add(time.Hour)}
	if err := svc.db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Create(&token).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.SetStatus(admin.ID, account.ID, types.UserStatusDisabled); err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if err := svc.db.First(&session, session.ID).Error; err != nil || session.RevokedAt == nil {
		t.Fatalf("session not revoked: %+v, %v", session, err)
	}
	if err := svc.db.First(&token, token.ID).Error; err != nil || token.RevokedAt == nil {
		t.Fatalf("device token not revoked: %+v, %v", token, err)
	}
	if _, err := svc.VerifyPassword("carol", "carol-password-123"); !errors.Is(err, apierrors.ErrInvalidCredentials) {
		t.Fatalf("disabled account should not authenticate, got %v", err)
	}
}

func TestLastActiveSystemAdminCannotBeDisabledOrDemoted(t *testing.T) {
	svc := userServiceTestDB(t)
	admin, err := svc.CreateAdminUser("admin", "admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetStatus(admin.ID, admin.ID, types.UserStatusDisabled); !errors.Is(err, apierrors.ErrInvalidInput) {
		t.Fatalf("expected last-admin guard, got %v", err)
	}
	if err := svc.UpdateRole(admin.ID, admin.ID, types.UserRoleMember); !errors.Is(err, apierrors.ErrInvalidInput) {
		t.Fatalf("expected last-admin role guard, got %v", err)
	}
	displayName := "Changed despite failed role"
	memberRole := types.UserRoleMember
	if _, err := svc.UpdateUser(admin.ID, admin.ID, UpdateUserInput{DisplayName: &displayName, Role: &memberRole}); !errors.Is(err, apierrors.ErrInvalidInput) {
		t.Fatalf("expected atomic last-admin guard, got %v", err)
	}
	stored, err := svc.GetUserByID(admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.DisplayName != "admin" {
		t.Fatalf("failed role update partially changed display name: %q", stored.DisplayName)
	}
}

func TestPasswordMinimumLength(t *testing.T) {
	svc := userServiceTestDB(t)
	if _, err := svc.CreateAdminUser("admin", "short"); !errors.Is(err, apierrors.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
