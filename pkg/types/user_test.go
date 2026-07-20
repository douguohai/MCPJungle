package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserRoleConstants(t *testing.T) {
	t.Parallel()

	require.Equal(t, UserRole("system_admin"), UserRoleSystemAdmin)
	require.Equal(t, UserRole("service_admin"), UserRoleServiceAdmin)
	require.Equal(t, UserRole("member"), UserRoleMember)
	require.Equal(t, UserRole("auditor"), UserRoleAuditor)
}

func TestIsValidUserRole(t *testing.T) {
	t.Parallel()

	require.True(t, IsValidUserRole(UserRoleSystemAdmin))
	require.True(t, IsValidUserRole(UserRoleServiceAdmin))
	require.True(t, IsValidUserRole(UserRoleMember))
	require.True(t, IsValidUserRole(UserRoleAuditor))
	require.False(t, IsValidUserRole(UserRole("owner")))
}

func TestUserStatusConstants(t *testing.T) {
	t.Parallel()

	require.Equal(t, UserStatus("pending"), UserStatusPending)
	require.Equal(t, UserStatus("active"), UserStatusActive)
	require.Equal(t, UserStatus("disabled"), UserStatusDisabled)
}

func TestIsValidUserStatus(t *testing.T) {
	t.Parallel()

	require.True(t, IsValidUserStatus(UserStatusPending))
	require.True(t, IsValidUserStatus(UserStatusActive))
	require.True(t, IsValidUserStatus(UserStatusDisabled))
	require.False(t, IsValidUserStatus(UserStatus("revoked")))
}

func TestDeviceTokenScopeConstants(t *testing.T) {
	t.Parallel()

	require.Equal(t, DeviceTokenScope("inherit_all"), DeviceTokenScopeInheritAll)
	require.Equal(t, DeviceTokenScope("restricted"), DeviceTokenScopeRestricted)
}

func TestUserJSONMarshaling(t *testing.T) {
	t.Parallel()

	user := User{
		ID:                 7,
		Username:           "alice",
		DisplayName:        "Alice",
		Role:               string(UserRoleMember),
		Status:             string(UserStatusActive),
		MustChangePassword: true,
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"id":7,
		"username":"alice",
		"display_name":"Alice",
		"role":"member",
		"status":"active",
		"must_change_password":true
	}`, string(data))
}
