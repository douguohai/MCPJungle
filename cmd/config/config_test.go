package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSaveAndLoadRegistryURLWithoutCredentials(t *testing.T) {
	setTestHome(t)

	original := &ClientConfig{RegistryURL: "https://hub.internal.example"}
	if err := Save(original); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded := Load()
	if loaded.RegistryURL != original.RegistryURL {
		t.Fatalf("registry URL mismatch: got %q want %q", loaded.RegistryURL, original.RegistryURL)
	}

	path, err := AbsPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(strings.ToLower(string(payload)), "token") {
		t.Fatalf("CLI config unexpectedly contains a credential field: %s", payload)
	}
}

func TestLoadIgnoresRemovedAccessTokenField(t *testing.T) {
	setTestHome(t)
	path, err := AbsPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}
	if err := os.WriteFile(path, []byte("registry_url: https://hub.internal.example\naccess_token: legacy-secret\n"), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	loaded := Load()
	if loaded.RegistryURL != "https://hub.internal.example" {
		t.Fatalf("load registry URL: got %q", loaded.RegistryURL)
	}
	if err := Save(loaded); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten config: %v", err)
	}
	if strings.Contains(string(payload), "legacy-secret") || strings.Contains(string(payload), "access_token") {
		t.Fatalf("legacy credential survived config rewrite: %s", payload)
	}
}

func TestLoadMissingOrInvalidConfigReturnsEmpty(t *testing.T) {
	setTestHome(t)
	if cfg := Load(); cfg.RegistryURL != "" {
		t.Fatalf("missing config returned registry URL %q", cfg.RegistryURL)
	}

	path, err := AbsPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}
	if err := os.WriteFile(path, []byte("invalid: yaml: ["), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	if cfg := Load(); cfg.RegistryURL != "" {
		t.Fatalf("invalid config returned registry URL %q", cfg.RegistryURL)
	}
}

func setTestHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
		t.Setenv("HOMEDRIVE", filepath.VolumeName(dir))
		t.Setenv("HOMEPATH", strings.TrimPrefix(dir, filepath.VolumeName(dir)))
	}
	return dir
}
