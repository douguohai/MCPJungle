// Package config provides configuration service functionality for the MCPJungle application.
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"gorm.io/gorm"
)

const DefaultSystemDisplayName = "MCPJungle"

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
	applyConfigDefaults(&config)
	return config, nil
}

// UpdateSystemDisplayName changes the dashboard product name shown to operators.
func (s *ServerConfigService) UpdateSystemDisplayName(name string) (model.ServerConfig, error) {
	name = normalizeSystemDisplayName(name)
	if name == "" {
		return model.ServerConfig{}, fmt.Errorf("system display name is required: %w", apierrors.ErrInvalidInput)
	}
	if len([]rune(name)) > 64 {
		return model.ServerConfig{}, fmt.Errorf("system display name must be at most 64 characters: %w", apierrors.ErrInvalidInput)
	}
	config, err := s.GetConfig()
	if err != nil {
		return model.ServerConfig{}, err
	}
	if !config.Initialized {
		return model.ServerConfig{}, fmt.Errorf("server is not initialized: %w", apierrors.ErrInvalidInput)
	}
	config.SystemDisplayName = name
	if err := s.db.Save(&config).Error; err != nil {
		return model.ServerConfig{}, fmt.Errorf("failed to update server configuration: %v", err)
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
			Mode:              mode,
			Initialized:       true,
			SystemDisplayName: DefaultSystemDisplayName,
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

func applyConfigDefaults(config *model.ServerConfig) {
	config.SystemDisplayName = normalizeSystemDisplayName(config.SystemDisplayName)
	if config.SystemDisplayName == "" {
		config.SystemDisplayName = DefaultSystemDisplayName
	}
}

func normalizeSystemDisplayName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}
