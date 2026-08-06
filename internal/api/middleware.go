package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// requireInitialized is middleware to reject requests to certain routes if the server is not initialized
func (s *Server) requireInitialized() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := s.configService.GetConfig()
		if err != nil || !cfg.Initialized {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "server is not initialized"})
			return
		}
		// propagate the server mode in context for other middleware/handlers to use
		c.Set("mode", cfg.Mode)
		c.Next()
	}
}

// requireDashboardMode allows dashboard access in development and enterprise modes,
// returning 404 for any other mode. It guards both the embedded frontend routes and
// the /api/dashboard endpoints.
func (s *Server) requireDashboardMode() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode, exists := c.Get("mode")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server mode not found in context"})
			return
		}
		currentMode, ok := mode.(model.ServerMode)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid server mode in context"})
			return
		}
		if currentMode != model.ModeDev && !model.IsEnterpriseMode(currentMode) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Next()
	}
}

// verifyUserAuthForAPIAccess is middleware that authenticates a dashboard
// request in enterprise mode via the web session (cookie or Bearer). Dev mode
// is always allowed. This middleware does not check the user's role;
// requireAdminUser does that.
func (s *Server) verifyUserAuthForAPIAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode, exists := c.Get("mode")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server mode not found in context"})
			return
		}
		m, ok := mode.(model.ServerMode)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid server mode in context"})
			return
		}
		if m == model.ModeDev {
			// no auth is required in dev mode
			c.Next()
			return
		}

		sess, err := s.userSessionService.Lookup(sessionFromRequest(c))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierrors.NewAPIError(apierrors.CodeUnauthenticated, "invalid credentials"))
			return
		}
		user, err := s.userService.GetUserByID(sess.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierrors.NewAPIError(apierrors.CodeUnauthenticated, "invalid credentials"))
			return
		}
		s.userSessionService.Touch(sess.ID)
		c.Set("user", user)
		c.Next()
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

// requireAdminUser is middleware that ensures the authenticated user has an admin role when in enterprise mode.
// It assumes that verifyUserAuthForAPIAccess middleware has already run and set the user in context.
func (s *Server) requireAdminUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode, exists := c.Get("mode")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server mode not found in context"})
			return
		}
		m, ok := mode.(model.ServerMode)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid server mode in context"})
			return
		}
		if m == model.ModeDev {
			// no admin check is required in dev mode
			c.Next()
			return
		}

		authenticatedUser, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user is not authenticated"})
			return
		}

		u, ok := authenticatedUser.(*model.User)
		if ok && u.Role == types.UserRoleSystemAdmin {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user is not authorized to perform this action"})
	}
}

// requireServerMode is middleware that checks if the server is in a specific mode.
// If not, the request is rejected with a 403 Forbidden status.
// This is useful for routes that should only be accessible in certain modes (e.g., enterprise-only features).
// NOTE: ModeProd is supported for backwards compatibility, it is equivalent to ModeEnterprise.
func (s *Server) requireServerMode(m model.ServerMode) gin.HandlerFunc {
	return func(c *gin.Context) {
		mode, exists := c.Get("mode")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server mode not found in context"})
			return
		}
		currentMode, ok := mode.(model.ServerMode)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid server mode in context"})
			return
		}

		if currentMode == m {
			// current mode matches the required mode, allow access
			c.Next()
			return
		}
		if model.IsEnterpriseMode(currentMode) && model.IsEnterpriseMode(m) {
			// both current and required modes are enterprise modes, allow access
			c.Next()
			return
		}
		// current mode does not match the required mode, reject the request
		c.AbortWithStatusJSON(
			http.StatusForbidden,
			gin.H{"error": fmt.Sprintf("this request is only allowed in %s mode", m)},
		)
	}
}

// checkAuthForMcpProxyAccess is middleware for MCP proxy that checks for a valid MCP client token
// checkAuthForMcpProxyAccess authenticates MCP proxy requests in enterprise mode.
// In development mode, no auth is required.
// The middleware validates the device token and computes the user's effective
// service set (permission groups ∩ token scope), injecting both into the
// request context for the proxy and tool filter to consume without DB lookups.
func (s *Server) checkAuthForMcpProxyAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode, exists := c.Get("mode")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server mode not found in context"})
			return
		}
		m, ok := mode.(model.ServerMode)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid server mode in context"})
			return
		}

		// the gin context doesn't get passed down to the MCP proxy server, so we need to
		// set values in the underlying request's context to be able to access them from proxy.
		ctx := context.WithValue(c.Request.Context(), "mode", m)
		c.Request = c.Request.WithContext(ctx)

		if m == model.ModeDev {
			// no auth is required in dev mode
			c.Next()
			return
		}

		rawToken := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if rawToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierrors.NewAPIError(apierrors.CodeUnauthenticated, "missing device token"))
			return
		}
		deviceToken, err := s.deviceTokenService.GetByToken(rawToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierrors.NewAPIError(apierrors.CodeUnauthenticated, "invalid device token"))
			return
		}

		// Compute effective service names (permission groups ∩ token scope).
		effectiveIDs := make(map[uint]bool)
		if ids, err := s.permissionService.UserEffectiveServices(deviceToken.UserID); err == nil {
			for _, id := range ids {
				effectiveIDs[id] = true
			}
		}
		if deviceToken.ScopeMode == model.DeviceTokenScopeRestricted {
			if restrictedIDs, err := s.deviceTokenService.GetRestrictedServices(deviceToken.ID); err == nil {
				allowed := make(map[uint]bool)
				for _, id := range restrictedIDs {
					if effectiveIDs[id] {
						allowed[id] = true
					}
				}
				effectiveIDs = allowed
			}
		}
		effectiveNames := make(map[string]bool)
		for id := range effectiveIDs {
			if srv, err := s.mcpService.GetMcpServerByID(id); err == nil {
				effectiveNames[srv.Name] = true
			}
		}

		ctx = context.WithValue(ctx, "device_token", deviceToken)
		ctx = context.WithValue(ctx, "effective_services", effectiveNames)
		if deviceToken.UserID > 0 {
			if user, err := s.userService.GetUserByID(deviceToken.UserID); err == nil {
				ctx = context.WithValue(ctx, "user", user)
			}
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
