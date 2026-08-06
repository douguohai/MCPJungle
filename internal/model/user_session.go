package model

import (
	"time"

	"gorm.io/gorm"
)

// UserSession represents an authenticated web dashboard session.
//
// The browser holds a random session identifier in an HttpOnly cookie; this
// table stores the SHA-256 hash of that identifier so a leaked database cannot
// be used to impersonate sessions. Sessions are revocable (RevokedAt) and expire
// (ExpiresAt); both are checked on every request.
type UserSession struct {
	gorm.Model

	UserID uint `gorm:"not null;index"`

	// SessionIDHash is the hex-encoded SHA-256 of the random session identifier
	// sent to the browser in the cookie.
	SessionIDHash string `gorm:"type:varchar(64);uniqueIndex;not null"`

	// ExpiresAt is the absolute time after which the session is no longer valid.
	ExpiresAt time.Time `gorm:"not null"`

	// LastActivityAt is refreshed on every authenticated request.
	LastActivityAt time.Time `gorm:"not null"`

	// SourceIP and UAHash are best-effort audit hints (UA is hashed to avoid
	// storing raw User-Agent strings).
	SourceIP string `gorm:"type:varchar(64)"`
	UAHash   string `gorm:"type:varchar(64)"`

	// RevokedAt is set when the session is explicitly terminated (logout / admin
	// revoke / password change). A non-nil value means the session is invalid.
	RevokedAt *time.Time `gorm:"index"`
}
