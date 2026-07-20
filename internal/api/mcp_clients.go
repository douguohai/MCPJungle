package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func (s *Server) listMcpClientsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		clients, err := s.mcpClientService.ListClients(u.ID, u.Role)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, clients)
	}
}

func (s *Server) createMcpClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		var req model.McpClient
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		// Non-admins cannot grant a client access to servers outside their own
		// AllowedServers (admin is unrestricted).
		if u.Role != types.UserRoleAdmin {
			var reqServers []string
			_ = json.Unmarshal(req.AllowList, &reqServers)
			for _, srv := range reqServers {
				if !u.CheckAllowedServer(srv) {
					c.AbortWithStatusJSON(
						http.StatusForbidden,
						gin.H{"error": fmt.Sprintf("you are not permitted to grant access to server %q", srv)},
					)
					return
				}
			}
		}
		req.UserID = u.ID
		client, err := s.mcpClientService.CreateClient(req)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, client)
	}
}

func (s *Server) deleteMcpClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if err := s.mcpClientService.DeleteClient(u.ID, u.Role, name); err != nil {
			handleServiceError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func (s *Server) updateMcpClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		var req model.McpClient
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		req.Name = name
		// Non-admins cannot raise a client's AllowList above their own AllowedServers.
		if u.Role != types.UserRoleAdmin && req.AllowList != nil {
			var reqServers []string
			_ = json.Unmarshal(req.AllowList, &reqServers)
			for _, srv := range reqServers {
				if !u.CheckAllowedServer(srv) {
					c.AbortWithStatusJSON(
						http.StatusForbidden,
						gin.H{"error": fmt.Sprintf("you are not permitted to grant access to server %q", srv)},
					)
					return
				}
			}
		}
		resp, err := s.mcpClientService.UpdateClient(u.ID, u.Role, req)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}
