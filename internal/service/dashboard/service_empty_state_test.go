package dashboard

import (
	"strings"
	"testing"
)

func TestNoServersEmptyStateUsesBrowserFirstChineseCopy(t *testing.T) {
	state := noServersEmptyState()

	if state.Title != "还没有 MCP 服务" {
		t.Fatalf("unexpected title: %q", state.Title)
	}
	if !strings.Contains(state.Description, "添加") {
		t.Fatalf("description should guide the browser action: %q", state.Description)
	}
}

func TestTroubleshootingHintsUseProductLanguage(t *testing.T) {
	hints := collectTroubleshootingHints(nil, 0, 0, 0)
	joined := strings.Join(hints, " ")

	if strings.Contains(joined, "CLI") || strings.Contains(joined, "No servers") {
		t.Fatalf("troubleshooting hints should be Chinese and browser-first: %q", joined)
	}
	if !strings.Contains(joined, "MCP 服务") {
		t.Fatalf("troubleshooting hints should use MCP service terminology: %q", joined)
	}
}
