package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

type dashboardVerifyTokenResponse struct {
	Authenticated bool   `json:"authenticated"`
	Mode          string `json:"mode"`
	Username      string `json:"username,omitempty"`
	Role          string `json:"role,omitempty"`
}

// dashboardLoginHandler uses the same database-backed session as /api/v1.
// The plaintext session identifier is delivered only in an HttpOnly cookie.
func (s *Server) dashboardLoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) { s.v1Login(c) }
}

func (s *Server) dashboardVerifyTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode := c.MustGet("mode").(model.ServerMode)
		if mode == model.ModeDev {
			c.JSON(http.StatusOK, dashboardVerifyTokenResponse{Authenticated: true, Mode: string(mode)})
			return
		}
		plain, err := c.Cookie(sessionCookieName)
		if err != nil || s.sessionService == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		account, _, err := s.sessionService.Authenticate(plain)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		c.JSON(http.StatusOK, dashboardVerifyTokenResponse{
			Authenticated: true, Mode: string(mode), Username: account.Username, Role: string(account.Role),
		})
	}
}
