package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/devicetoken"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func (s *Server) listDeviceTokensHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		tokens, err := s.deviceTokenService.List(u.ID, u.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, tokens)
	}
}

func (s *Server) createDeviceTokenHandler() gin.HandlerFunc {
	type req struct {
		Name                  string   `json:"name"`
		ScopeMode             string   `json:"scope_mode"`
		RestrictedServerNames []string `json:"restricted_server_names,omitempty"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		u := currentUser(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		if r.ScopeMode == "" {
			r.ScopeMode = "inherit_all"
		}
		// Resolve restricted server names to IDs if scope is restricted.
		var restrictedIDs []uint
		if r.ScopeMode == model.DeviceTokenScopeRestricted && len(r.RestrictedServerNames) > 0 {
			for _, name := range r.RestrictedServerNames {
				srv, err := s.mcpService.GetMcpServer(name)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("MCP server '%s' not found", name)})
					return
				}
				restrictedIDs = append(restrictedIDs, srv.ID)
			}
		}
		raw, tok, err := s.deviceTokenService.Create(u.ID, r.Name, r.ScopeMode, restrictedIDs, devicetoken.DefaultTokenTTL)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		// The raw token is returned exactly once; the response contains both the
		// raw value (for the user to copy) and the token record (for display).
		c.JSON(http.StatusCreated, gin.H{
			"raw_token": raw,
			"token":     tok,
		})
	}
}

func (s *Server) getDeviceTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token id"})
			return
		}
		u := currentUser(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		tok, err := s.deviceTokenService.GetByID(uint(tid))
		if err != nil {
			handleServiceError(c, err)
			return
		}
		if u.Role != types.UserRoleSystemAdmin && tok.UserID != u.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusOK, tok)
	}
}

func (s *Server) revokeDeviceTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token id"})
			return
		}
		if err := s.deviceTokenService.Revoke(uint(tid)); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func (s *Server) deleteDeviceTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		tid, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token id"})
			return
		}
		u := currentUser(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		if err := s.deviceTokenService.Delete(uint(tid), u.ID, u.Role); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
