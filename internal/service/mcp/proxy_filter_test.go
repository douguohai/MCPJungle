package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

func TestProxyToolFilterUsesDeviceTokenServiceSet(t *testing.T) {
	tools := []mcp.Tool{{Name: "allowed__one"}, {Name: "denied__two"}, {Name: "malformed"}}
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = WithAccessContext(ctx, AccessContext{
		UserID: 1, DeviceTokenID: 2,
		EffectiveServerNames: map[string]struct{}{"allowed": {}},
	})
	got := ProxyToolFilter(ctx, tools)
	if len(got) != 1 || got[0].Name != "allowed__one" {
		t.Fatalf("unexpected filtered tools: %+v", got)
	}
}

func TestProxyToolFilterFailsClosedWithoutAccessContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	if got := ProxyToolFilter(ctx, []mcp.Tool{{Name: "server__tool"}}); len(got) != 0 {
		t.Fatalf("enterprise request without device context must be denied: %+v", got)
	}
}

func TestProxyToolFilterAllowsDevelopmentMode(t *testing.T) {
	tools := []mcp.Tool{{Name: "server__tool"}}
	ctx := context.WithValue(context.Background(), "mode", model.ModeDev)
	if got := ProxyToolFilter(ctx, tools); len(got) != 1 {
		t.Fatalf("development mode should remain unrestricted: %+v", got)
	}
}

func TestProxyPromptFilterUsesDeviceTokenServiceSet(t *testing.T) {
	prompts := []mcp.Prompt{{Name: "allowed__review"}, {Name: "denied__review"}, {Name: "malformed"}}
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = WithAccessContext(ctx, AccessContext{
		UserID: 7, DeviceTokenID: 9,
		EffectiveServerNames: map[string]struct{}{"allowed": {}},
	})
	got := ProxyPromptFilter(ctx, prompts)
	if len(got) != 1 || got[0].Name != "allowed__review" {
		t.Fatalf("unexpected filtered prompts: %+v", got)
	}
}

func TestProxyPromptFilterFailsClosedWithoutAccessContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	if got := ProxyPromptFilter(ctx, []mcp.Prompt{{Name: "server__prompt"}}); len(got) != 0 {
		t.Fatalf("enterprise request without device context must be denied: %+v", got)
	}
}
