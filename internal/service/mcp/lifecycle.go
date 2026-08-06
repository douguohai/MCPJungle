package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/datatypes"
)

// slugify produces a URL-friendly identifier from the given name.
func slugify(name string) string {
	s := strings.ToLower(name)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Proxy-only registration helpers (read from DB, add to proxy server)
// ---------------------------------------------------------------------------

// registerToProxyTools loads all tools for a server from the DB and adds them
// to the MCP proxy server. Tools are loaded in their canonical form
// (serverName__toolName).
func (m *MCPService) registerToProxyTools(s *model.McpServer) error {
	var tools []model.Tool
	if err := m.db.Where("server_id = ?", s.ID).Find(&tools).Error; err != nil {
		return fmt.Errorf("failed to load tools for server %s from DB: %w", s.Name, err)
	}
	for i := range tools {
		mcpTool, err := convertToolModelToMcpObject(&tools[i])
		if err != nil {
			return fmt.Errorf("failed to convert tool model %s: %w", tools[i].Name, err)
		}
		mcpTool.Name = mergeServerToolNames(s.Name, tools[i].Name)
		if s.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddTool(mcpTool, m.MCPProxyToolCallHandler)
		} else {
			m.mcpProxyServer.AddTool(mcpTool, m.MCPProxyToolCallHandler)
		}
		m.addToolInstance(mcpTool)
		m.notifyToolAddition(mcpTool.Name)
	}
	return nil
}

// registerToProxyPrompts loads all prompts for a server from the DB and adds
// them to the MCP proxy server.
func (m *MCPService) registerToProxyPrompts(s *model.McpServer) error {
	var prompts []model.Prompt
	if err := m.db.Where("server_id = ?", s.ID).Find(&prompts).Error; err != nil {
		return fmt.Errorf("failed to load prompts for server %s from DB: %w", s.Name, err)
	}
	for i := range prompts {
		mcpPrompt, err := convertPromptModelToMcpObject(&prompts[i])
		if err != nil {
			log.Printf("[WARN] failed to convert prompt model %s: %v", prompts[i].Name, err)
			continue
		}
		mcpPrompt.Name = mergeServerPromptNames(s.Name, prompts[i].Name)
		if s.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddPrompt(mcpPrompt, m.mcpProxyPromptHandler)
		} else {
			m.mcpProxyServer.AddPrompt(mcpPrompt, m.mcpProxyPromptHandler)
		}
	}
	return nil
}

// registerToProxyResources loads all resources for a server from the DB and
// adds them to the MCP proxy server.
func (m *MCPService) registerToProxyResources(s *model.McpServer) error {
	var resources []model.Resource
	if err := m.db.Where("server_id = ?", s.ID).Find(&resources).Error; err != nil {
		return fmt.Errorf("failed to load resources for server %s from DB: %w", s.Name, err)
	}
	for i := range resources {
		mcpResource, err := convertResourceModelToMcpObject(&resources[i])
		if err != nil {
			log.Printf("[WARN] failed to convert resource model %s: %v", resources[i].URI, err)
			continue
		}
		mcpResource.Name = mergeServerResourceNames(s.Name, resources[i].Name)
		mcpResource.URI = resources[i].URI
		if s.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddResource(mcpResource, m.mcpProxyResourceHandler)
		} else {
			m.mcpProxyServer.AddResource(mcpResource, m.mcpProxyResourceHandler)
		}
	}
	return nil
}

