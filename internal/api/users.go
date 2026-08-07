package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func (s *Server) createUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input model.User
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		newUser, err := s.userService.CreateUser(&input)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		s.recordAuditEvent(c, "user.created", "user", fmt.Sprintf("%d", newUser.ID),
			fmt.Sprintf("created user %s with role %s", newUser.Username, newUser.Role))

		resp := &types.CreateOrUpdateUserResponse{
			Username: newUser.Username,
			Role:     string(newUser.Role),
		}
		c.JSON(http.StatusCreated, resp)
	}
}

func (s *Server) listUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := s.userService.ListUsers()
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp := make([]*types.User, len(users))
		for i, u := range users {
			resp[i] = &types.User{
				ID:       u.ID,
				Username: u.Username,
				Role:     string(u.Role),
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) deleteUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
			return
		}

		err := s.userService.DeleteUser(username)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		s.recordAuditEvent(c, "user.deleted", "user", username,
			fmt.Sprintf("deleted user %s", username))

		c.Status(http.StatusNoContent)
	}
}

func (s *Server) disableUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		// Fetch user first so we have the ID for session/token revocation.
		targetUser, err := s.userService.GetUserByUsername(username)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		if err := s.userService.DisableUser(username); err != nil {
			handleServiceError(c, err)
			return
		}

		// Revoke all sessions and device tokens for the disabled user to
		// force re-authentication if the account is ever re-enabled.
		if s.userSessionService != nil {
			if err := s.userSessionService.RevokeAllForUser(targetUser.ID); err != nil {
				slog.Error("disableUser: failed to revoke sessions", "username", username, "error", err)
			}
		}
		if s.deviceTokenService != nil {
			if err := s.deviceTokenService.RevokeAllForUser(targetUser.ID); err != nil {
				slog.Error("disableUser: failed to revoke device tokens", "username", username, "error", err)
			}
		}

		s.recordAuditEvent(c, "user.disabled", "user", username,
			fmt.Sprintf("disabled user %s", username))

		c.JSON(http.StatusOK, gin.H{
			"ok":       true,
			"username": targetUser.Username,
			"status":   model.UserStatusDisabled,
		})
	}
}

func (s *Server) whoAmIHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		u, ok := currentUser.(*model.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user from context"})
			return
		}

		resp := types.User{
			Username: u.Username,
			Role:     string(u.Role),
		}
		c.JSON(http.StatusOK, resp)
	}
}
