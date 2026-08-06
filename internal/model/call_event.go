// Package model provides data models for the MCPJungle application.
package model

import (
	"time"

	"gorm.io/gorm"
)

// CallEvent result constants (design doc §10).
const (
	CallEventResultSuccess          = "success"
	CallEventResultUpstreamError    = "upstream_error"
	CallEventResultTimeout          = "timeout"
	CallEventResultPermissionDenied = "permission_denied"
	CallEventResultServiceDisabled  = "service_disabled"
)

// CallEvent records a single MCP tool/resource/prompt call for analytics.
// Arguments, return content, and request headers are never stored.
// Events are retained for 90 days by default and then purged.
type CallEvent struct {
	gorm.Model

	// RequestID is a unique identifier for this call (UUID).
	RequestID string `json:"request_id" gorm:"type:varchar(36);uniqueIndex;not null"`

	// CallTime is the UTC timestamp when the call was initiated.
	CallTime time.Time `json:"call_time" gorm:"not null;index"`

	UserID        uint `json:"user_id" gorm:"not null;index"`
	DeviceTokenID uint `json:"device_token_id" gorm:"index"`

	// McpServiceID is the upstream MCP server that handled the call.
	McpServiceID uint `json:"mcp_service_id" gorm:"not null;index"`

	// ToolName is the bare tool/prompt/resource name (without server prefix).
	ToolName string `json:"tool_name" gorm:"type:varchar(255);not null"`

	// CallType is "tool", "prompt", or "resource".
	CallType string `json:"call_type" gorm:"type:varchar(20);not null"`

	// Result is one of the CallEventResult* constants.
	Result string `json:"result" gorm:"type:varchar(30);not null;index"`

	// LatencyMs is the total wall-clock latency in milliseconds.
	LatencyMs int64 `json:"latency_ms"`

	// UpstreamLatencyMs measures only the upstream call (excluding proxy overhead).
	UpstreamLatencyMs int64 `json:"upstream_latency_ms"`

	// RequestBytes / ResponseBytes are optional payload sizes.
	RequestBytes  int `json:"request_bytes"`
	ResponseBytes int `json:"response_bytes"`

	// ErrorCode is a machine-readable error code (empty on success).
	ErrorCode string `json:"error_code" gorm:"type:varchar(64)"`

	// SourceIP is the client IP address (best-effort).
	SourceIP string `json:"source_ip" gorm:"type:varchar(64)"`

	// ClientID is an optional client-provided identifier (from device token).
	ClientID string `json:"client_id" gorm:"type:varchar(255)"`
}
