package mcp

import (
	"context"
	"errors"
	"fmt"

	mcpgotransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// RegisterMcpServerWithOAuthSupport attempts to register a server immediately and,
// if upstream OAuth authorization is required, persists a pending auth session that
// can be completed later.
func (m *MCPService) RegisterMcpServerWithOAuthSupport(
	ctx context.Context,
	input *types.RegisterServerInput,
	s *model.McpServer,
	force bool,
	initiatedBy string,
) error {
	// First attempt to register the server without involving any oauth flows.
	// This covers all mcp servers that DO NOT specifically require oauth-based authentication.
	err := m.registerMcpServerWithoutOAuth(ctx, s)
	if err == nil {
		return nil
	}

	// If registration failed and the error is not related to oauth, return error.
	if s.Transport != types.TransportStreamableHTTP && s.Transport != types.TransportSSE {
		return err
	}
	if !errors.Is(err, mcpgotransport.ErrUnauthorized) {
		return err
	}

	// registration failed due to missing/invalid upstream OAuth credentials.
	// notify the client so the oauth flow can be initiated.
	return m.bootstrapUpstreamOAuth(ctx, input, s, force, initiatedBy)
}

// registerMcpServerWithoutOAuth performs the initial upstream registration
// attempt without attaching any stored upstream OAuth credentials.
func (m *MCPService) registerMcpServerWithoutOAuth(ctx context.Context, s *model.McpServer) error {
	return m.registerMcpServer(ctx, s, false)
}

// finalizeMcpServerRegistration performs the plain MCP server registration flow
// once upstream authentication has already been satisfied and any stored
// upstream OAuth credentials should be attached to the connection attempt.
func (m *MCPService) finalizeMcpServerRegistration(ctx context.Context, s *model.McpServer) error {
	return m.registerMcpServer(ctx, s, true)
}

// registerMcpServer performs the core MCP server registration flow.
//
// Internally it delegates to the new lifecycle methods:
//
//	CreateDraftServer -> ValidateServer -> PublishServer
//
// This preserves backward compatibility for callers that expect a single
// synchronous call that results in an online server with all capabilities
// registered.
//
// This method assumes that any OAuth nuance is already handled and simply uses existing auth info.
func (m *MCPService) registerMcpServer(ctx context.Context, s *model.McpServer, useStoredUpstreamAuth bool) error {
	if err := validateServerName(s.Name); err != nil {
		return err
	}

	// Only validate URLs for transports that actually carry a URL in their config.
	switch s.Transport {
	case types.TransportStreamableHTTP:
		conf, err := s.GetStreamableHTTPConfig()
		if err != nil {
			return err
		}
		if err := validateURL(conf.URL); err != nil {
			return err
		}
	case types.TransportSSE:
		conf, err := s.GetSSEConfig()
		if err != nil {
			return err
		}
		if err := validateURL(conf.URL); err != nil {
			return err
		}
	}

	// --- Lifecycle: CreateDraft -> Validate -> Publish ---

	// 1. Create draft (persists to DB with status=draft).
	draft, err := m.CreateDraftServer(
		s.Name,
		s.Description,
		s.Transport,
		s.Config,
		s.SessionMode,
	)
	if err != nil {
		return fmt.Errorf("failed to create draft MCP server: %w", err)
	}

	// Carry over the generated ID so downstream callers can reference the
	// server that was actually persisted.
	s.ID = draft.ID
	s.CreatedAt = draft.CreatedAt
	s.UpdatedAt = draft.UpdatedAt
	s.DeletedAt = draft.DeletedAt

	// 2. Validate: connect upstream, discover capabilities, persist to DB.
	if err := m.ValidateServer(ctx, draft.ID, useStoredUpstreamAuth); err != nil {
		// Clean up the draft if validation fails.
		_ = m.db.Unscoped().Delete(draft).Error
		return err
	}

	// 3. Publish: set status=online, register capabilities to proxy.
	if err := m.PublishServer(draft.ID); err != nil {
		return fmt.Errorf("failed to publish MCP server: %w", err)
	}

	// Keep backward compat: reflect the final state on the caller's model.
	s.Status = model.StatusOnline

	return nil
}

// DeregisterMcpServer deregisters an MCP server from the database.
// It also deregisters all the tools, prompts and resources registered by the server.
// If even a single tool, prompt or resource fails to deregister, the server deregistration fails.
// Deregistered tools, prompts and resources are also removed from the MCP proxy server.
// Any stateful sessions associated with this server are also closed.
func (m *MCPService) DeregisterMcpServer(name string) error {
	s, err := m.GetMcpServer(name)
	if err != nil {
		return fmt.Errorf("failed to get MCP server %s from DB: %w", name, err)
	}
	if err := m.deregisterServerTools(s); err != nil {
		return fmt.Errorf(
			"failed to deregister tools for server %s, cannot proceed with server deregistration: %w",
			name,
			err,
		)
	}
	if err := m.deregisterServerPrompts(s); err != nil {
		return fmt.Errorf(
			"failed to deregister prompts for server %s, cannot proceed with server deregistration: %w",
			name,
			err,
		)
	}
	if err := m.deregisterServerResources(s); err != nil {
		return fmt.Errorf(
			"failed to deregister resources for server %s, cannot proceed with server deregistration: %w",
			name,
			err,
		)
	}
	if err := m.db.Unscoped().Delete(s).Error; err != nil {
		return fmt.Errorf("failed to deregister server %s: %w", name, err)
	}
	if err := m.db.Unscoped().Where("server_name = ?", name).Delete(&model.UpstreamOAuthToken{}).Error; err != nil {
		return fmt.Errorf("failed to remove upstream OAuth tokens for server %s: %w", name, err)
	}
	if err := m.db.Unscoped().Where("server_name = ?", name).Delete(&model.UpstreamOAuthPendingSession{}).Error; err != nil {
		return fmt.Errorf("failed to remove pending upstream OAuth sessions for server %s: %w", name, err)
	}

	// Close any stateful session associated with this server
	m.sessionManager.CloseSession(name)

	return nil
}

// ListMcpServers returns all registered MCP servers.
func (m *MCPService) ListMcpServers() ([]model.McpServer, error) {
	var servers []model.McpServer
	if err := m.db.Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

// GetMcpServer fetches a server from the database by name.
func (m *MCPService) GetMcpServer(name string) (*model.McpServer, error) {
	var serverModel model.McpServer
	if err := m.db.Where("name = ?", name).First(&serverModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("MCP server %s not found: %w", name, apierrors.ErrNotFound)
		}
		return nil, err
	}
	return &serverModel, nil
}

// GetMcpServerByID fetches a server from the database by primary key.
func (m *MCPService) GetMcpServerByID(id uint) (*model.McpServer, error) {
	var serverModel model.McpServer
	if err := m.db.First(&serverModel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("MCP server not found: %w", apierrors.ErrNotFound)
		}
		return nil, err
	}
	return &serverModel, nil
}

