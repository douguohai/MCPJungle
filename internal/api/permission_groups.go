package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) listPermissionGroupsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		groups, err := s.permissionService.ListGroups()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, groups)
	}
}

func (s *Server) createPermissionGroupHandler() gin.HandlerFunc {
	type req struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
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
		g, err := s.permissionService.CreateGroup(r.Name, r.DisplayName, r.Description, u.ID)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, g)
	}
}

func (s *Server) getPermissionGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
			return
		}
		g, err := s.permissionService.GetGroup(uint(id))
		if err != nil {
			handleServiceError(c, err)
			return
		}
		members, _ := s.permissionService.ListMembers(g.ID)
		services, _ := s.permissionService.ListServices(g.ID)
		c.JSON(http.StatusOK, gin.H{
			"group":    g,
			"members":  members,
			"services": services,
		})
	}
}

func (s *Server) updatePermissionGroupHandler() gin.HandlerFunc {
	type req struct {
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
	}
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
			return
		}
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if err := s.permissionService.UpdateGroup(uint(id), r.DisplayName, r.Description); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func (s *Server) disablePermissionGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
			return
		}
		if err := s.permissionService.DisableGroup(uint(id)); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func (s *Server) addPermissionGroupMemberHandler() gin.HandlerFunc {
	type req struct {
		UserID uint `json:"user_id"`
	}
	return func(c *gin.Context) {
		gid, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
			return
		}
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
		if err := s.permissionService.AddMember(uint(gid), r.UserID, u.ID); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	}
}

func (s *Server) removePermissionGroupMemberHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		gid, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
			return
		}
		uid, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}
		if err := s.permissionService.RemoveMember(uint(gid), uint(uid)); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func (s *Server) addPermissionGroupServiceHandler() gin.HandlerFunc {
	type req struct {
		McpServerID uint `json:"mcp_server_id"`
	}
	return func(c *gin.Context) {
		gid, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
			return
		}
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
		if err := s.permissionService.AddService(uint(gid), r.McpServerID, u.ID); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	}
}

func (s *Server) removePermissionGroupServiceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		gid, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
			return
		}
		sid, err := strconv.ParseUint(c.Param("server_id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server id"})
			return
		}
		if err := s.permissionService.RemoveService(uint(gid), uint(sid)); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
