package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type dashboardSettingsResponse struct {
	SystemDisplayName string `json:"system_display_name"`
}

func (s *Server) dashboardSettingsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := s.configService.GetConfig()
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dashboardSettingsResponse{
			SystemDisplayName: cfg.SystemDisplayName,
		})
	}
}

func (s *Server) dashboardUpdateSettingsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SystemDisplayName string `json:"system_display_name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}
		cfg, err := s.configService.UpdateSystemDisplayName(req.SystemDisplayName)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, dashboardSettingsResponse{
			SystemDisplayName: cfg.SystemDisplayName,
		})
	}
}
