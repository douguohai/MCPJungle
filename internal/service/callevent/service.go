// Package callevent manages CallEvent records (MCP call audit trail) and
// CallDailyAggregate roll-ups.  Writing a CallEvent is best-effort: errors
// are logged but never propagated to the caller so that analytics never
// blocks an MCP tool/resource/prompt call.
package callevent

import (
	"fmt"
	"log"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service manages CallEvent and CallDailyAggregate records.
type Service struct {
	db *gorm.DB
}

// NewService creates a new CallEvent service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// RecordEvent persists a single call event.  Callers should treat this as
// best-effort: if the write fails the error is returned for logging but
// should never block the upstream MCP call.
func (s *Service) RecordEvent(event *model.CallEvent) error {
	return s.db.Create(event).Error
}

// AggregateDaily aggregates all CallEvent rows for the given UTC date
// (YYYY-MM-DD) into CallDailyAggregate rows.  The function is idempotent:
// it uses an UPSERT so re-running for the same date overwrites existing
// aggregates without creating duplicates.
func (s *Service) AggregateDaily(date string) error {
	// Parse date boundaries.
	dayStart, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date %q: %w", date, err)
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	// Query aggregation grouped by (user_id, device_token_id, mcp_service_id,
	// tool_name, result).
	type aggRow struct {
		UserID          uint
		DeviceTokenID   uint
		McpServiceID    uint
		ToolName        string
		Result          string
		CallCount       int64
		TotalLatencyMs  int64
		MaxLatencyMs    int64
		TotalReqBytes   int64
		TotalRespBytes  int64
	}

	var rows []aggRow
	err = s.db.Model(&model.CallEvent{}).
		Where("call_time >= ? AND call_time < ?", dayStart, dayEnd).
		Select(
			"user_id",
			"device_token_id",
			"mcp_service_id",
			"tool_name",
			"result",
			"COUNT(*)             AS call_count",
			"COALESCE(SUM(latency_ms), 0)   AS total_latency_ms",
			"COALESCE(MAX(latency_ms), 0)   AS max_latency_ms",
			"COALESCE(SUM(request_bytes), 0)  AS total_req_bytes",
			"COALESCE(SUM(response_bytes), 0) AS total_resp_bytes",
		).
		Group("user_id, device_token_id, mcp_service_id, tool_name, result").
		Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("aggregate query: %w", err)
	}

	// UPSERT each aggregated row.
	for _, r := range rows {
		agg := model.CallDailyAggregate{
			Date:              date,
			UserID:            r.UserID,
			DeviceTokenID:     r.DeviceTokenID,
			McpServiceID:      r.McpServiceID,
			ToolName:          r.ToolName,
			Result:            r.Result,
			CallCount:         r.CallCount,
			TotalLatencyMs:    r.TotalLatencyMs,
			MaxLatencyMs:      r.MaxLatencyMs,
			TotalRequestBytes: r.TotalReqBytes,
			TotalResponseBytes: r.TotalRespBytes,
		}
		err = s.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "date"},
				{Name: "user_id"},
				{Name: "device_token_id"},
				{Name: "mcp_service_id"},
				{Name: "tool_name"},
				{Name: "result"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"call_count",
				"total_latency_ms",
				"max_latency_ms",
				"total_request_bytes",
				"total_response_bytes",
				"updated_at",
			}),
		}).Create(&agg).Error
		if err != nil {
			return fmt.Errorf("upsert aggregate: %w", err)
		}
	}
	return nil
}

// ListEventsQuery defines filter criteria for listing call events.
type ListEventsQuery struct {
	UserID     uint   // 0 = no filter
	ServerID   uint   // 0 = no filter
	ToolName   string // "" = no filter
	DateFrom   string // YYYY-MM-DD inclusive; "" = no lower bound
	DateTo     string // YYYY-MM-DD inclusive; "" = no upper bound
	Result     string // "" = no filter
	CallType   string // "" = no filter
	Limit      int    // 0 = use default (200)
}

// ListEvents returns call events matching the given filters, ordered newest first.
func (s *Service) ListEvents(q ListEventsQuery) ([]model.CallEvent, error) {
	tx := s.db.Model(&model.CallEvent{})
	if q.UserID > 0 {
		tx = tx.Where("user_id = ?", q.UserID)
	}
	if q.ServerID > 0 {
		tx = tx.Where("mcp_service_id = ?", q.ServerID)
	}
	if q.ToolName != "" {
		tx = tx.Where("tool_name = ?", q.ToolName)
	}
	if q.DateFrom != "" {
		tx = tx.Where("call_time >= ?", q.DateFrom+"T00:00:00")
	}
	if q.DateTo != "" {
		tx = tx.Where("call_time < ?", q.DateTo+"T23:59:59")
	}
	if q.Result != "" {
		tx = tx.Where("result = ?", q.Result)
	}
	if q.CallType != "" {
		tx = tx.Where("call_type = ?", q.CallType)
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}

	var events []model.CallEvent
	err := tx.Order("call_time DESC").Limit(limit).Find(&events).Error
	return events, err
}

// ListDailyAggregatesQuery defines filter criteria for listing daily aggregates.
type ListDailyAggregatesQuery struct {
	UserID    uint   // 0 = no filter
	ServerID  uint   // 0 = no filter
	DateFrom  string // YYYY-MM-DD inclusive
	DateTo    string // YYYY-MM-DD inclusive
	Limit     int    // 0 = use default (90)
}

// ListDailyAggregates returns daily aggregates matching the given filters.
func (s *Service) ListDailyAggregates(q ListDailyAggregatesQuery) ([]model.CallDailyAggregate, error) {
	tx := s.db.Model(&model.CallDailyAggregate{})
	if q.UserID > 0 {
		tx = tx.Where("user_id = ?", q.UserID)
	}
	if q.ServerID > 0 {
		tx = tx.Where("mcp_service_id = ?", q.ServerID)
	}
	if q.DateFrom != "" {
		tx = tx.Where("date >= ?", q.DateFrom)
	}
	if q.DateTo != "" {
		tx = tx.Where("date <= ?", q.DateTo)
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 90
	}

	var aggs []model.CallDailyAggregate
	err := tx.Order("date DESC").Limit(limit).Find(&aggs).Error
	return aggs, err
}

// CleanupOldEvents deletes CallEvent rows older than retentionDays.
func (s *Service) CleanupOldEvents(retentionDays int) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	result := s.db.Where("call_time < ?", cutoff).Delete(&model.CallEvent{})
	if result.Error != nil {
		return result.Error
	}
	log.Printf("[callevent] cleaned up %d call events older than %d days", result.RowsAffected, retentionDays)
	return nil
}

// CleanupOldAggregates deletes CallDailyAggregate rows older than retentionDays.
func (s *Service) CleanupOldAggregates(retentionDays int) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays).Format("2006-01-02")
	result := s.db.Where("date < ?", cutoff).Delete(&model.CallDailyAggregate{})
	if result.Error != nil {
		return result.Error
	}
	log.Printf("[callevent] cleaned up %d daily aggregates older than %d days", result.RowsAffected, retentionDays)
	return nil
}
