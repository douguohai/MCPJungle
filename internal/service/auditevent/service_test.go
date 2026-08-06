package auditevent

import (
	"testing"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := testhelpers.CreateTestDB(t)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AuditEvent{}))
	return db
}

func TestRecordAndListEvents(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	now := time.Now().UTC()

	event := &model.AuditEvent{
		ActorUserID:   1,
		ActionType:    "user.create",
		TargetType:    "user",
		TargetID:      "42",
		OccurredAt:    now,
		SourceIP:      "10.0.0.1",
		Result:        "success",
		ChangeSummary: "Created user alice (role=member)",
	}

	err := svc.RecordEvent(event)
	require.NoError(t, err)
	assert.NotZero(t, event.ID)

	// List by actor
	events, err := svc.ListEvents(ListEventsQuery{ActorID: 1})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "user.create", events[0].ActionType)
	assert.Equal(t, "Created user alice (role=member)", events[0].ChangeSummary)

	// List by action type
	events, err = svc.ListEvents(ListEventsQuery{ActionType: "user.create"})
	require.NoError(t, err)
	require.Len(t, events, 1)

	// List by target type
	events, err = svc.ListEvents(ListEventsQuery{TargetType: "user"})
	require.NoError(t, err)
	require.Len(t, events, 1)

	// Non-matching actor
	events, err = svc.ListEvents(ListEventsQuery{ActorID: 999})
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestListEventsDateFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Insert an event from 5 days ago.
	require.NoError(t, svc.RecordEvent(&model.AuditEvent{
		ActorUserID: 1,
		ActionType:  "token.revoke",
		TargetType:  "device_token",
		TargetID:    "7",
		OccurredAt:  time.Now().UTC().AddDate(0, 0, -5),
		Result:      "success",
	}))

	// Insert a recent event.
	require.NoError(t, svc.RecordEvent(&model.AuditEvent{
		ActorUserID: 1,
		ActionType:  "server.enable",
		TargetType:  "mcp_server",
		TargetID:    "3",
		OccurredAt:  time.Now().UTC(),
		Result:      "success",
	}))

	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	// Should only return the recent event.
	events, err := svc.ListEvents(ListEventsQuery{DateFrom: yesterday, DateTo: today})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "server.enable", events[0].ActionType)
}

func TestMultipleEventsOrdering(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	for i := 0; i < 5; i++ {
		require.NoError(t, svc.RecordEvent(&model.AuditEvent{
			ActorUserID: 1,
			ActionType:  "user.create",
			TargetType:  "user",
			TargetID:    string(rune('A' + i)),
			OccurredAt:  time.Now().UTC().Add(time.Duration(i) * time.Second),
			Result:      "success",
		}))
	}

	events, err := svc.ListEvents(ListEventsQuery{ActorID: 1, Limit: 10})
	require.NoError(t, err)
	require.Len(t, events, 5)

	// Should be ordered newest first.
	for i := 1; i < len(events); i++ {
		assert.True(t, events[i-1].OccurredAt.After(events[i].OccurredAt) ||
			events[i-1].OccurredAt.Equal(events[i].OccurredAt),
			"events should be ordered newest first")
	}
}