// registerToProxy loads all persisted capabilities (tools, prompts, resources)
// for a server from the DB and registers them with the MCP proxy server.
// Best-effort: prompt/resource errors are logged but do not fail the operation.
func (m *MCPService) registerToProxy(s *model.McpServer) error {
	if err := m.registerToProxyTools(s); err != nil {
		return err
	}
	if err := m.registerToProxyPrompts(s); err != nil {
		log.Printf("[WARN] failed to register prompts to proxy for server %s: %v", s.Name, err)
	}
	if err := m.registerToProxyResources(s); err != nil {
		log.Printf("[WARN] failed to register resources to proxy for server %s: %v", s.Name, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Capability persistence helpers (upstream -> DB, no proxy registration)
// ---------------------------------------------------------------------------

// persistServerCapabilities connects to the upstream MCP server, discovers
// tools, prompts and resources, and persists them to the DB. It does NOT
// register anything with the MCP proxy server -- that is the responsibility
// of PublishServer.
//
// If validateOnly is true the method skips proxy registration even for tools
// (which the existing registerServer* helpers add by default). This is used
// by ValidateServer so that capabilities are persisted but not exposed until
// the server is explicitly published.
func (m *MCPService) persistServerCapabilities(
	ctx context.Context,
	s *model.McpServer,
	useStoredUpstreamAuth bool,
) error {
	mcpClient, err := createMcpServerConnectionWithDB(
		ctx,
		m.db,
		s,
		m.mcpServerInitReqTimeoutSec,
		useStoredUpstreamAuth,
	)
	if err != nil {
		return err
	}
	defer mcpClient.Close()

	if err := m.discoverAndPersistTools(ctx, s, mcpClient); err != nil {
		return fmt.Errorf("failed to discover tools for MCP server %s: %w", s.Name, err)
	}

	// Prompts and resources are best-effort.
	if mcpClient.GetServerCapabilities().Prompts != nil {
		if err := m.discoverAndPersistPrompts(ctx, s, mcpClient); err != nil {
			log.Printf("[WARN] failed to discover prompts for MCP server %s: %v", s.Name, err)
		}
	}
	if mcpClient.GetServerCapabilities().Resources != nil {
		if err := m.discoverAndPersistResources(ctx, s, mcpClient); err != nil {
			log.Printf("[WARN] failed to discover resources for MCP server %s: %v", s.Name, err)
		}
	}

	return nil
}

// discoverAndPersistTools calls ListTools on the upstream server and persists
// each tool to the DB. It does NOT add tools to the MCP proxy server.
func (m *MCPService) discoverAndPersistTools(
	ctx context.Context,
	s *model.McpServer,
	c *client.Client,
) error {
	resp, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("failed to fetch tools from MCP server %s: %w", s.Name, err)
	}
	for _, tool := range resp.Tools {
		canonicalToolName := mergeServerToolNames(s.Name, tool.GetName())
		jsonSchema, _ := json.Marshal(tool.InputSchema)
		annotationsJSON, _ := json.Marshal(tool.Annotations)

		t := &model.Tool{
			ServerID:    s.ID,
			Name:        tool.GetName(),
			Description: tool.Description,
			InputSchema: jsonSchema,
			Annotations: annotationsJSON,
		}
		if err := m.db.Create(t).Error; err != nil {
			log.Printf("[ERROR] failed to persist tool %s in DB: %v", canonicalToolName, err)
			continue
		}
	}
	return nil
}

// discoverAndPersistPrompts calls ListPrompts on the upstream server and
// persists each prompt to the DB. It does NOT add prompts to the proxy.
func (m *MCPService) discoverAndPersistPrompts(
	ctx context.Context,
	s *model.McpServer,
	c *client.Client,
) error {
	resp, err := c.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		return fmt.Errorf("failed to fetch prompts from MCP server %s: %w", s.Name, err)
	}
	for _, prompt := range resp.Prompts {
		canonicalPromptName := mergeServerPromptNames(s.Name, prompt.GetName())
		jsonArguments, _ := json.Marshal(prompt.Arguments)

		p := &model.Prompt{
			ServerID:    s.ID,
			Name:        prompt.GetName(),
			Description: prompt.Description,
			Arguments:   jsonArguments,
		}
		if err := m.db.Create(p).Error; err != nil {
			log.Printf("[ERROR] failed to persist prompt %s in DB: %v", canonicalPromptName, err)
			continue
		}
	}
	return nil
}

// discoverAndPersistResources calls ListResources on the upstream server and
// persists each resource to the DB. It does NOT add resources to the proxy.
func (m *MCPService) discoverAndPersistResources(
	ctx context.Context,
	s *model.McpServer,
	c *client.Client,
) error {
	resp, err := c.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		return fmt.Errorf("failed to fetch resources from MCP server %s: %w", s.Name, err)
	}
	for _, resource := range resp.Resources {
		canonicalResourceName := mergeServerResourceNames(s.Name, resource.GetName())
		annotationsJSON, _ := json.Marshal(resource.Annotations)
		metaJSON, _ := json.Marshal(resource.Meta)

		r := &model.Resource{
			ServerID:    s.ID,
			URI:         buildResourceURI(s.Name, resource.URI),
			OriginalURI: resource.URI,
			Name:        resource.GetName(),
			Description: resource.Description,
			MIMEType:    resource.MIMEType,
			Annotations: annotationsJSON,
			Meta:        metaJSON,
		}
		if err := m.db.Create(r).Error; err != nil {
			log.Printf("[ERROR] failed to persist resource %s (%s) in DB: %v", canonicalResourceName, resource.URI, err)
			continue
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Lifecycle service methods
// ---------------------------------------------------------------------------

// CreateDraftServer creates an McpServer with status=draft. It does NOT connect
// to the upstream server or discover any capabilities.
func (m *MCPService) CreateDraftServer(
	name, description string,
	transport types.McpServerTransport,
	config datatypes.JSON,
	sessionMode types.SessionMode,
) (*model.McpServer, error) {
	if err := validateServerName(name); err != nil {
		return nil, err
	}
	if sessionMode == "" {
		sessionMode = types.SessionModeStateless
	}
	s := &model.McpServer{
		Name:        name,
		Description: description,
		Transport:   transport,
		Config:      config,
		SessionMode: sessionMode,
		Status:      model.StatusDraft,
		Slug:        slugify(name),
	}
	if err := m.db.Create(s).Error; err != nil {
		return nil, fmt.Errorf("failed to create draft MCP server: %w", err)
	}
	return s, nil
}

// ValidateServer performs upstream validation of a draft or previously-failed
// server. It is safe to call from a goroutine (async UI flow) or synchronously.
//
// Steps:
//  1. Set status = validating
//  2. Connect to upstream MCP server (Initialize + ListTools/Prompts/Resources)
//  3. On success: persist capabilities to DB, set status = pending_publish,
//     record LastValidatedAt
//  4. On failure: set status = validation_failed, record LastErrorSummary
//
// NOTE: capabilities are persisted to the DB during validation but are NOT
// registered with the MCP proxy server. Use PublishServer for that.
//
// useStoredUpstreamAuth controls whether stored upstream OAuth credentials
// should be attached to the connection attempt. Most callers should pass false;
// the OAuth finalization flow passes true.
func (m *MCPService) ValidateServer(ctx context.Context, serverID uint, useStoredUpstreamAuth bool) error {
	s, err := m.GetMcpServerByID(serverID)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}
	if s.Status != model.StatusDraft && s.Status != model.StatusValidationFailed {
		return fmt.Errorf(
			"cannot validate server in status %q (must be %q or %q)",
			s.Status, model.StatusDraft, model.StatusValidationFailed,
		)
	}

	// Transition to validating.
	s.Status = model.StatusValidating
	s.LastErrorSummary = ""
	if err := m.db.Save(s).Error; err != nil {
		return fmt.Errorf("failed to set server status to validating: %w", err)
	}

	// Discover and persist capabilities (no proxy registration).
	if err := m.persistServerCapabilities(ctx, s, useStoredUpstreamAuth); err != nil {
		s.Status = model.StatusValidationFailed
		errSummary := err.Error()
		if len(errSummary) > 1000 {
			errSummary = errSummary[:1000]
		}
		s.LastErrorSummary = errSummary
		if saveErr := m.db.Save(s).Error; saveErr != nil {
			log.Printf("[lifecycle] failed to persist validation_failed status for server %s: %v", s.Name, saveErr)
		}
		return fmt.Errorf("validation failed: %w", err)
	}

	// Success.
	now := time.Now()
	s.Status = model.StatusPendingPublish
	s.LastValidatedAt = &now
	s.LastErrorSummary = ""
	if err := m.db.Save(s).Error; err != nil {
		return fmt.Errorf("failed to update server status to pending_publish: %w", err)
	}
	return nil
}

// PublishServer transitions a server from pending_publish to online and
// registers its persisted capabilities (tools, prompts, resources) with the
// MCP proxy server so they become available to clients.
func (m *MCPService) PublishServer(serverID uint) error {
	s, err := m.GetMcpServerByID(serverID)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}
	if s.Status != model.StatusPendingPublish {
		return fmt.Errorf("cannot publish server in status %q (must be %q)", s.Status, model.StatusPendingPublish)
	}

	s.Status = model.StatusOnline
	if err := m.db.Save(s).Error; err != nil {
		return fmt.Errorf("failed to set server status to online: %w", err)
	}

	// Load persisted capabilities into the proxy.
	if err := m.registerToProxy(s); err != nil {
		return fmt.Errorf("failed to register capabilities to proxy: %w", err)
	}
	return nil
}

// DisableServer transitions a server from online or unhealthy to disabled and
// removes its capabilities from the MCP proxy server.
func (m *MCPService) DisableServer(name string) error {
	if err := validateServerName(name); err != nil {
		return err
	}
	s, err := m.GetMcpServer(name)
	if err != nil {
		return err
	}
	if s.Status != model.StatusOnline && s.Status != model.StatusUnhealthy {
		return fmt.Errorf("cannot disable server in status %q (must be %q or %q)", s.Status, model.StatusOnline, model.StatusUnhealthy)
	}

	// Remove from proxy.
	if err := m.deregisterServerTools(s); err != nil {
		return fmt.Errorf("failed to deregister tools from proxy: %w", err)
	}
	if err := m.deregisterServerPrompts(s); err != nil {
		return fmt.Errorf("failed to deregister prompts from proxy: %w", err)
	}
	if err := m.deregisterServerResources(s); err != nil {
		return fmt.Errorf("failed to deregister resources from proxy: %w", err)
	}

	// Close any stateful session.
	m.sessionManager.CloseSession(name)

	s.Status = model.StatusDisabled
	if err := m.db.Save(s).Error; err != nil {
		return fmt.Errorf("failed to set server status to disabled: %w", err)
	}
	return nil
}

// ArchiveServer transitions a server to archived from any status and removes
// its capabilities from the MCP proxy server.
func (m *MCPService) ArchiveServer(name string) error {
	if err := validateServerName(name); err != nil {
		return err
	}
	s, err := m.GetMcpServer(name)
	if err != nil {
		return err
	}

	// Remove from proxy (only meaningful if the server is currently online).
	if s.Status == model.StatusOnline || s.Status == model.StatusUnhealthy {
		if err := m.deregisterServerTools(s); err != nil {
			return fmt.Errorf("failed to deregister tools from proxy: %w", err)
		}
		if err := m.deregisterServerPrompts(s); err != nil {
			return fmt.Errorf("failed to deregister prompts from proxy: %w", err)
		}
		if err := m.deregisterServerResources(s); err != nil {
			return fmt.Errorf("failed to deregister resources from proxy: %w", err)
		}
		m.sessionManager.CloseSession(name)
	}

	now := time.Now()
	s.Status = model.StatusArchived
	s.ArchivedAt = &now
	if err := m.db.Save(s).Error; err != nil {
		return fmt.Errorf("failed to set server status to archived: %w", err)
	}
	return nil
}

// UpdateServerConfig updates server configuration. If transport, config, or
// session-mode changed the server is moved to draft status (requiring
// re-validation). Metadata-only changes (description) are saved without
// changing the status.
func (m *MCPService) UpdateServerConfig(serverID uint, newConfig model.McpServer) error {
	s, err := m.GetMcpServerByID(serverID)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	configChanged := s.Transport != newConfig.Transport ||
		string(s.Config) != string(newConfig.Config) ||
		s.SessionMode != newConfig.SessionMode

	s.Description = newConfig.Description
	if configChanged {
		s.Transport = newConfig.Transport
		s.Config = newConfig.Config
		s.SessionMode = newConfig.SessionMode
		s.Status = model.StatusDraft
	}

	if err := m.db.Save(s).Error; err != nil {
		return fmt.Errorf("failed to update server config: %w", err)
	}
	return nil
}

// AddManager assigns a user as a manager (primary or backup) of an MCP server.
func (m *MCPService) AddManager(serverID, userID uint, roleType string) error {
	if roleType != "primary" && roleType != "backup" {
		return fmt.Errorf("invalid role_type %q (must be \"primary\" or \"backup\")", roleType)
	}
	mgr := &model.McpServiceManager{
		McpServerID: serverID,
		UserID:      userID,
		RoleType:    roleType,
	}
	if err := m.db.Create(mgr).Error; err != nil {
		return fmt.Errorf("failed to add manager: %w", err)
	}
	return nil
}

// RemoveManager removes a user as a manager of an MCP server.
func (m *MCPService) RemoveManager(serverID, userID uint) error {
	result := m.db.Where("mcp_server_id = ? AND user_id = ?", serverID, userID).Delete(&model.McpServiceManager{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove manager: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("manager not found for server %d user %d", serverID, userID)
	}
	return nil
}

// ListManagers returns all managers for a given MCP server.
func (m *MCPService) ListManagers(serverID uint) ([]model.McpServiceManager, error) {
	var managers []model.McpServiceManager
	if err := m.db.Where("mcp_server_id = ?", serverID).Find(&managers).Error; err != nil {
		return nil, fmt.Errorf("failed to list managers: %w", err)
	}
	return managers, nil
}
