package mcp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

func (m *MCPService) authorizeProxyServerAccess(ctx context.Context, serverName string) error {
	serverMode := ctx.Value("mode").(model.ServerMode)
	if !model.IsEnterpriseMode(serverMode) {
		return nil
	}

	effective, _ := ctx.Value("effective_services").(map[string]bool)
	if effective == nil || !effective[serverName] {
		return fmt.Errorf("access denied: device token is not authorized to access MCP server %s", serverName)
	}

	// Check lifecycle status (design doc §9.4: unhealthy servers return a
	// distinct UPSTREAM_UNAVAILABLE error, not a permission error).
	var srv model.McpServer
	if err := m.db.Where("name = ?", serverName).First(&srv).Error; err != nil {
		return fmt.Errorf("MCP server %s not found", serverName)
	}
	switch srv.Status {
	case model.StatusOnline:
		return nil
	case model.StatusUnhealthy:
		return fmt.Errorf("MCP server %s is temporarily unavailable (health check failing): %s",
			serverName, srv.LastErrorSummary)
	default:
		return fmt.Errorf("MCP server %s is %s", serverName, srv.Status)
	}
}

// recordUserCall increments the calling user's per-server call counter. Only the
// count is stored. Best-effort: errors are ignored so analytics never breaks a
// tool call.
func (m *MCPService) recordUserCall(ctx context.Context, serverName string) {
	dt, ok := ctx.Value("device_token").(*model.DeviceToken)
	if !ok || dt == nil || dt.UserID == 0 {
		return // dev mode or token without an owner
	}
	today := time.Now().UTC().Format("2006-01-02")
	var stat model.UserCallStat
	err := m.db.Where("user_id = ? AND server_name = ? AND date = ?", dt.UserID, serverName, today).First(&stat).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_ = m.db.Create(&model.UserCallStat{UserID: dt.UserID, ServerName: serverName, Date: today, Count: 1}).Error
		return
	}
	if err == nil {
		_ = m.db.Model(&stat).Update("count", gorm.Expr("count + 1")).Error
	}
}

// recordCallEvent writes a CallEvent for analytics.  Best-effort: errors are
// logged but never returned so that analytics never blocks an MCP call.
func (m *MCPService) recordCallEvent(event *model.CallEvent) {
	if m.callEventService == nil {
		return
	}
	if err := m.callEventService.RecordEvent(event); err != nil {
		log.Printf("[callevent] failed to record call event: %v", err)
	}
}

// callEventFromContext extracts user/token metadata from context for building
// a CallEvent.  Returns zero values in dev mode or when metadata is missing.
func callEventFromContext(ctx context.Context) (userID, deviceTokenID uint, sourceIP, clientID string) {
	if dt, ok := ctx.Value("device_token").(*model.DeviceToken); ok && dt != nil {
		userID = dt.UserID
		deviceTokenID = dt.ID
		clientID = dt.ClientID
	}
	if u, ok := ctx.Value("user").(*model.User); ok && u != nil && userID == 0 {
		userID = u.ID
	}
	return
}

// callResultFromError maps an error to a CallEvent result string.
func callResultFromError(err error) string {
	if err == nil {
		return model.CallEventResultSuccess
	}
	if errors.Is(err, apierrors.ErrInvalidInput) {
		return model.CallEventResultPermissionDenied
	}
	errMsg := err.Error()
	if errors.Is(err, context.DeadlineExceeded) {
		return model.CallEventResultTimeout
	}
	if contains(errMsg, "access denied") || contains(errMsg, "not authorized") {
		return model.CallEventResultPermissionDenied
	}
	if contains(errMsg, "disabled") || contains(errMsg, "archived") {
		return model.CallEventResultServiceDisabled
	}
	return model.CallEventResultUpstreamError
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSlow(s, substr))
}

