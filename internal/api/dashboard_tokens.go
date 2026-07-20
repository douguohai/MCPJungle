package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

type createTokenRequest struct {
	Name          string `json:"name"`
	ExpiresInDays int    `json:"expires_in_days"`
}

type tokenView struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type createTokenResponse struct {
	Token string    `json:"token"` // plaintext, shown only once at creation
	View  tokenView `json:"view"`
}

// dashboardCreateTokenHandler issues a new PAT for the authenticated user.
func (s *Server) dashboardCreateTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		var req createTokenRequest
		_ = c.ShouldBindJSON(&req)
		var expiresAt *time.Time
		if req.ExpiresInDays > 0 {
			t := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
			expiresAt = &t
		}
		plaintext, pat, err := s.userService.CreatePAT(u.ID, req.Name, expiresAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
			return
		}
		c.JSON(http.StatusOK, createTokenResponse{
			Token: plaintext,
			View: tokenView{
				ID: pat.ID, Name: pat.Name, Prefix: pat.Prefix,
				CreatedAt: pat.CreatedAt, ExpiresAt: pat.ExpiresAt,
			},
		})
	}
}

// dashboardListTokensHandler lists the authenticated user's PATs (no plaintext).
func (s *Server) dashboardListTokensHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		pats, err := s.userService.ListPATs(u.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tokens"})
			return
		}
		views := make([]tokenView, 0, len(pats))
		for _, p := range pats {
			views = append(views, tokenView{
				ID: p.ID, Name: p.Name, Prefix: p.Prefix,
				CreatedAt: p.CreatedAt, LastUsedAt: p.LastUsedAt, ExpiresAt: p.ExpiresAt,
			})
		}
		c.JSON(http.StatusOK, gin.H{"tokens": views})
	}
}

// dashboardDeleteTokenHandler revokes one of the authenticated user's PATs.
func (s *Server) dashboardDeleteTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token id"})
			return
		}
		if err := s.userService.RevokePAT(u.ID, uint(id)); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "token revoked"})
	}
}

// currentUser returns the authenticated user placed in the gin context by
// verifyUserAuthForAPIAccess, or nil.
func currentUser(c *gin.Context) *model.User {
	v, ok := c.Get("user")
	if !ok {
		return nil
	}
	u, ok := v.(*model.User)
	if !ok {
		return nil
	}
	return u
}
