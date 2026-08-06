// Package model provides data models for the MCPJungle application.
package model

import (
	"gorm.io/gorm"
)

// CallDailyAggregate stores pre-aggregated per-day call statistics.
// The composite unique index (Date, UserID, DeviceTokenID, McpServiceID,
// ToolName, Result) supports idempotent re-aggregation.
type CallDailyAggregate struct {
	gorm.Model

	// Date is the calendar day in YYYY-MM-DD (UTC).
	Date string `json:"date" gorm:"type:varchar(10);not null;uniqueIndex:idx_daily_agg,priority:1"`

	UserID        uint `json:"user_id" gorm:"not null;uniqueIndex:idx_daily_agg,priority:2"`
	DeviceTokenID uint `json:"device_token_id" gorm:"uniqueIndex:idx_daily_agg,priority:3"`
	McpServiceID  uint `json:"mcp_service_id" gorm:"not null;uniqueIndex:idx_daily_agg,priority:4"`

	// ToolName is the bare tool name; empty string means "all tools".
	ToolName string `json:"tool_name" gorm:"type:varchar(255);uniqueIndex:idx_daily_agg,priority:5"`

	// Result is one of the CallEventResult* constants; empty means "all results".
	Result string `json:"result" gorm:"type:varchar(30);uniqueIndex:idx_daily_agg,priority:6"`

	CallCount       int64 `json:"call_count"`
	TotalLatencyMs  int64 `json:"total_latency_ms"`
	MaxLatencyMs    int64 `json:"max_latency_ms"`
	TotalRequestBytes  int64 `json:"total_request_bytes"`
	TotalResponseBytes int64 `json:"total_response_bytes"`
}