func containsSlow(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// MCPProxyToolCallHandler handles tool calls for the MCP proxy server
// by forwarding the request to the appropriate upstream MCP server and
// relaying the response back.
func (m *MCPService) MCPProxyToolCallHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	started := time.Now()
	outcome := telemetry.ToolCallOutcomeSuccess

	name := request.Params.Name
	serverName, toolName, ok := splitServerToolName(name)
	if !ok {
		return nil, fmt.Errorf("tool name does not contain a %s separator: %w", serverToolNameSep, apierrors.ErrInvalidInput)
	}

	if err := m.authorizeProxyServerAccess(ctx, serverName); err != nil {
		return nil, err
	}

	// Record per-user call count for analytics (only the count, never content).
	m.recordUserCall(ctx, serverName)

	// Build call event metadata (will be persisted after the upstream call).
	userID, deviceTokenID, _, clientID := callEventFromContext(ctx)
	var serverID uint
	if srv, err := m.GetMcpServer(serverName); err == nil {
		serverID = srv.ID
	}

	// Record the tool call metrics at the end of the function
	defer func() {
		m.metrics.RecordToolCall(ctx, serverName, toolName, outcome, time.Since(started))
	}()

	// get the MCP server details from the database
	server, err := m.GetMcpServer(serverName)
	if err != nil {
		// TODO: differentiate between "server not found" and other errors.
		// server not found is not an internal error, so outcome should be success.
		outcome = telemetry.ToolCallOutcomeError

		// Record call event for the error case.
		latency := time.Since(started)
		m.recordCallEvent(&model.CallEvent{
			RequestID:         uuid.New().String(),
			CallTime:          started,
			UserID:            userID,
			DeviceTokenID:     deviceTokenID,
			McpServiceID:      serverID,
			ToolName:          toolName,
			CallType:          "tool",
			Result:            callResultFromError(err),
			LatencyMs:         latency.Milliseconds(),
			UpstreamLatencyMs: 0,
			ErrorCode:         "server_not_found",
			ClientID:          clientID,
		})

		return nil, fmt.Errorf(
			"failed to get details about MCP server %s from DB: %w", serverName, err,
		)
	}

	session, err := m.getSession(ctx, server)
	if err != nil {
		outcome = telemetry.ToolCallOutcomeError

		latency := time.Since(started)
		m.recordCallEvent(&model.CallEvent{
			RequestID:         uuid.New().String(),
			CallTime:          started,
			UserID:            userID,
			DeviceTokenID:     deviceTokenID,
			McpServiceID:      serverID,
			ToolName:          toolName,
			CallType:          "tool",
			Result:            callResultFromError(err),
			LatencyMs:         latency.Milliseconds(),
			UpstreamLatencyMs: 0,
			ErrorCode:         "session_error",
			ClientID:          clientID,
		})

		return nil, err
	}
	defer session.closeIfApplicable()

	// Ensure the tool name is set correctly, ie, without the server name prefix
	request.Params.Name = toolName
	// Do not let any client-sent headers get forwarded to the upstream MCP server.
	// See https://github.com/mcpjungle/MCPJungle/issues/252
	request.Header = nil

	upstreamStart := time.Now()
	res, err := session.client.CallTool(ctx, request)
	upstreamLatency := time.Since(upstreamStart)
	if err != nil {
		outcome = telemetry.ToolCallOutcomeError
		session.invalidateOnError(err) // Invalidate unhealthy stateful sessions
	}

	// Record call event (best-effort).
	totalLatency := time.Since(started)
	m.recordCallEvent(&model.CallEvent{
		RequestID:         uuid.New().String(),
		CallTime:          started,
		UserID:            userID,
		DeviceTokenID:     deviceTokenID,
		McpServiceID:      serverID,
		ToolName:          toolName,
		CallType:          "tool",
		Result:            callResultFromError(err),
		LatencyMs:         totalLatency.Milliseconds(),
		UpstreamLatencyMs: upstreamLatency.Milliseconds(),
		ClientID:          clientID,
	})

	// forward the request to the upstream MCP server and relay the response back
	return res, err
}

