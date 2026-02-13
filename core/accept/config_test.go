package accept

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitAndLoadConfigAbsolutePath(t *testing.T) {
	workspace := t.TempDir()
	configPath := filepath.Join(workspace, "accept.yaml")

	written, err := InitConfig(configPath, false)
	if err != nil {
		t.Fatalf("InitConfig: %v", err)
	}
	if written != configPath {
		t.Fatalf("expected path %q, got %q", configPath, written)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file missing: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.SchemaID != "wrkr.accept_config" {
		t.Fatalf("unexpected schema id: %q", cfg.SchemaID)
	}
	if cfg.SchemaVersion != "v1" {
		t.Fatalf("unexpected schema version: %q", cfg.SchemaVersion)
	}
}

func TestLoadConfigOrDefaultMissingPath(t *testing.T) {
	workspace := t.TempDir()
	configPath := filepath.Join(workspace, "missing.yaml")

	cfg, resolved, err := LoadConfigOrDefault(configPath)
	if err != nil {
		t.Fatalf("LoadConfigOrDefault: %v", err)
	}
	if resolved != configPath {
		t.Fatalf("expected resolved path %q, got %q", configPath, resolved)
	}
	if cfg.SchemaID != "wrkr.accept_config" {
		t.Fatalf("expected default schema id, got %q", cfg.SchemaID)
	}
}
