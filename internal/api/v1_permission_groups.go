package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	permissionservice "github.com/mcpjungle/mcpjungle/internal/service/permission"
)

type v1PermissionGroupCreateRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}
type v1PermissionGroupUpdateRequest struct {
	DisplayName *string `json:"display_name"`
	Description *string `json:"description"`
}
type v1IDsRequest struct {
	IDs []uint `json:"ids"`
}

func (s *Server) v1CreatePermissionGroup(c *gin.Context) {
	var input v1PermissionGroupCreateRequest
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	group, err := s.permissionService.Create(permissionservice.CreateInput(input))
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"permission_group": group})
}
func (s *Server) v1ListPermissionGroups(c *gin.Context) {
	groups, err := s.permissionService.List()
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"permission_groups": groups})
}
func (s *Server) v1GetPermissionGroup(c *gin.Context) {
	id, ok := v1ID(c)
	if !ok {
		return
	}
	detail, err := s.permissionService.Get(id)
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}
func (s *Server) v1UpdatePermissionGroup(c *gin.Context) {
	id, ok := v1ID(c)
	if !ok {
		return
	}
	var input v1PermissionGroupUpdateRequest
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := s.permissionService.Update(id, permissionservice.UpdateInput(input)); err != nil {
		writeV1Error(c, err)
		return
	}
	s.v1GetPermissionGroup(c)
}
func (s *Server) v1ReplacePermissionGroupUsers(c *gin.Context) {
	s.v1ReplacePermissionGroupLinks(c, true)
}
func (s *Server) v1ReplacePermissionGroupServices(c *gin.Context) {
	s.v1ReplacePermissionGroupLinks(c, false)
}
func (s *Server) v1ReplacePermissionGroupLinks(c *gin.Context, users bool) {
	id, ok := v1ID(c)
	if !ok {
		return
	}
	var input v1IDsRequest
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required"})
		return
	}
	var err error
	if users {
		err = s.permissionService.ReplaceUsers(id, input.IDs)
	} else {
		err = s.permissionService.ReplaceServices(id, input.IDs)
	}
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (s *Server) v1EnablePermissionGroup(c *gin.Context)  { s.v1SetPermissionGroupEnabled(c, true) }
func (s *Server) v1DisablePermissionGroup(c *gin.Context) { s.v1SetPermissionGroupEnabled(c, false) }
func (s *Server) v1SetPermissionGroupEnabled(c *gin.Context, enabled bool) {
	id, ok := v1ID(c)
	if !ok {
		return
	}
	if err := s.permissionService.SetEnabled(id, enabled); err != nil {
		writeV1Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
