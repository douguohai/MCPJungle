package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

// ProxyToolFilter filters tools exposed by MCP proxy for enterprise mode based on client allow-list.
func ProxyToolFilter(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	serverMode, ok := ctx.Value("mode").(model.ServerMode)
	if !ok {
		// Missing/invalid mode in context: fail closed.
		return nil
	}
	if !model.IsEnterpriseMode(serverMode) {
		// In non-enterprise mode, there are no access restrictions, so return all tools
		return tools
	}

	dt, ok := ctx.Value("device_token").(*model.DeviceToken)
	if !ok || dt == nil {
		// Enterprise mode requires an authenticated device token; fail closed if absent.
		return nil
	}

	effective, _ := ctx.Value("effective_services").(map[string]bool)
	if effective == nil {
		return nil
	}

	var filteredTools []mcp.Tool
	for _, tool := range tools {
		serverName, _, _ := splitServerToolName(tool.Name)
		if effective[serverName] {
			filteredTools = append(filteredTools, tool)
		}
	}
	return filteredTools
}
