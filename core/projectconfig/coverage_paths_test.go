package projectconfig

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func TestInitJobSpecCoveragePaths(t *testing.T) {
	wd := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	now := time.Date(2026, 2, 14, 12, 30, 0, 0, time.UTC)

	written, err := InitJobSpec("", false, now, "")
	if err != nil {
		t.Fatalf("InitJobSpec default path: %v", err)
	}
	if filepath.Base(written) != DefaultJobSpecPath {
		t.Fatalf("expected default path %q, got %q", DefaultJobSpecPath, written)
	}

	_, err = InitJobSpec(DefaultJobSpecPath, false, now, "test")
	if err == nil {
		t.Fatal("expected existing jobspec failure without --force")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EInvalidInputSchema {
		t.Fatalf("expected E_INVALID_INPUT_SCHEMA, got %v", err)
	}

	if _, err := InitJobSpec(DefaultJobSpecPath, true, now, "test"); err != nil {
		t.Fatalf("InitJobSpec force overwrite: %v", err)
	}
}

func TestLoadJobSpecCoveragePaths(t *testing.T) {
	wd := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	now := time.Date(2026, 2, 14, 12, 40, 0, 0, time.UTC)
	spec := defaultJobSpec(now, "test")

	jsonPath := filepath.Join(wd, "jobspec.json")
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal jobspec json: %v", err)
	}
	if err := os.WriteFile(jsonPath, raw, 0o600); err != nil {
		t.Fatalf("write jobspec json: %v", err)
	}

	loaded, err := LoadJobSpec(jsonPath)
	if err != nil {
		t.Fatalf("LoadJobSpec json: %v", err)
	}
	if loaded.Adapter.Name != "reference" {
		t.Fatalf("unexpected adapter: %s", loaded.Adapter.Name)
	}

	if _, err := LoadJobSpec(""); err == nil {
		t.Fatal("expected empty path failure")
	}

	if err := os.WriteFile("bad.yaml", []byte(":\n"), 0o600); err != nil {
		t.Fatalf("write bad yaml: %v", err)
	}
	if _, err := LoadJobSpec("bad.yaml"); err == nil {
		t.Fatal("expected invalid yaml parse failure")
	}

	if err := os.WriteFile("invalid_schema.yaml", []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write invalid schema yaml: %v", err)
	}
	if _, err := LoadJobSpec("invalid_schema.yaml"); err == nil {
		t.Fatal("expected schema validation failure")
	}
}

func TestNormalizeYAMLAndResolvePathCoverage(t *testing.T) {
	value := map[any]any{
		"root": []any{
			map[any]any{
				"nested": "value",
			},
		},
	}
	normalized := normalizeYAMLValue(value)
	asMap, ok := normalized.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", normalized)
	}
	if _, ok := asMap["root"]; !ok {
		t.Fatalf("expected normalized root key, got %+v", asMap)
	}

	wd := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	resolved, err := resolveJobSpecPath("jobs/spec.yaml")
	if err != nil {
		t.Fatalf("resolveJobSpecPath: %v", err)
	}
	if !strings.HasSuffix(resolved, filepath.Join("jobs", "spec.yaml")) {
		t.Fatalf("unexpected resolved path: %s", resolved)
	}
}
