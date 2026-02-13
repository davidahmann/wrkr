package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
)

func TestCheckpointListAndShowJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 16, 0, 0, 0, time.UTC)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_cli_cp"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_cli_cp", queue.StatusRunning); err != nil {
		t.Fatalf("status: %v", err)
	}
	cp, err := r.EmitCheckpoint("job_cli_cp", runner.CheckpointInput{Type: "progress", Summary: "implemented checkpoint list/show"})
	if err != nil {
		t.Fatalf("emit checkpoint: %v", err)
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"checkpoint", "list", "job_cli_cp", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), cp.CheckpointID) {
		t.Fatalf("expected checkpoint id %s in output: %s", cp.CheckpointID, out.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"checkpoint", "show", "job_cli_cp", cp.CheckpointID, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"schema_id\": \"wrkr.checkpoint\"") {
		t.Fatalf("expected checkpoint schema output, got %s", out.String())
	}
}

func TestApproveAndResumeFlow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 16, 0, 0, 0, time.UTC)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_cli_approve"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_cli_approve", queue.StatusRunning); err != nil {
		t.Fatalf("running: %v", err)
	}
	cp, err := r.EmitCheckpoint("job_cli_approve", runner.CheckpointInput{
		Type:    "decision-needed",
		Summary: "approval required",
		Status:  queue.StatusBlockedDecision,
		RequiredAction: &v1.RequiredAction{
			Kind:         "approval",
			Instructions: "confirm step",
		},
	})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if _, err := r.ChangeStatus("job_cli_approve", queue.StatusBlockedDecision); err != nil {
		t.Fatalf("blocked decision: %v", err)
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"resume", "job_cli_approve", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 4 {
		t.Fatalf("expected exit 4 (approval required), got %d", code)
	}
	if !strings.Contains(errBuf.String(), "E_CHECKPOINT_APPROVAL_REQUIRED") {
		t.Fatalf("expected approval-required envelope, got %s", errBuf.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"approve", "job_cli_approve", "--checkpoint", cp.CheckpointID, "--reason", "approved", "--approved-by", "lead", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("expected exit 0 from approve, got %d: %s", code, errBuf.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"resume", "job_cli_approve", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("expected resume success, got %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"status\": \"running\"") {
		t.Fatalf("expected running status, got %s", out.String())
	}
}

func TestCheckpointEmitAndBudgetCheck(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 16, 0, 0, 0, time.UTC)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_cli_budget"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_cli_budget", queue.StatusRunning); err != nil {
		t.Fatalf("running: %v", err)
	}
	if _, err := r.UpdateCounters("job_cli_budget", 0, 10, 1); err != nil {
		t.Fatalf("counters: %v", err)
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"checkpoint", "emit", "job_cli_budget", "--type", "plan", "--summary", "starting run", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("checkpoint emit failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"type\": \"plan\"") {
		t.Fatalf("expected plan checkpoint json, got %s", out.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"budget", "check", "job_cli_budget", "--max-step-count", "1", "--json"}, &out, &errBuf, func() time.Time { return now.Add(2 * time.Minute) })
	if code != 1 {
		t.Fatalf("expected budget exceeded exit 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "E_BUDGET_EXCEEDED") {
		t.Fatalf("expected E_BUDGET_EXCEEDED, got %s", errBuf.String())
	}
}
