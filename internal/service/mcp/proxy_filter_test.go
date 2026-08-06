package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestMcpProxyToolFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		mode              model.ServerMode
		deviceToken       *model.DeviceToken
		effectiveServices map[string]bool
		tools             []mcp.Tool
		wantNames         []string
	}{
		{
			name: "development mode returns all tools",
			mode: model.ModeDev,
			tools: []mcp.Tool{
				{Name: "time__get_current_time"},
				{Name: "deepwiki__search_wiki"},
			},
			wantNames: []string{"time__get_current_time", "deepwiki__search_wiki"},
		},
		{
			name: "enterprise mode filters unauthorized tools",
			mode: model.ModeEnterprise,
			deviceToken: &model.DeviceToken{
				Name: "test-token",
			},
			effectiveServices: map[string]bool{"time": true},
			tools: []mcp.Tool{
				{Name: "time__get_current_time"},
				{Name: "deepwiki__search_wiki"},
			},
			wantNames: []string{"time__get_current_time"},
		},
		{
			name: "enterprise mode with all servers allowed",
			mode: model.ModeEnterprise,
			deviceToken: &model.DeviceToken{
				Name: "test-token",
			},
			effectiveServices: map[string]bool{"time": true, "deepwiki": true},
			tools: []mcp.Tool{
				{Name: "time__get_current_time"},
				{Name: "deepwiki__search_wiki"},
			},
			wantNames: []string{"time__get_current_time", "deepwiki__search_wiki"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(context.Background(), "mode", tt.mode)
			if tt.deviceToken != nil {
				ctx = context.WithValue(ctx, "device_token", tt.deviceToken)
			}
			if tt.effectiveServices != nil {
				ctx = context.WithValue(ctx, "effective_services", tt.effectiveServices)
			}

			got := ProxyToolFilter(ctx, tt.tools)
			assert.Equal(t, tt.wantNames, toolNames(got))
		})
	}
}

func TestMcpProxyToolFilter_MissingModeInContext(t *testing.T) {
	t.Parallel()

	got := ProxyToolFilter(context.Background(), []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_InvalidModeTypeInContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", "enterprise")
	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_EnterpriseMissingDeviceTokenInContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_EnterpriseInvalidDeviceTokenTypeInContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = context.WithValue(ctx, "device_token", "not-a-device-token")
	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_EnterpriseNilDeviceTokenInContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	var dt *model.DeviceToken
	ctx = context.WithValue(ctx, "device_token", dt)
	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_EnterpriseMissingEffectiveServicesInContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = context.WithValue(ctx, "device_token", &model.DeviceToken{Name: "test"})
	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_EnterpriseMalformedToolNamesAreDenied(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = context.WithValue(ctx, "device_token", &model.DeviceToken{Name: "test"})
	ctx = context.WithValue(ctx, "effective_services", map[string]bool{"time": true})

	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "missing_separator"},
		{Name: "time__get_current_time"},
	})

	assert.Equal(t, []string{"time__get_current_time"}, toolNames(got))
}

func toolNames(tools []mcp.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	return names
}
