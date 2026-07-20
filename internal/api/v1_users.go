package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	userservice "github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

type v1CreateUserRequest struct {
	Username    string         `json:"username" binding:"required"`
	DisplayName string         `json:"display_name"`
	Role        types.UserRole `json:"role"`
}

type v1UpdateUserRequest struct {
	DisplayName *string         `json:"display_name"`
	Role        *types.UserRole `json:"role"`
}

func (s *Server) v1CreateUser(c *gin.Context) {
	var input v1CreateUserRequest
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}
	account, initialPassword, err := s.userService.Create(userservice.CreateUserInput(input))
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": account, "initial_password": initialPassword})
}

func (s *Server) v1ListUsers(c *gin.Context) {
	accounts, err := s.userService.ListUsers()
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": accounts})
}

func (s *Server) v1UpdateUser(c *gin.Context) {
	id, ok := v1ID(c)
	if !ok {
		return
	}
	var input v1UpdateUserRequest
	if c.ShouldBindJSON(&input) != nil || (input.DisplayName == nil && input.Role == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nothing to update"})
		return
	}
	actor := currentV1User(c)
	account, err := s.userService.UpdateUser(actor.ID, id, userservice.UpdateUserInput{
		DisplayName: input.DisplayName,
		Role:        input.Role,
	})
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": account})
}

func (s *Server) v1EnableUser(c *gin.Context)  { s.v1SetUserStatus(c, types.UserStatusActive) }
func (s *Server) v1DisableUser(c *gin.Context) { s.v1SetUserStatus(c, types.UserStatusDisabled) }

func (s *Server) v1SetUserStatus(c *gin.Context, status types.UserStatus) {
	id, ok := v1ID(c)
	if !ok {
		return
	}
	if err := s.userService.SetStatus(currentV1User(c).ID, id, status); err != nil {
		writeV1Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func v1ID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(value), true
}
