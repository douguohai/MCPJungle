package mcp

import (
	"context"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

type accessContextKey struct{}

// AccessContext is the authorization result attached to one MCP request after
// a personal device token has been authenticated.
type AccessContext struct {
	UserID               uint
	DeviceTokenID        uint
	EffectiveServiceIDs  map[uint]struct{}
	EffectiveServerNames map[string]struct{}
}

func WithAccessContext(ctx context.Context, access AccessContext) context.Context {
	return context.WithValue(ctx, accessContextKey{}, access)
}

func AccessFromContext(ctx context.Context) (AccessContext, bool) {
	access, ok := ctx.Value(accessContextKey{}).(AccessContext)
	return access, ok
}

// ProxyToolFilter applies the same fail-closed service set used by direct
// calls. Development mode remains unrestricted.
func ProxyToolFilter(ctx context.Context, tools []mcpsdk.Tool) []mcpsdk.Tool {
	serverMode, ok := ctx.Value("mode").(model.ServerMode)
	if !ok {
		return nil
	}
	if !model.IsEnterpriseMode(serverMode) {
		return tools
	}
	access, ok := AccessFromContext(ctx)
	if !ok || access.UserID == 0 || access.DeviceTokenID == 0 {
		return nil
	}

	filtered := make([]mcpsdk.Tool, 0, len(tools))
	for _, tool := range tools {
		serverName, _, valid := splitServerToolName(tool.Name)
		if !valid {
			continue
		}
		if _, allowed := access.EffectiveServerNames[serverName]; allowed {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// ProxyPromptFilter applies the same fail-closed service set to prompts/list.
func ProxyPromptFilter(ctx context.Context, prompts []mcpsdk.Prompt) []mcpsdk.Prompt {
	serverMode, ok := ctx.Value("mode").(model.ServerMode)
	if !ok {
		return nil
	}
	if !model.IsEnterpriseMode(serverMode) {
		return prompts
	}
	access, ok := AccessFromContext(ctx)
	if !ok || access.UserID == 0 || access.DeviceTokenID == 0 {
		return nil
	}

	filtered := make([]mcpsdk.Prompt, 0, len(prompts))
	for _, prompt := range prompts {
		serverName, _, valid := splitServerPromptName(prompt.Name)
		if !valid {
			continue
		}
		if _, allowed := access.EffectiveServerNames[serverName]; allowed {
			filtered = append(filtered, prompt)
		}
	}
	return filtered
}
