package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/service/auditevent"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// listAuditEventsHandler returns audit event records.
// Only system_admin and auditor roles may access this endpoint.
func (s *Server) listAuditEventsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		// Only system admins and auditors may view audit events.
		if u.Role != types.UserRoleSystemAdmin && u.Role != types.UserRoleAuditor {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions to view audit events"})
			return
		}

		q := auditevent.ListEventsQuery{
			ActionType: c.Query("action_type"),
			TargetType: c.Query("target_type"),
			DateFrom:   c.Query("date_from"),
			DateTo:     c.Query("date_to"),
		}

		if v := c.Query("actor_id"); v != "" {
			id, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid actor_id"})
				return
			}
			q.ActorID = uint(id)
		}

		if v := c.Query("limit"); v != "" {
			limit, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
				return
			}
			q.Limit = limit
		}

		events, err := s.auditEventService.ListEvents(q)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit events"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"events": events})
	}
}
