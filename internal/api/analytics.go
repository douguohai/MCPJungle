package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/service/callevent"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// listCallEventsHandler returns call event records filtered by query parameters.
// Role filtering:
//   - system_admin / auditor: sees all events
//   - service_admin: sees events for servers they manage (TODO: once manager
//     lookup is wired; for now, same as system_admin)
//   - member: sees only their own events
func (s *Server) listCallEventsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		q := callevent.ListEventsQuery{
			ToolName: c.Query("tool_name"),
			Result:   c.Query("result"),
			CallType: c.Query("call_type"),
			DateFrom: c.Query("date_from"),
			DateTo:   c.Query("date_to"),
		}

		if v := c.Query("server_id"); v != "" {
			id, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server_id"})
				return
			}
			q.ServerID = uint(id)
		}

		if v := c.Query("device_token_id"); v != "" {
			id, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_token_id"})
				return
			}
			tokenID := uint(id)
			q.DeviceTokenID = &tokenID
		}

		if v := c.Query("limit"); v != "" {
			limit, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
				return
			}
			q.Limit = limit
		}

		// Role-based scoping.
		switch u.Role {
		case types.UserRoleSystemAdmin, types.UserRoleAuditor:
			// No user filter — sees everything.
		case types.UserRoleServiceAdmin:
			// TODO: restrict to managed servers once manager lookup is available.
			// For now, allow all.
		default:
			// member: own events only.
			q.UserID = u.ID
		}

		events, err := s.callEventService.ListEvents(q)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list call events"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"events": events})
	}
}

// listDailyAggregatesHandler returns daily-aggregated call stats.
func (s *Server) listDailyAggregatesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		q := callevent.ListDailyAggregatesQuery{
			DateFrom: c.Query("date_from"),
			DateTo:   c.Query("date_to"),
		}

		if v := c.Query("server_id"); v != "" {
			id, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server_id"})
				return
			}
			q.ServerID = uint(id)
		}

		if v := c.Query("limit"); v != "" {
			limit, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
				return
			}
			q.Limit = limit
		}

		switch u.Role {
		case types.UserRoleSystemAdmin, types.UserRoleAuditor, types.UserRoleServiceAdmin:
			// No user filter.
		default:
			q.UserID = u.ID
		}

		aggs, err := s.callEventService.ListDailyAggregates(q)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list daily aggregates"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"aggregates": aggs})
	}
}

// callSummaryHandler returns a quick summary for today / 7-day / 30-day windows.
func (s *Server) callSummaryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		today := time.Now().UTC().Format("2006-01-02")
		sevenDaysAgo := time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02")
		thirtyDaysAgo := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")

		var userID uint
		switch u.Role {
		case types.UserRoleSystemAdmin, types.UserRoleAuditor, types.UserRoleServiceAdmin:
			// global summary
		default:
			userID = u.ID
		}

		type windowSummary struct {
			DateFrom  string `json:"date_from"`
			DateTo    string `json:"date_to"`
			Aggs      interface{} `json:"aggregates"`
		}

		fetchAggs := func(from, to string) interface{} {
			q := callevent.ListDailyAggregatesQuery{
				UserID:   userID,
				DateFrom: from,
				DateTo:   to,
				Limit:    100,
			}
			aggs, err := s.callEventService.ListDailyAggregates(q)
			if err != nil {
				return nil
			}
			return aggs
		}

		summary := gin.H{
			"today": windowSummary{
				DateFrom: today,
				DateTo:   today,
				Aggs:     fetchAggs(today, today),
			},
			"last_7_days": windowSummary{
				DateFrom: sevenDaysAgo,
				DateTo:   today,
				Aggs:     fetchAggs(sevenDaysAgo, today),
			},
			"last_30_days": windowSummary{
				DateFrom: thirtyDaysAgo,
				DateTo:   today,
				Aggs:     fetchAggs(thirtyDaysAgo, today),
			},
		}
		c.JSON(http.StatusOK, summary)
	}
}
