package model

import (
	"time"

	"gorm.io/gorm"
)

// UserSession stores a server-side dashboard session. The browser keeps only
// the opaque plaintext token; the database stores its hash plus metadata.
type UserSession struct {
	gorm.Model

	UserID uint `json:"user_id" gorm:"not null;index"`

	TokenHash string `json:"-" gorm:"type:char(64);not null;uniqueIndex"`

	ExpiresAt  time.Time  `json:"expires_at" gorm:"not null"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	IPAddress  string     `json:"ip_address,omitempty" gorm:"type:varchar(45)"`
	UserAgent  string     `json:"user_agent,omitempty" gorm:"type:varchar(512)"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}
