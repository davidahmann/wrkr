package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRunVersionText(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var err bytes.Buffer
	code := run(nil, &out, &err, time.Now)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(out.String(), "wrkr ") {
		t.Fatalf("expected version output, got %q", out.String())
	}
}

func TestRunUnknownCommandJSON(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var err bytes.Buffer
	fixed := func() time.Time { return time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC) }
	code := run([]string{"--json", "unknown"}, &out, &err, fixed)
	if code != 6 {
		t.Fatalf("expected exit code 6, got %d", code)
	}
	if !strings.Contains(err.String(), "\"schema_id\": \"wrkr.error_envelope\"") {
		t.Fatalf("expected error envelope json, got %q", err.String())
	}
	if !strings.Contains(err.String(), "E_INVALID_INPUT_SCHEMA") {
		t.Fatalf("expected error code in output, got %q", err.String())
	}
}
