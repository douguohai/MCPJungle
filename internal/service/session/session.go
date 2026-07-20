// Package session provides server-side dashboard session management.
package session

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/auth"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

const (
	sessionTokenPrefix = "mjs_"
	sessionTokenBytes  = 32
	sessionLifetime    = 8 * time.Hour
	activeUserStatus   = types.UserStatusActive
	pendingUserStatus  = types.UserStatusPending
)

// Service manages short-lived server-side web sessions for dashboard users.
type Service struct {
	db       *gorm.DB
	now      func() time.Time
	lifetime time.Duration
}

// NewService creates a session service backed by the provided database handle.
func NewService(db *gorm.DB) *Service {
	return &Service{
		db:       db,
		now:      time.Now,
		lifetime: sessionLifetime,
	}
}

// Create creates a new 8-hour session for a user and returns the plaintext
// token exactly once. Persistent storage only keeps the token hash.
func (s *Service) Create(userID uint, ip, userAgent string) (plain string, session *model.UserSession, err error) {
	if userID == 0 {
		return "", nil, fmt.Errorf("user ID must be greater than zero: %w", apierrors.ErrInvalidInput)
	}

	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, fmt.Errorf("user not found: %w", apierrors.ErrNotFound)
		}
		return "", nil, fmt.Errorf("failed to load user: %w", err)
	}

	plain, err = auth.Generate(sessionTokenPrefix, sessionTokenBytes)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	expiresAt := s.now().UTC().Add(s.lifetime)
	session = &model.UserSession{
		UserID:    user.ID,
		TokenHash: auth.Hash(plain),
		ExpiresAt: expiresAt,
		IPAddress: strings.TrimSpace(ip),
		UserAgent: strings.TrimSpace(userAgent),
	}
	if err := s.db.Create(session).Error; err != nil {
		return "", nil, fmt.Errorf("failed to create session: %w", err)
	}

	return plain, session, nil
}

// Authenticate resolves a plaintext session token to its user and session,
// rejecting unknown, revoked, expired, and disabled users. Pending users may
// authenticate only so the API can require their first password change.
func (s *Service) Authenticate(plain string) (*model.User, *model.UserSession, error) {
	if strings.TrimSpace(plain) == "" {
		return nil, nil, fmt.Errorf("invalid session token: %w", apierrors.ErrInvalidCredentials)
	}

	var session model.UserSession
	if err := s.db.Where("token_hash = ?", auth.Hash(plain)).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("invalid session token: %w", apierrors.ErrInvalidCredentials)
		}
		return nil, nil, fmt.Errorf("failed to load session: %w", err)
	}

	now := s.now().UTC()
	if session.RevokedAt != nil {
		return nil, nil, fmt.Errorf("session revoked: %w", apierrors.ErrInvalidCredentials)
	}
	if !session.ExpiresAt.After(now) {
		return nil, nil, fmt.Errorf("session expired: %w", apierrors.ErrInvalidCredentials)
	}

	var user model.User
	if err := s.db.First(&user, session.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("user not found for session: %w", apierrors.ErrInvalidCredentials)
		}
		return nil, nil, fmt.Errorf("failed to load session user: %w", err)
	}
	if user.Status != activeUserStatus && user.Status != pendingUserStatus {
		return nil, nil, fmt.Errorf("user is not active: %w", apierrors.ErrInvalidCredentials)
	}

	lastSeenAt := now
	if err := s.db.Model(&session).Update("last_seen_at", lastSeenAt).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to update session last seen time: %w", err)
	}
	session.LastSeenAt = &lastSeenAt

	return &user, &session, nil
}

// Revoke revokes a single session.
func (s *Service) Revoke(sessionID uint) error {
	if sessionID == 0 {
		return fmt.Errorf("session ID must be greater than zero: %w", apierrors.ErrInvalidInput)
	}

	var session model.UserSession
	if err := s.db.First(&session, sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("session not found: %w", apierrors.ErrNotFound)
		}
		return fmt.Errorf("failed to load session: %w", err)
	}
	if session.RevokedAt != nil {
		return nil
	}

	revokedAt := s.now().UTC()
	if err := s.db.Model(&session).Update("revoked_at", revokedAt).Error; err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}
	return nil
}

// RevokeAllForUser revokes every active session belonging to the user.
func (s *Service) RevokeAllForUser(userID uint) error {
	if userID == 0 {
		return fmt.Errorf("user ID must be greater than zero: %w", apierrors.ErrInvalidInput)
	}

	revokedAt := s.now().UTC()
	if err := s.db.Model(&model.UserSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", revokedAt).Error; err != nil {
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}
	return nil
}
