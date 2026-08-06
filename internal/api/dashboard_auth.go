package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/usersession"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
)

const sessionCookieName = "mcpjungle_session"

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
	ExpiresAt time.Time     `json:"expires_at"`
	User      dashboardUser `json:"user"`
}

// dashboardLoginHandler authenticates a user with username+password and issues
// a web session in an HttpOnly cookie. CLI clients read the session id from the
// Set-Cookie response header. This endpoint is intentionally NOT protected by
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
			c.JSON(http.StatusUnauthorized, apierrors.NewAPIError(apierrors.CodeUnauthenticated, "invalid credentials"))
			return
		}
		rawID, sess, err := s.userSessionService.Create(user.ID, c.ClientIP(), hashUA(c.GetHeader("User-Agent")), usersession.DefaultSessionTTL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			return
		}
		setSessionCookie(c, rawID, sess.ExpiresAt)
		c.JSON(http.StatusOK, dashboardLoginResponse{
			ExpiresAt: sess.ExpiresAt,
			User:      dashboardUser{Username: user.Username, Role: string(user.Role)},
		})
	}
}

// dashboardVerifyTokenHandler reports whether the request carries a valid web
// session. Dev mode always succeeds. The login page reaches this before auth.
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

		sess, err := s.userSessionService.Lookup(sessionFromRequest(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		user, err := s.userService.GetUserByID(sess.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		s.userSessionService.Touch(sess.ID)
		c.JSON(http.StatusOK, dashboardVerifyTokenResponse{
			Authenticated: true,
			Mode:          string(mode),
			Username:      user.Username,
			Role:          string(user.Role),
		})
	}
}

// dashboardLogoutHandler revokes the current web session and clears the cookie.
func (s *Server) dashboardLogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if rawID := sessionFromRequest(c); rawID != "" {
			_ = s.userSessionService.Revoke(rawID)
		}
		clearSessionCookie(c)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// sessionFromRequest extracts the raw session id from the session cookie
// (browser) or, falling back, from an Authorization: Bearer header (CLI, which
// stores the session id in its config file).
func sessionFromRequest(c *gin.Context) string {
	if v, err := c.Cookie(sessionCookieName); err == nil && v != "" {
		return v
	}
	return strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
}

// setSessionCookie writes the session id as an HttpOnly cookie. Secure is
// disabled in development mode (which runs over plain HTTP locally).
func setSessionCookie(c *gin.Context, rawID string, expires time.Time) {
	secure := true
	if mode, ok := c.Get("mode"); ok {
		if m, ok := mode.(model.ServerMode); ok && m == model.ModeDev {
			secure = false
		}
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, rawID, int(time.Until(expires).Seconds()), "/", "", secure, true)
}

func clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, "", -1, "/", "", true, true)
}

// hashUA returns the hex SHA-256 of a User-Agent string, or "" if empty.
func hashUA(ua string) string {
	if ua == "" {
		return ""
	}
	h := sha256.Sum256([]byte(ua))
	return hex.EncodeToString(h[:])
}
