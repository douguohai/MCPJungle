package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	userservice "github.com/mcpjungle/mcpjungle/internal/service/user"
	"gorm.io/gorm"
)

func (s *Server) registerInitServerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Mode          model.ServerMode `json:"mode" binding:"required,oneof=development enterprise"`
			AdminUsername string           `json:"admin_username"`
			AdminPassword string           `json:"admin_password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}
		if req.Mode == model.ModeDev {
			ok, err := s.configService.Init(req.Mode)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize server: " + err.Error()})
				return
			}
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Server is already initialized"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "Server initialized successfully in development mode"})
			return
		}

		var admin *model.User
		ok, err := s.configService.InitWith(req.Mode, func(tx *gorm.DB) error {
			var createErr error
			admin, createErr = userservice.NewUserService(tx).CreateAdminUser(req.AdminUsername, req.AdminPassword)
			return createErr
		})
		if err != nil {
			handleServiceError(c, err)
			return
		}
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Server is already initialized"})
			return
		}
		payload := gin.H{
			"status":         "Server initialized successfully",
			"admin_username": admin.Username,
			"message":        "Use these credentials to log in to the dashboard",
		}
		c.JSON(http.StatusOK, payload)
	}
}
