package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	devicetokenservice "github.com/mcpjungle/mcpjungle/internal/service/devicetoken"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

type v1DeviceTokenCreateRequest struct {
	Name       string                 `json:"name" binding:"required"`
	Scope      types.DeviceTokenScope `json:"scope"`
	ExpiresAt  *time.Time             `json:"expires_at"`
	ServiceIDs []uint                 `json:"service_ids"`
}

func (s *Server) v1CreateDeviceToken(c *gin.Context) {
	var input v1DeviceTokenCreateRequest
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	account := currentV1User(c)
	token, plain, err := s.deviceTokenService.Create(account.ID, devicetokenservice.CreateInput{Name: input.Name, Scope: input.Scope, ExpiresAt: input.ExpiresAt, ServiceIDs: input.ServiceIDs})
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"device_token": token, "token": plain})
}
func (s *Server) v1ListDeviceTokens(c *gin.Context) {
	tokens, err := s.deviceTokenService.ListForUser(currentV1User(c).ID)
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_tokens": tokens})
}

func (s *Server) v1ListUserDeviceTokens(c *gin.Context) {
	userID, ok := v1ID(c)
	if !ok {
		return
	}
	if _, err := s.userService.GetUserByID(userID); err != nil {
		writeV1Error(c, err)
		return
	}
	tokens, err := s.deviceTokenService.ListForUser(userID)
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_tokens": tokens})
}
func (s *Server) v1RevokeDeviceToken(c *gin.Context) {
	id, ok := v1ID(c)
	if !ok {
		return
	}
	account := currentV1User(c)
	if err := s.deviceTokenService.Revoke(account.ID, id, account.Role == types.UserRoleSystemAdmin); err != nil {
		writeV1Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