// EnableMcpServer enables all tools, prompts and resources registered by the given MCP server.
// It returns the names of the enabled tools and prompts.
// If even a single tool, prompt or resource fails to enable, the operation fails.
func (m *MCPService) EnableMcpServer(name string) ([]string, []string, error) {
	if err := validateServerName(name); err != nil {
		return nil, nil, err
	}
	if err := m.setMcpServerEnabled(name, true); err != nil {
		return nil, nil, fmt.Errorf("failed to mark server %s enabled: %w", name, err)
	}
	toolsEnabled, err := m.EnableTools(name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to enable tools for server %s: %w", name, err)
	}
	promptsEnabled, err := m.EnablePrompts(name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to enable prompts for server %s: %w", name, err)
	}
	if _, err := m.EnableResources(name); err != nil {
		return nil, nil, fmt.Errorf("failed to enable resources for server %s: %w", name, err)
	}
	return toolsEnabled, promptsEnabled, nil
}

// DisableMcpServer disables all tools, prompts and resources registered by the given MCP server.
// It returns the names of the disabled tools and prompts.
// If even a single tool, prompt or resource fails to disable, the operation fails.
func (m *MCPService) DisableMcpServer(name string) ([]string, []string, error) {
	if err := validateServerName(name); err != nil {
		return nil, nil, err
	}
	if err := m.setMcpServerEnabled(name, false); err != nil {
		return nil, nil, fmt.Errorf("failed to mark server %s disabled: %w", name, err)
	}
	toolsDisabled, err := m.DisableTools(name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to disable tools for server %s: %w", name, err)
	}
	promptsDisabled, err := m.DisablePrompts(name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to disable prompts for server %s: %w", name, err)
	}
	if _, err := m.DisableResources(name); err != nil {
		return nil, nil, fmt.Errorf("failed to disable resources for server %s: %w", name, err)
	}
	return toolsDisabled, promptsDisabled, nil
}

// SetDashboardServerStatus reuses the standard server enable/disable flow so dashboard
// toggles stay consistent with CLI semantics and cascade to the server's tools, prompts,
// and resources.
func (m *MCPService) SetDashboardServerStatus(name string, enabled bool) error {
	if enabled {
		_, _, err := m.EnableMcpServer(name)
		return err
	}
	_, _, err := m.DisableMcpServer(name)
	return err
}

// setMcpServerEnabled is a helper that updates the enabled status of the MCP server in the DB.
func (m *MCPService) setMcpServerEnabled(name string, enabled bool) error {
	server, err := m.GetMcpServer(name)
	if err != nil {
		return err
	}
	newStatus := model.StatusOnline
	if !enabled {
		newStatus = model.StatusDisabled
	}
	if server.Status == newStatus {
		return nil
	}
	server.Status = newStatus
	if err := m.db.Save(server).Error; err != nil {
		return fmt.Errorf("failed to set server %s status=%s: %w", name, newStatus, err)
	}
	return nil
}
