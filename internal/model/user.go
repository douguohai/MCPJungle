// Package model provides data models for the MCPJungle application.
package model

import (
	"encoding/json"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// User represents an authenticated, human user in enterprise mode.
// A user can be an admin or a regular user.
// There are no users if mcpjungle is running in development mode.
type User struct {
	gorm.Model

	Username     string         `json:"username" gorm:"uniqueIndex;not null;type:varchar(255)"`
	Role         types.UserRole `json:"role" gorm:"not null"`

	// PasswordHash stores the bcrypt hash of the user's password and is the
	// primary credential for dashboard login (exchanged for a short-lived JWT).
	PasswordHash string `json:"-" gorm:"type:varchar(255)"`

	// AccessToken is a legacy long-lived token retained for backward
	// compatibility during the migration to password+JWT auth. It is no longer
	// issued by init-server and is not used for new logins.
	AccessToken string `json:"access_token,omitempty" gorm:"uniqueIndex;type:varchar(255)"`

	// AllowedServers restricts which MCP servers this user's clients may access.
	// Empty/nil or ["*"] means all servers (default). Set by admins to implement
	// "these MCPs are only available to these people".
	AllowedServers datatypes.JSON `json:"allowed_servers,omitempty" gorm:"type:jsonb"`
}

// CheckAllowedServer reports whether the user is permitted to access the given
// MCP server. Empty AllowedServers or a "*" entry means all servers (default,
// fail-open on parse error so a malformed config doesn't lock everyone out).
func (u *User) CheckAllowedServer(serverName string) bool {
	if len(u.AllowedServers) == 0 {
		return true
	}
	var servers []string
	if err := json.Unmarshal(u.AllowedServers, &servers); err != nil {
		return true
	}
	if len(servers) == 0 {
		return true
	}
	for _, s := range servers {
		if s == "*" || s == serverName {
			return true
		}
	}
	return false
}

// AllowedServerNames returns the user's AllowedServers as a string slice
// (empty/nil means all servers).
func (u *User) AllowedServerNames() []string {
	var names []string
	_ = json.Unmarshal(u.AllowedServers, &names)
	return names
}
