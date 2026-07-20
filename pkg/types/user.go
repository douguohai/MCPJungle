package types

// UserRole represents the role of a user in the MCPJungle system.
type UserRole string

const (
	UserRoleSystemAdmin  UserRole = "system_admin"
	UserRoleServiceAdmin UserRole = "service_admin"
	UserRoleMember       UserRole = "member"
	UserRoleAuditor      UserRole = "auditor"
)

// IsValidUserRole reports whether role is one of the supported internal
// account roles.
func IsValidUserRole(role UserRole) bool {
	switch role {
	case UserRoleSystemAdmin, UserRoleServiceAdmin, UserRoleMember, UserRoleAuditor:
		return true
	default:
		return false
	}
}

// UserStatus represents the lifecycle status of an internal account.
type UserStatus string

const (
	UserStatusPending  UserStatus = "pending"
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

// IsValidUserStatus reports whether status is one of the supported account
// lifecycle states.
func IsValidUserStatus(status UserStatus) bool {
	switch status {
	case UserStatusPending, UserStatusActive, UserStatusDisabled:
		return true
	default:
		return false
	}
}

// DeviceTokenScope describes how a personal device token narrows a user's
// effective service permissions.
type DeviceTokenScope string

const (
	DeviceTokenScopeInheritAll DeviceTokenScope = "inherit_all"
	DeviceTokenScopeRestricted DeviceTokenScope = "restricted"
)

// User represents an authenticated, human user in mcpjungle.
type User struct {
	ID                 uint   `json:"id,omitempty"`
	Username           string `json:"username"`
	DisplayName        string `json:"display_name,omitempty"`
	Role               string `json:"role"`
	Status             string `json:"status,omitempty"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
}
