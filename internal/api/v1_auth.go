package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

const sessionCookieName = "mcpj_session"

type v1LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type v1PasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (s *Server) v1Login(c *gin.Context) {
	var input v1LoginRequest
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}
	account, err := s.userService.VerifyPassword(input.Username, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	plain, session, err := s.sessionService.Create(account.ID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		writeV1Error(c, err)
		return
	}
	setV1SessionCookie(c, plain, session)
	c.JSON(http.StatusOK, gin.H{"user": account, "must_change_password": account.MustChangePassword})
}

func (s *Server) v1Logout(c *gin.Context) {
	session := currentV1Session(c)
	if session != nil {
		_ = s.sessionService.Revoke(session.ID)
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: requestUsesTLS(c.Request), SameSite: http.SameSiteLaxMode,
	})
	c.Status(http.StatusNoContent)
}

func (s *Server) v1Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user": currentV1User(c)})
}

func (s *Server) v1ChangePassword(c *gin.Context) {
	var input v1PasswordRequest
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current_password and new_password are required"})
		return
	}
	account := currentV1User(c)
	if account == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}
	if err := s.userService.ChangePassword(account.ID, input.CurrentPassword, input.NewPassword); err != nil {
		writeV1Error(c, err)
		return
	}
	plain, newSession, err := s.sessionService.Create(account.ID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		writeV1Error(c, err)
		return
	}
	setV1SessionCookie(c, plain, newSession)
	updated, err := s.userService.GetUserByID(account.ID)
	if err != nil {
		writeV1Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": updated})
}

func setV1SessionCookie(c *gin.Context, plain string, session *model.UserSession) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookieName, Value: plain, Path: "/", Expires: session.ExpiresAt,
		MaxAge: int(session.ExpiresAt.Sub(session.CreatedAt).Seconds()), HttpOnly: true,
		Secure: requestUsesTLS(c.Request), SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) requireV1Session() gin.HandlerFunc {
	return func(c *gin.Context) {
		plain, err := c.Cookie(sessionCookieName)
		if err != nil || strings.TrimSpace(plain) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		account, session, err := s.sessionService.Authenticate(plain)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		c.Set("v1_user", account)
		c.Set("v1_session", session)
		c.Next()
	}
}

func (s *Server) requireActiveV1User() gin.HandlerFunc {
	return func(c *gin.Context) {
		account := currentV1User(c)
		if account == nil || account.Status != types.UserStatusActive || account.MustChangePassword {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "password change is required"})
			return
		}
		c.Next()
	}
}

func (s *Server) requireV1SystemAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		account := currentV1User(c)
		if account == nil || account.Role != types.UserRoleSystemAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "system administrator required"})
			return
		}
		c.Next()
	}
}

func currentV1User(c *gin.Context) *model.User {
	value, _ := c.Get("v1_user")
	account, _ := value.(*model.User)
	return account
}

func currentV1Session(c *gin.Context) *model.UserSession {
	value, _ := c.Get("v1_session")
	session, _ := value.(*model.UserSession)
	return session
}

func requestUsesTLS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func writeV1Error(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apierrors.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	case errors.Is(err, apierrors.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, apierrors.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
