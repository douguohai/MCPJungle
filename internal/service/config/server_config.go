// Package config provides configuration service functionality for the MCPJungle application.
package config

import (
	"errors"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

// ServerConfigService provides methods to manage server configuration in the database.
type ServerConfigService struct {
	db *gorm.DB
}

func NewServerConfigService(db *gorm.DB) *ServerConfigService {
	return &ServerConfigService{db: db}
}

// GetConfig retrieves the server configuration from the database.
// If no configuration exists, it returns a default uninitialized config.
func (s *ServerConfigService) GetConfig() (model.ServerConfig, error) {
	var config model.ServerConfig
	err := s.db.First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.ServerConfig{Initialized: false}, nil
	}
	if err != nil {
		return model.ServerConfig{}, fmt.Errorf("failed to fetch server configuration from db: %v", err)
	}
	return config, nil
}

// Init initializes the server configuration in the database.
// It is an idempotent operation. It returns true if the config was created.
// If the config already exists, it returns false and does nothing else.
func (s *ServerConfigService) Init(mode model.ServerMode) (bool, error) {
	return s.InitWith(mode, nil)
}

// InitWith initializes the server and runs setup in the same transaction.
// This prevents enterprise mode from becoming initialized without its first
// administrator when account creation fails.
func (s *ServerConfigService) InitWith(mode model.ServerMode, setup func(*gorm.DB) error) (bool, error) {
	created := false
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txService := NewServerConfigService(tx)
		config, err := txService.GetConfig()
		if err != nil {
			return err
		}
		if config.Initialized {
			return nil
		}
		config = model.ServerConfig{
			Mode:        mode,
			Initialized: true,
		}
		if err := tx.Create(&config).Error; err != nil {
			return err
		}
		if setup != nil {
			if err := setup(tx); err != nil {
				return err
			}
		}
		created = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return created, nil
}
