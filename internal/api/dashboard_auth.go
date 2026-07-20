package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

type dashboardVerifyTokenRequest struct {
	AccessToken string `json:"access_token"`
}

type dashboardVerifyTokenResponse struct {
	Authenticated bool   `json:"authenticated"`
	Mode          string `json:"mode"`
	Username      string `json:"username,omitempty"`
	Role          string `json:"role,omitempty"`
}

type dashboardUser struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type dashboardLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type dashboardLoginResponse struct {
	Token     string        `json:"token"`
	ExpiresAt time.Time     `json:"expires_at"`
	User      dashboardUser `json:"user"`
}

// dashboardLoginHandler authenticates a user with username+password and issues
// a short-lived session JWT. This endpoint is intentionally NOT protected by
// verifyUserAuthForAPIAccess so unauthenticated users can log in.
func (s *Server) dashboardLoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dashboardLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		user, err := s.userService.VerifyPassword(req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		token, exp, err := s.authSigner.Sign(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue session token"})
			return
		}
		c.JSON(http.StatusOK, dashboardLoginResponse{
			Token:     token,
			ExpiresAt: exp,
			User:      dashboardUser{Username: user.Username, Role: string(user.Role)},
		})
	}
}

// dashboardVerifyTokenHandler validates a session JWT (or, during the migration,
// a legacy access token) for the dashboard. In development mode it always
// succeeds without requiring a token. This endpoint is intentionally NOT
// protected by verifyUserAuthForAPIAccess so the login page can reach it before
// the user is authenticated.
func (s *Server) dashboardVerifyTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode := c.MustGet("mode").(model.ServerMode)

		if mode == model.ModeDev {
			c.JSON(http.StatusOK, dashboardVerifyTokenResponse{
				Authenticated: true,
				Mode:          string(mode),
			})
			return
		}

		// Read the token from the request body, falling back to the
		// Authorization header so the same endpoint works for JWT and legacy tokens.
		var input dashboardVerifyTokenRequest
		_ = c.ShouldBindJSON(&input)
		token := strings.TrimSpace(input.AccessToken)
		if token == "" {
			token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing access token"})
			return
		}

		// Prefer a session JWT.
		if s.authSigner != nil {
			if claims, err := s.authSigner.Parse(token); err == nil {
				c.JSON(http.StatusOK, dashboardVerifyTokenResponse{
					Authenticated: true,
					Mode:          string(mode),
					Username:      claims.Username,
					Role:          claims.Role,
				})
				return
			}
		}

		// Fall back to a legacy long-lived access token during the migration.
		if u, err := s.userService.GetUserByAccessToken(token); err == nil {
			c.JSON(http.StatusOK, dashboardVerifyTokenResponse{
				Authenticated: true,
				Mode:          string(mode),
				Username:      u.Username,
				Role:          string(u.Role),
			})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	}
}
