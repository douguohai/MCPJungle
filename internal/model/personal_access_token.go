package model

import (
	"time"

	"gorm.io/gorm"
)

// PersonalAccessToken is a long-lived, revocable API credential (PAT) used by
// the CLI and automation. The plaintext token is returned to the caller only at
// creation time; only its sha256 hash is persisted.
type PersonalAccessToken struct {
	gorm.Model

	UserID     uint       `gorm:"not null;index"`
	Name       string     `gorm:"type:varchar(255)"`
	Prefix     string     `gorm:"type:varchar(32);uniqueIndex"`
	TokenHash  string     `gorm:"type:varchar(64);uniqueIndex;not null"`
	ExpiresAt  *time.Time `gorm:"index"`
	LastUsedAt *time.Time
	RevokedAt  *time.Time `gorm:"index"`
}
