package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/pack"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
)

func setupEpic4Job(t *testing.T, jobID string, now time.Time) {
	t.Helper()
	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob(jobID); err != nil {
		t.Fatalf("init job: %v", err)
	}
	if _, err := r.ChangeStatus(jobID, queue.StatusRunning); err != nil {
		t.Fatalf("status running: %v", err)
	}
	if _, err := r.EmitCheckpoint(jobID, runner.CheckpointInput{Type: "progress", Summary: "checkpoint"}); err != nil {
		t.Fatalf("emit checkpoint: %v", err)
	}
}

func TestExportVerifyAndReceipt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 19, 0, 0, 0, time.UTC)
	setupEpic4Job(t, "job_cli_export", now)
	outDir := t.TempDir()

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"export", "job_cli_export", "--out-dir", outDir, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("export failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"job_id\": \"job_cli_export\"") {
		t.Fatalf("unexpected export output: %s", out.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"verify", "job_cli_export", "--out-dir", outDir, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("verify failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"job_id\": \"job_cli_export\"") {
		t.Fatalf("unexpected verify output: %s", out.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"receipt", "job_cli_export", "--out-dir", outDir}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("receipt failed: %d %s", code, errBuf.String())
	}
	footer := strings.TrimSpace(out.String())
	if _, err := pack.ParseTicketFooter(footer); err != nil {
		t.Fatalf("invalid footer format: %v (%s)", err, footer)
	}
}

func TestJobInspectAndDiff(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 19, 0, 0, 0, time.UTC)
	setupEpic4Job(t, "job_cli_diff_a", now)
	setupEpic4Job(t, "job_cli_diff_b", now)
	outDir := t.TempDir()

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"checkpoint", "emit", "job_cli_diff_b", "--type", "progress", "--summary", "extra"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("emit checkpoint failed: %d %s", code, errBuf.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"export", "job_cli_diff_a", "--out-dir", outDir, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("export A failed: %d %s", code, errBuf.String())
	}
	pathA := extractJSONField(t, out.String(), "path")

	out.Reset()
	errBuf.Reset()
	code = run([]string{"export", "job_cli_diff_b", "--out-dir", outDir, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("export B failed: %d %s", code, errBuf.String())
	}
	pathB := extractJSONField(t, out.String(), "path")

	out.Reset()
	errBuf.Reset()
	code = run([]string{"job", "inspect", "job_cli_diff_a", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("inspect failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"job_id\": \"job_cli_diff_a\"") {
		t.Fatalf("unexpected inspect output: %s", out.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run([]string{"job", "diff", pathA, pathB, "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("diff failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"changed\"") {
		t.Fatalf("unexpected diff output: %s", out.String())
	}
}

func extractJSONField(t *testing.T, output, key string) string {
	t.Helper()

	needle := `"` + key + `": "`
	start := strings.Index(output, needle)
	if start == -1 {
		t.Fatalf("field %s missing in output: %s", key, output)
	}
	start += len(needle)
	end := strings.Index(output[start:], `"`)
	if end == -1 {
		t.Fatalf("field %s not terminated in output: %s", key, output)
	}
	return output[start : start+end]
}
