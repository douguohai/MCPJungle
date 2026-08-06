package types

// UserRole represents the role of a user in the MCPJungle system.
type UserRole string

const (
	UserRoleSystemAdmin  UserRole = "system_admin"
	UserRoleServiceAdmin UserRole = "service_admin"
	UserRoleMember       UserRole = "member"
	UserRoleAuditor      UserRole = "auditor"
)

// User represents an authenticated, human user in mcpjungle.
type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type CreateOrUpdateUserRequest struct {
	Username string `json:"username"`
}

type CreateOrUpdateUserResponse struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}
