// Package auditevent manages AuditEvent records – an append-only log of
// significant administrative actions.  The API never exposes update or delete
// operations on these records.
package auditevent

import (
	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

// Service manages AuditEvent records.
type Service struct {
	db *gorm.DB
}

// NewService creates a new AuditEvent service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// RecordEvent persists a single audit event.  Only append – no update/delete.
func (s *Service) RecordEvent(event *model.AuditEvent) error {
	return s.db.Create(event).Error
}

// ListEventsQuery defines filter criteria for listing audit events.
type ListEventsQuery struct {
	ActorID    uint   // 0 = no filter
	ActionType string // "" = no filter
	TargetType string // "" = no filter
	DateFrom   string // YYYY-MM-DD inclusive
	DateTo     string // YYYY-MM-DD inclusive
	Limit      int    // 0 = use default (200)
}

// ListEvents returns audit events matching the given filters, newest first.
func (s *Service) ListEvents(q ListEventsQuery) ([]model.AuditEvent, error) {
	tx := s.db.Model(&model.AuditEvent{})
	if q.ActorID > 0 {
		tx = tx.Where("actor_user_id = ?", q.ActorID)
	}
	if q.ActionType != "" {
		tx = tx.Where("action_type = ?", q.ActionType)
	}
	if q.TargetType != "" {
		tx = tx.Where("target_type = ?", q.TargetType)
	}
	if q.DateFrom != "" {
		tx = tx.Where("occurred_at >= ?", q.DateFrom+"T00:00:00")
	}
	if q.DateTo != "" {
		tx = tx.Where("occurred_at < ?", q.DateTo+"T23:59:59")
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}

	var events []model.AuditEvent
	err := tx.Order("occurred_at DESC").Limit(limit).Find(&events).Error
	return events, err
}
