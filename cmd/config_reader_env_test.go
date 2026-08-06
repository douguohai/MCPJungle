package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfigFilesResolveEnvironmentVariables(t *testing.T) {
	t.Setenv("MCPJ_TEST_SERVER_ID", "workspace-123")
	t.Setenv("MCPJ_TEST_SERVER_TOKEN", "server-token")
	t.Setenv("MCPJ_TEST_ALLOW_SERVER", "affine-main")
	t.Setenv("MCPJ_TEST_USER_NAME", "alice")
	t.Setenv("MCPJ_TEST_GROUP_NAME", "shared-tools")

	tempDir := t.TempDir()

	serverPath := filepath.Join(tempDir, "server.json")
	if err := os.WriteFile(serverPath, []byte(`{
		"name": "affine-main",
		"transport": "streamable_http",
		"url": "https://app.affine.pro/api/workspaces/${MCPJ_TEST_SERVER_ID}/mcp",
		"bearer_token": "${MCPJ_TEST_SERVER_TOKEN}",
		"headers": {
			"Authorization": "Bearer ${MCPJ_TEST_SERVER_TOKEN}"
		}
	}`), 0o600); err != nil {
		t.Fatalf("failed to write server config: %v", err)
	}

	groupPath := filepath.Join(tempDir, "group.json")
	if err := os.WriteFile(groupPath, []byte(`{
		"name": "${MCPJ_TEST_GROUP_NAME}",
		"included_servers": ["${MCPJ_TEST_ALLOW_SERVER}"],
		"description": "tools-for-${MCPJ_TEST_USER_NAME}"
	}`), 0o600); err != nil {
		t.Fatalf("failed to write group config: %v", err)
	}

	serverCfg, err := readMcpServerConfig(serverPath)
	if err != nil {
		t.Fatalf("unexpected error reading server config: %v", err)
	}
	if serverCfg.URL != "https://app.affine.pro/api/workspaces/workspace-123/mcp" {
		t.Fatalf("expected resolved server URL, got %q", serverCfg.URL)
	}
	if serverCfg.BearerToken != "server-token" {
		t.Fatalf("expected resolved bearer token, got %q", serverCfg.BearerToken)
	}
	if serverCfg.Headers["Authorization"] != "Bearer server-token" {
		t.Fatalf("expected resolved server header, got %q", serverCfg.Headers["Authorization"])
	}

	groupCfg, err := readToolGroupConfig(groupPath)
	if err != nil {
		t.Fatalf("unexpected error reading group config: %v", err)
	}
	if groupCfg.Name != "shared-tools" {
		t.Fatalf("expected resolved group name, got %q", groupCfg.Name)
	}
	if groupCfg.IncludedServers[0] != "affine-main" {
		t.Fatalf("expected resolved included server, got %q", groupCfg.IncludedServers[0])
	}
	if groupCfg.Description != "tools-for-alice" {
		t.Fatalf("expected resolved group description, got %q", groupCfg.Description)
	}
}
