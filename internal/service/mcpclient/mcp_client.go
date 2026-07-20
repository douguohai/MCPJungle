// Package mcpclient provides MCP client service functionality for the MCPJungle application.
package mcpclient

import (
	"errors"
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// McpClientService provides methods to manage MCP clients in the database.
type McpClientService struct {
	db *gorm.DB
}

func NewMCPClientService(db *gorm.DB) *McpClientService {
	return &McpClientService{db: db}
}

// ListClients returns clients visible to the given user: all clients for admins,
// only the user's own clients otherwise.
func (m *McpClientService) ListClients(userID uint, role types.UserRole) ([]*model.McpClient, error) {
	var clients []*model.McpClient
	q := m.db
	if role != types.UserRoleAdmin {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

// CreateClient creates a new MCP client. UserID must be set by the caller (handler).
func (m *McpClientService) CreateClient(client model.McpClient) (*model.McpClient, error) {
	if client.AccessToken != "" {
		if err := internal.ValidateAccessToken(client.AccessToken); err != nil {
			return nil, fmt.Errorf("invalid access token: %v: %w", err, apierrors.ErrInvalidInput)
		}
	} else {
		token, err := internal.GenerateAccessToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate access token: %w", err)
		}
		client.AccessToken = token
	}
	if client.AllowList == nil {
		client.AllowList = []byte("[]")
	}
	if err := m.db.Create(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

// GetClientByToken retrieves an MCP client by its access token. Used on the MCP
// proxy hot path (no user context available there).
func (m *McpClientService) GetClientByToken(token string) (*model.McpClient, error) {
	var client model.McpClient
	if err := m.db.Where("access_token = ?", token).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("client not found: %w", apierrors.ErrNotFound)
		}
		return nil, err
	}
	return &client, nil
}

// DeleteClient removes a client. Admins can delete any client; others only their own.
func (m *McpClientService) DeleteClient(userID uint, role types.UserRole, name string) error {
	q := m.db.Unscoped().Where("name = ?", name)
	if role != types.UserRoleAdmin {
		q = q.Where("user_id = ?", userID)
	}
	result := q.Delete(&model.McpClient{})
	return result.Error
}

// UpdateClient updates a client's AllowList (and access token if supplied).
// Admins can update any client; others only their own.
func (m *McpClientService) UpdateClient(userID uint, role types.UserRole, updated model.McpClient) (*model.McpClient, error) {
	q := m.db.Where("name = ?", updated.Name)
	if role != types.UserRoleAdmin {
		q = q.Where("user_id = ?", userID)
	}
	var client model.McpClient
	if err := q.First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("client not found: %w", apierrors.ErrNotFound)
		}
		return nil, err
	}

	if updated.AllowList != nil {
		client.AllowList = updated.AllowList
	}
	if updated.AccessToken != "" {
		if err := internal.ValidateAccessToken(updated.AccessToken); err != nil {
			return nil, fmt.Errorf("invalid access token: %v: %w", err, apierrors.ErrInvalidInput)
		}
		client.AccessToken = updated.AccessToken
	}

	if err := m.db.Save(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}