// mcpProxyResourceHandler handles resource reads for the MCP proxy server
// by forwarding the request to the appropriate upstream MCP server and
// relaying the response back.
func (m *MCPService) mcpProxyResourceHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	started := time.Now()

	// get the upstream mcp server and original resource uri for the requested resource uri
	resource, err := m.GetResource(request.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s from DB: %w", request.Params.URI, err)
	}

	if err := m.authorizeProxyServerAccess(ctx, resource.Server.Name); err != nil {
		return nil, err
	}

	userID, deviceTokenID, _, clientID := callEventFromContext(ctx)
	serverID := resource.Server.ID
	resourceName := request.Params.URI

	session, err := m.getSession(ctx, &resource.Server)
	if err != nil {
		latency := time.Since(started)
		m.recordCallEvent(&model.CallEvent{
			RequestID:     uuid.New().String(),
			CallTime:      started,
			UserID:        userID,
			DeviceTokenID: deviceTokenID,
			McpServiceID:  serverID,
			ToolName:      resourceName,
			CallType:      "resource",
			Result:        callResultFromError(err),
			LatencyMs:     latency.Milliseconds(),
			ErrorCode:     "session_error",
			ClientID:      clientID,
		})
		return nil, err
	}
	defer session.closeIfApplicable()

	request.Params.URI = resource.OriginalURI

	// Do not let any client-sent headers get forwarded to the upstream MCP server.
	request.Header = nil

	upstreamStart := time.Now()
	res, err := session.client.ReadResource(ctx, request)
	upstreamLatency := time.Since(upstreamStart)
	if err != nil {
		session.invalidateOnError(err)
	}

	// Record call event (best-effort).
	totalLatency := time.Since(started)
	m.recordCallEvent(&model.CallEvent{
		RequestID:         uuid.New().String(),
		CallTime:          started,
		UserID:            userID,
		DeviceTokenID:     deviceTokenID,
		McpServiceID:      serverID,
		ToolName:          resourceName,
		CallType:          "resource",
		Result:            callResultFromError(err),
		LatencyMs:         totalLatency.Milliseconds(),
		UpstreamLatencyMs: upstreamLatency.Milliseconds(),
		ClientID:          clientID,
	})

	if err != nil {
		return nil, err
	}

	return rewriteResourceContentsURI(res.Contents, resource.URI), nil
}

// mcpProxyPromptHandler handles prompt requests for the MCP proxy server
// by forwarding the request to the appropriate upstream MCP server and
// relaying the response back.
func (m *MCPService) mcpProxyPromptHandler(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	started := time.Now()
	outcome := telemetry.PromptCallOutcomeSuccess

	name := request.Params.Name
	serverName, promptName, ok := splitServerPromptName(name)
	if !ok {
		return nil, fmt.Errorf("prompt name does not contain a %s separator: %w", serverPromptNameSep, apierrors.ErrInvalidInput)
	}

	if err := m.authorizeProxyServerAccess(ctx, serverName); err != nil {
		return nil, err
	}

	// Build call event metadata.
	userID, deviceTokenID, _, clientID := callEventFromContext(ctx)
	var serverID uint
	if srv, err := m.GetMcpServer(serverName); err == nil {
		serverID = srv.ID
	}

	// Record the prompt call metrics at the end of the function
	defer func() {
		m.metrics.RecordPromptCall(ctx, serverName, promptName, outcome, time.Since(started))
	}()

	// get the MCP server details from the database
	server, err := m.GetMcpServer(serverName)
	if err != nil {
		// TODO: differentiate between "server not found" and other errors.
		// server not found is not an internal error, so outcome should be success.
		outcome = telemetry.PromptCallOutcomeError

		latency := time.Since(started)
		m.recordCallEvent(&model.CallEvent{
			RequestID:     uuid.New().String(),
			CallTime:      started,
			UserID:        userID,
			DeviceTokenID: deviceTokenID,
			McpServiceID:  serverID,
			ToolName:      promptName,
			CallType:      "prompt",
			Result:        callResultFromError(err),
			LatencyMs:     latency.Milliseconds(),
			ErrorCode:     "server_not_found",
			ClientID:      clientID,
		})

		return nil, fmt.Errorf(
			"failed to get details about MCP server %s from DB: %w", serverName, err,
		)
	}

	session, err := m.getSession(ctx, server)
	if err != nil {
		outcome = telemetry.PromptCallOutcomeError

		latency := time.Since(started)
		m.recordCallEvent(&model.CallEvent{
			RequestID:     uuid.New().String(),
			CallTime:      started,
			UserID:        userID,
			DeviceTokenID: deviceTokenID,
			McpServiceID:  serverID,
			ToolName:      promptName,
			CallType:      "prompt",
			Result:        callResultFromError(err),
			LatencyMs:     latency.Milliseconds(),
			ErrorCode:     "session_error",
			ClientID:      clientID,
		})

		return nil, err
	}
	defer session.closeIfApplicable()

	// Ensure the prompt name is set correctly, ie, without the server name prefix
	request.Params.Name = promptName
	// Do not let any client-sent headers get forwarded to the upstream MCP server.
	request.Header = nil

	// forward the request to the upstream MCP server and relay the response back
	upstreamStart := time.Now()
	res, err := session.client.GetPrompt(ctx, request)
	upstreamLatency := time.Since(upstreamStart)
	if err != nil {
		outcome = telemetry.PromptCallOutcomeError
		session.invalidateOnError(err) // Invalidate unhealthy stateful sessions
	}

	// Record call event (best-effort).
	totalLatency := time.Since(started)
	m.recordCallEvent(&model.CallEvent{
		RequestID:         uuid.New().String(),
		CallTime:          started,
		UserID:            userID,
		DeviceTokenID:     deviceTokenID,
		McpServiceID:      serverID,
		ToolName:          promptName,
		CallType:          "prompt",
		Result:            callResultFromError(err),
		LatencyMs:         totalLatency.Milliseconds(),
		UpstreamLatencyMs: upstreamLatency.Milliseconds(),
		ClientID:          clientID,
	})

	return res, err
}

