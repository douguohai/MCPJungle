package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// dashboardCallStatsHandler returns per-user per-server MCP tool-call counts.
// Admin only (route is under dashboardAdminAPI).
func (s *Server) dashboardCallStatsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := s.userService.ListCallStats()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load call stats"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"stats": rows})
	}
}
