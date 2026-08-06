// Package model provides data models for the MCPJungle application.
package model

import (
	"time"

	"gorm.io/gorm"
)

// AuditEvent records a significant administrative action for compliance.
// Audit events are append-only; the API never offers update or delete.
type AuditEvent struct {
	gorm.Model

	// ActorUserID is the user who performed the action (0 for system).
	ActorUserID uint `json:"actor_user_id" gorm:"not null;index"`

	// ActionType is a machine-readable identifier, e.g. "user.create",
	// "token.revoke", "server.enable".
	ActionType string `json:"action_type" gorm:"type:varchar(100);not null;index"`

	// TargetType is the kind of object acted upon, e.g. "user", "device_token",
	// "mcp_server".
	TargetType string `json:"target_type" gorm:"type:varchar(50);not null"`

	// TargetID is the primary key (as string) of the target object.
	TargetID string `json:"target_id" gorm:"type:varchar(64);not null"`

	// OccurredAt is the UTC timestamp of the action.
	OccurredAt time.Time `json:"occurred_at" gorm:"not null;index"`

	// SourceIP is the IP address of the actor.
	SourceIP string `json:"source_ip" gorm:"type:varchar(64)"`

	// Result is "success" or "failure".
	Result string `json:"result" gorm:"type:varchar(20);not null"`

	// ChangeSummary is a human-readable, sanitized description of the change.
	// No raw passwords, tokens, or PII should appear here.
	ChangeSummary string `json:"change_summary" gorm:"type:text"`
}
