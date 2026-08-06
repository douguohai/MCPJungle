// Package usersession manages web dashboard sessions.
//
// The browser holds a random session identifier in an HttpOnly cookie; this
// service stores only the SHA-256 hash of that identifier so a leaked database
// cannot impersonate sessions. Sessions expire and are revocable.
package usersession

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

// DefaultSessionTTL matches design doc §5.2 (8h default web session lifetime).
const DefaultSessionTTL = 8 * time.Hour

const sessionIDBytes = 32 // → 64 hex chars

// Service manages UserSession records.
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Create issues a new session for userID and returns the raw session identifier
// (to place in a cookie) and the persisted record.
func (s *Service) Create(userID uint, sourceIP, uaHash string, ttl time.Duration) (string, *model.UserSession, error) {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	raw, err := generateID()
	if err != nil {
		return "", nil, err
	}
	now := time.Now().UTC()
	sess := &model.UserSession{
		UserID:         userID,
		SessionIDHash:  hashID(raw),
		ExpiresAt:      now.Add(ttl),
		LastActivityAt: now,
		SourceIP:       sourceIP,
		UAHash:         uaHash,
	}
	if err := s.db.Create(sess).Error; err != nil {
		return "", nil, err
	}
	return raw, sess, nil
}

// Lookup validates a raw session identifier and returns the active session.
// Returns gorm.ErrRecordNotFound if the session is missing, revoked, or expired.
func (s *Service) Lookup(rawID string) (*model.UserSession, error) {
	if rawID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var sess model.UserSession
	if err := s.db.Where("session_id_hash = ?", hashID(rawID)).First(&sess).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	if sess.RevokedAt != nil {
		return nil, gorm.ErrRecordNotFound
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		return nil, gorm.ErrRecordNotFound
	}
	return &sess, nil
}

// Touch refreshes LastActivityAt for a session (best-effort, errors ignored).
func (s *Service) Touch(id uint) {
	_ = s.db.Model(&model.UserSession{}).Where("id = ?", id).
		Update("last_activity_at", time.Now().UTC()).Error
}

// Revoke marks a session revoked by raw identifier (idempotent).
func (s *Service) Revoke(rawID string) error {
	if rawID == "" {
		return nil
	}
	now := time.Now().UTC()
	return s.db.Model(&model.UserSession{}).Where("session_id_hash = ?", hashID(rawID)).
		Update("revoked_at", now).Error
}

func generateID() (string, error) {
	b := make([]byte, sessionIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashID(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
