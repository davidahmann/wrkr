package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
)

func TestStatusJSONIncludesLease(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }, LeaseTTL: 30 * time.Second})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}

	if _, err := r.InitJob("job_status"); err != nil {
		t.Fatalf("init job: %v", err)
	}
	if _, err := r.ChangeStatus("job_status", queue.StatusRunning); err != nil {
		t.Fatalf("change status: %v", err)
	}
	if _, err := r.AcquireLease("job_status", "worker-a", "lease-a"); err != nil {
		t.Fatalf("acquire lease: %v", err)
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"status", "job_status", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"schema_id\": \"wrkr.status_response\"") {
		t.Fatalf("expected status response json, got %q", out.String())
	}
	if !strings.Contains(out.String(), "\"worker_id\": \"worker-a\"") {
		t.Fatalf("expected lease worker in output, got %q", out.String())
	}
}

func TestStatusMissingJobReturnsExit6(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"status", "missing", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 6 {
		t.Fatalf("expected exit code 6, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "E_INVALID_INPUT_SCHEMA") {
		t.Fatalf("expected invalid input envelope, got %q", errBuf.String())
	}
}
