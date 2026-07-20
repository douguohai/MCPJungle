// Package migrations provides database migration functionality for the MCPJungle application.
package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

// Migrate performs the database migration for the application.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.McpServer{}); err != nil {
		return fmt.Errorf("auto‑migration failed for McpServer model: %v", err)
	}
	if err := db.AutoMigrate(&model.Tool{}); err != nil {
		return fmt.Errorf("auto‑migration failed for Tool model: %v", err)
	}
	if err := db.AutoMigrate(&model.ServerConfig{}); err != nil {
		return fmt.Errorf("auto‑migration failed for ServerConfig model: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		return fmt.Errorf("auto‑migration failed for User model: %v", err)
	}
	if err := db.AutoMigrate(&model.McpClient{}); err != nil {
		return fmt.Errorf("auto‑migration failed for McpClient model: %v", err)
	}
	if err := db.AutoMigrate(&model.ToolGroup{}); err != nil {
		return fmt.Errorf("auto‑migration failed for ToolGroup model: %v", err)
	}
	if err := db.AutoMigrate(&model.Prompt{}); err != nil {
		return fmt.Errorf("auto‑migration failed for Prompt model: %v", err)
	}
	if err := db.AutoMigrate(&model.Resource{}); err != nil {
		return fmt.Errorf("auto‑migration failed for Resource model: %v", err)
	}
	if err := db.AutoMigrate(&model.UpstreamOAuthPendingSession{}); err != nil {
		return fmt.Errorf("auto-migration failed for UpstreamOAuthPendingSession model: %v", err)
	}
	if err := db.AutoMigrate(&model.UpstreamOAuthToken{}); err != nil {
		return fmt.Errorf("auto-migration failed for UpstreamOAuthToken model: %v", err)
	}
	if err := db.AutoMigrate(&model.PersonalAccessToken{}); err != nil {
		return fmt.Errorf("auto-migration failed for PersonalAccessToken model: %v", err)
	}
	if err := migrateLegacyAccessTokens(db); err != nil {
		return fmt.Errorf("failed to migrate legacy access tokens: %v", err)
	}
	return nil
}

// migrateLegacyAccessTokens converts each user's legacy plaintext access_token
// into a PersonalAccessToken (hash only) so existing CLI/automation tokens keep
// working as PATs after the move to password+JWT auth. Idempotent.
func migrateLegacyAccessTokens(db *gorm.DB) error {
	var users []model.User
	if err := db.Where("access_token != ''").Find(&users).Error; err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}
	for i := range users {
		u := users[i]
		var count int64
		if err := db.Model(&model.PersonalAccessToken{}).
			Where("user_id = ? AND name = ?", u.ID, "migrated").
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check existing PAT: %w", err)
		}
		if count > 0 {
			continue
		}
		prefix := u.AccessToken
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		pat := model.PersonalAccessToken{
			UserID:    u.ID,
			Name:      "migrated",
			Prefix:    prefix,
			TokenHash: sha256Hex(u.AccessToken),
		}
		if err := db.Create(&pat).Error; err != nil {
			return fmt.Errorf("failed to migrate token for user %s: %w", u.Username, err)
		}
	}
	return nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
