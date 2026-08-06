package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// handleServiceError writes the appropriate HTTP error response for a service-layer error.
// It maps apierrors.ErrNotFound to 404 not found,
// apierrors.ErrInvalidInput to 400 bad request,
// apierrors.ErrInvalidCredentials to 401 with code UNAUTHENTICATED,
// apierrors.ErrPermissionDenied to 403 with code PERMISSION_DENIED,
// apierrors.ErrUpstreamUnavailable to 502 with code UPSTREAM_UNAVAILABLE.
// All other errors become 500.
func handleServiceError(c *gin.Context, err error) {
	if errors.Is(err, apierrors.ErrNotFound) {
		c.JSON(http.StatusNotFound, types.APIErrorResponse{Error: err.Error()})
		return
	}
	if errors.Is(err, apierrors.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, types.APIErrorResponse{
			Error: err.Error(),
			Code:  string(apierrors.CodeUnauthenticated),
		})
		return
	}
	if errors.Is(err, apierrors.ErrPermissionDenied) {
		c.JSON(http.StatusForbidden, types.APIErrorResponse{
			Error: err.Error(),
			Code:  string(apierrors.CodePermissionDenied),
		})
		return
	}
	if errors.Is(err, apierrors.ErrUpstreamUnavailable) {
		c.JSON(http.StatusBadGateway, types.APIErrorResponse{
			Error: err.Error(),
			Code:  string(apierrors.CodeUpstreamUnavailable),
		})
		return
	}
	if errors.Is(err, apierrors.ErrInvalidInput) {
		resp := types.APIErrorResponse{Error: err.Error()}
		if errors.Is(err, apierrors.ErrUpstreamOAuthRequired) {
			resp.Code = apierrors.CodeUpstreamOAuthRequired
		}
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	// Catch-all: log detailed error, return generic message to client.
	log.Printf("[api] unhandled error: %v", err)
	c.JSON(http.StatusInternalServerError, types.APIErrorResponse{Error: "internal server error"})
}

// recordAuditEvent is a best-effort helper that persists an audit event.
// If the audit service is nil or the write fails, only a log message is emitted;
// the caller's operation is never blocked.
func (s *Server) recordAuditEvent(c *gin.Context, actionType, targetType, targetID, changeSummary string) {
	if s.auditEventService == nil {
		return
	}
	var actorID uint
	if u := currentUser(c); u != nil {
		actorID = u.ID
	}
	event := &model.AuditEvent{
		ActorUserID:   actorID,
		ActionType:    actionType,
		TargetType:    targetType,
		TargetID:      targetID,
		OccurredAt:    time.Now().UTC(),
		SourceIP:      c.ClientIP(),
		Result:        "success",
		ChangeSummary: changeSummary,
	}
	if err := s.auditEventService.RecordEvent(event); err != nil {
		log.Printf("[audit] failed to record event %s: %v", actionType, err)
	}
}
