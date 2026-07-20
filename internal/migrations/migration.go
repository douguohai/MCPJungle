// Package migrations provides database migration functionality for the MCPJungle application.
package migrations

import (
	"errors"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// Migrate performs the database migration for the application.
func Migrate(db *gorm.DB) error {
	for _, m := range []interface{}{
		&model.McpServer{},
		&model.Tool{},
		&model.ServerConfig{},
		&model.User{},
		&model.McpClient{},
		&model.ToolGroup{},
		&model.Prompt{},
		&model.Resource{},
		&model.UpstreamOAuthPendingSession{},
		&model.UpstreamOAuthToken{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			return fmt.Errorf("auto-migration failed for %T: %v", m, err)
		}
	}
	if err := backfillClientUserIDs(db); err != nil {
		return fmt.Errorf("failed to backfill client user_ids: %v", err)
	}
	return nil
}

// backfillClientUserIDs assigns pre-existing clients (UserID == 0, created before
// clients were owned by users) to the first admin user, if one exists.
func backfillClientUserIDs(db *gorm.DB) error {
	var admin model.User
	err := db.Where("role = ?", types.UserRoleAdmin).Order("id").First(&admin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // no admin yet (e.g. dev mode); nothing to backfill
		}
		return err
	}
	return db.Model(&model.McpClient{}).Where("user_id = 0").Update("user_id", admin.ID).Error
}
