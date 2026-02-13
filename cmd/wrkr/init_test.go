package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunInitWritesDefaultJobSpecJSON(t *testing.T) {
	workspace := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})

	var out bytes.Buffer
	var errBuf bytes.Buffer
	now := func() time.Time { return time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC) }
	code := runInit([]string{}, true, &out, &errBuf, now)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"jobspec_path\"") {
		t.Fatalf("expected jobspec path in JSON output, got %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(workspace, "jobspec.yaml")); err != nil {
		t.Fatalf("jobspec file missing: %v", err)
	}
}

func TestRunInitRequiresPathValue(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errBuf bytes.Buffer
	now := func() time.Time { return time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC) }
	code := runInit([]string{"--path"}, true, &out, &errBuf, now)
	if code != 6 {
		t.Fatalf("expected exit 6, got %d stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "E_INVALID_INPUT_SCHEMA") {
		t.Fatalf("expected invalid input schema, got %q", errBuf.String())
	}
}

func TestRunInitRejectsUnknownFlag(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errBuf bytes.Buffer
	now := func() time.Time { return time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC) }
	code := runInit([]string{"--nope"}, true, &out, &errBuf, now)
	if code != 6 {
		t.Fatalf("expected exit 6, got %d stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "unknown init flag") {
		t.Fatalf("expected unknown init flag error, got %q", errBuf.String())
	}
}