// initMCPProxyServer initializes the MCP proxy server.
// It loads all the registered MCP tools, prompts and resources from the database
// into the proxy server. Only servers with status=online are loaded.
func (m *MCPService) initMCPProxyServer() error {
	mcpServerModelsCache := make(map[string]*model.McpServer)

	// isActiveServer checks whether the given server is online or unhealthy.
	// Unhealthy servers remain in the proxy so that calls return a clear
	// UPSTREAM_UNAVAILABLE error rather than "tool not found" (design doc §9.4).
	isActiveServer := func(serverName string) bool {
		if server, ok := mcpServerModelsCache[serverName]; ok {
			return server.Status == model.StatusOnline || server.Status == model.StatusUnhealthy
		}
		server, err := m.GetMcpServer(serverName)
		if err != nil {
			return false
		}
		mcpServerModelsCache[serverName] = server
		return server.Status == model.StatusOnline || server.Status == model.StatusUnhealthy
	}

	// Load Tools
	tools, err := m.ListTools()
	if err != nil {
		return fmt.Errorf("failed to list tools from DB: %w", err)
	}

	for _, tm := range tools {
		if !tm.Enabled {
			// do not add disabled tools to the proxy
			continue
		}

		serverName, _, _ := splitServerToolName(tm.Name)
		if !isActiveServer(serverName) {
			continue
		}

		// Add tool to the MCP proxy server
		tool, err := convertToolModelToMcpObject(&tm)
		if err != nil {
			return fmt.Errorf("failed to convert tool model to MCP object for tool %s: %w", tm.Name, err)
		}

		server := mcpServerModelsCache[serverName]

		if server.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddTool(tool, m.MCPProxyToolCallHandler)
		} else {
			m.mcpProxyServer.AddTool(tool, m.MCPProxyToolCallHandler)
		}

		m.addToolInstance(tool)
	}

	// Load prompts
	prompts, err := m.ListPrompts()
	if err != nil {
		return fmt.Errorf("failed to list prompts from DB: %w", err)
	}

	for _, pm := range prompts {
		if !pm.Enabled {
			// do not add disabled prompts to the proxy
			continue
		}

		serverName, _, _ := splitServerPromptName(pm.Name)
		if !isActiveServer(serverName) {
			continue
		}

		// Add prompt to the MCP proxy server
		prompt, err := convertPromptModelToMcpObject(&pm)
		if err != nil {
			return fmt.Errorf("failed to convert prompt model to MCP object for prompt %s: %w", pm.Name, err)
		}

		server := mcpServerModelsCache[serverName]

		if server.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddPrompt(prompt, m.mcpProxyPromptHandler)
		} else {
			m.mcpProxyServer.AddPrompt(prompt, m.mcpProxyPromptHandler)
		}
	}

	// Load resources
	resources, err := m.ListResources()
	if err != nil {
		return fmt.Errorf("failed to list resources from DB: %w", err)
	}

	for _, rm := range resources {
		if !rm.Enabled {
			continue
		}

		if !isActiveServer(rm.Server.Name) {
			continue
		}

		resource, err := convertResourceModelToMcpObject(&rm)
		if err != nil {
			return fmt.Errorf("failed to convert resource model to MCP object for resource %s: %w", rm.URI, err)
		}
		resource.Name = rm.Name

		if rm.Server.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddResource(resource, m.mcpProxyResourceHandler)
		} else {
			m.mcpProxyServer.AddResource(resource, m.mcpProxyResourceHandler)
		}
	}

	return nil
}
