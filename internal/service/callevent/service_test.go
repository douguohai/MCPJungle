package callevent

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
	require.NoError(t, db.AutoMigrate(
		&model.CallEvent{},
		&model.CallDailyAggregate{},
	))
	return db
}

func TestRecordAndListEvents(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	now := time.Now().UTC()
	event := &model.CallEvent{
		RequestID:     "req-001",
		CallTime:      now,
		UserID:        1,
		DeviceTokenID: 10,
		McpServiceID:  2,
		ToolName:      "test_tool",
		CallType:      "tool",
		Result:        model.CallEventResultSuccess,
		LatencyMs:     150,
		ClientID:      "claude-desktop",
	}

	// Record
	err := svc.RecordEvent(event)
	require.NoError(t, err)
	assert.NotZero(t, event.ID)

	// List by user
	events, err := svc.ListEvents(ListEventsQuery{UserID: 1})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "req-001", events[0].RequestID)
	assert.Equal(t, "test_tool", events[0].ToolName)

	// List by server
	events, err = svc.ListEvents(ListEventsQuery{ServerID: 2})
	require.NoError(t, err)
	require.Len(t, events, 1)

	// List by result
	events, err = svc.ListEvents(ListEventsQuery{Result: model.CallEventResultSuccess})
	require.NoError(t, err)
	require.Len(t, events, 1)

	// List by different user -- empty
	events, err = svc.ListEvents(ListEventsQuery{UserID: 999})
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestAggregateDaily(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	today := time.Now().UTC().Format("2006-01-02")
	now := time.Now().UTC()

	// Insert two events for the same user/server/tool but different results.
	require.NoError(t, svc.RecordEvent(&model.CallEvent{
		RequestID:    "agg-001",
		CallTime:     now,
		UserID:       1,
		McpServiceID: 2,
		ToolName:     "tool_a",
		Result:       model.CallEventResultSuccess,
		LatencyMs:    100,
	}))
	require.NoError(t, svc.RecordEvent(&model.CallEvent{
		RequestID:    "agg-002",
		CallTime:     now,
		UserID:       1,
		McpServiceID: 2,
		ToolName:     "tool_a",
		Result:       model.CallEventResultSuccess,
		LatencyMs:    200,
	}))
	require.NoError(t, svc.RecordEvent(&model.CallEvent{
		RequestID:    "agg-003",
		CallTime:     now,
		UserID:       1,
		McpServiceID: 2,
		ToolName:     "tool_a",
		Result:       model.CallEventResultUpstreamError,
		LatencyMs:    50,
		ErrorCode:    "upstream_fail",
	}))

	// Aggregate
	err := svc.AggregateDaily(today)
	require.NoError(t, err)

	// Verify aggregates
	aggs, err := svc.ListDailyAggregates(ListDailyAggregatesQuery{UserID: 1, DateFrom: today, DateTo: today})
	require.NoError(t, err)
	require.Len(t, aggs, 2)

	// Find the success aggregate
	var successAgg, errorAgg *model.CallDailyAggregate
	for i := range aggs {
		switch aggs[i].Result {
		case model.CallEventResultSuccess:
			successAgg = &aggs[i]
		case model.CallEventResultUpstreamError:
			errorAgg = &aggs[i]
		}
	}

	require.NotNil(t, successAgg)
	assert.Equal(t, int64(2), successAgg.CallCount)
	assert.Equal(t, int64(300), successAgg.TotalLatencyMs)
	assert.Equal(t, int64(200), successAgg.MaxLatencyMs)

	require.NotNil(t, errorAgg)
	assert.Equal(t, int64(1), errorAgg.CallCount)
	assert.Equal(t, int64(50), errorAgg.TotalLatencyMs)

	// Idempotent re-aggregation: running again should not duplicate rows.
	err = svc.AggregateDaily(today)
	require.NoError(t, err)
	aggs, err = svc.ListDailyAggregates(ListDailyAggregatesQuery{UserID: 1, DateFrom: today, DateTo: today})
	require.NoError(t, err)
	assert.Len(t, aggs, 2)
}

func TestCleanupOldEvents(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Insert an event from 100 days ago.
	oldTime := time.Now().UTC().AddDate(0, 0, -100)
	require.NoError(t, svc.RecordEvent(&model.CallEvent{
		RequestID:    "old-001",
		CallTime:     oldTime,
		UserID:       1,
		McpServiceID: 1,
		ToolName:     "old_tool",
		Result:       model.CallEventResultSuccess,
	}))

	// Insert a recent event.
	require.NoError(t, svc.RecordEvent(&model.CallEvent{
		RequestID:    "new-001",
		CallTime:     time.Now().UTC(),
		UserID:       1,
		McpServiceID: 1,
		ToolName:     "new_tool",
		Result:       model.CallEventResultSuccess,
	}))

	// Cleanup with 90-day retention.
	err := svc.CleanupOldEvents(90)
	require.NoError(t, err)

	// Only the recent event should remain.
	events, err := svc.ListEvents(ListEventsQuery{UserID: 1, Limit: 10})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "new-001", events[0].RequestID)
}

func TestCleanupOldAggregates(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	oldDate := time.Now().UTC().AddDate(0, 0, -100).Format("2006-01-02")
	recentDate := time.Now().UTC().Format("2006-01-02")

	// Insert events and aggregate for both dates.
	require.NoError(t, svc.RecordEvent(&model.CallEvent{
		RequestID:    "agg-old-001",
		CallTime:     time.Now().UTC().AddDate(0, 0, -100),
		UserID:       1,
		McpServiceID: 1,
		ToolName:     "tool",
		Result:       model.CallEventResultSuccess,
		LatencyMs:    10,
	}))
	require.NoError(t, svc.RecordEvent(&model.CallEvent{
		RequestID:    "agg-new-001",
		CallTime:     time.Now().UTC(),
		UserID:       1,
		McpServiceID: 1,
		ToolName:     "tool",
		Result:       model.CallEventResultSuccess,
		LatencyMs:    20,
	}))

	require.NoError(t, svc.AggregateDaily(oldDate))
	require.NoError(t, svc.AggregateDaily(recentDate))

	// Cleanup with 90-day retention.
	err := svc.CleanupOldAggregates(90)
	require.NoError(t, err)

	// Only recent aggregate should remain.
	aggs, err := svc.ListDailyAggregates(ListDailyAggregatesQuery{UserID: 1, DateFrom: oldDate, DateTo: recentDate})
	require.NoError(t, err)
	require.Len(t, aggs, 1)
	assert.Equal(t, recentDate, aggs[0].Date)
}

func TestListEventsDateRange(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Insert events across multiple days.
	for i := 0; i < 5; i++ {
		d := time.Now().UTC().AddDate(0, 0, -i)
		require.NoError(t, svc.RecordEvent(&model.CallEvent{
			RequestID:    "dr-" + d.Format("20060102"),
			CallTime:     d,
			UserID:       1,
			McpServiceID: 1,
			ToolName:     "tool",
			Result:       model.CallEventResultSuccess,
		}))
	}

	today := time.Now().UTC().Format("2006-01-02")
	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2).Format("2006-01-02")

	// Should return today and yesterday (2 days inclusive).
	events, err := svc.ListEvents(ListEventsQuery{
		UserID:   1,
		DateFrom: twoDaysAgo,
		DateTo:   today,
	})
	require.NoError(t, err)
	// Date filter is inclusive on both ends, so we expect 3 days (today, yesterday, 2 days ago).
	assert.True(t, len(events) >= 2 && len(events) <= 3, "expected 2-3 events, got %d", len(events))
}
