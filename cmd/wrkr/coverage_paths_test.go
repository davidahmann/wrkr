package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
)

func setupCLIWorkspace(t *testing.T) (string, time.Time) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	workspace := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("chdir workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})

	now := time.Date(2026, 2, 14, 7, 0, 0, 0, time.UTC)
	return workspace, now
}

func setupCLIJob(t *testing.T, now time.Time, jobID string, status queue.Status) *runner.Runner {
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
		t.Fatalf("InitJob: %v", err)
	}
	if status != queue.StatusQueued {
		if _, err := r.ChangeStatus(jobID, status); err != nil {
			t.Fatalf("ChangeStatus: %v", err)
		}
	}
	return r
}

func TestRunBudgetCoveragePaths(t *testing.T) {
	_, now := setupCLIWorkspace(t)
	r := setupCLIJob(t, now, "job_budget_paths", queue.StatusRunning)
	if _, err := r.UpdateCounters("job_budget_paths", 0, 10, 0); err != nil {
		t.Fatalf("UpdateCounters: %v", err)
	}

	cases := []struct {
		args     []string
		exitCode int
		needle   string
	}{
		{args: []string{}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"check"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"check", "job_budget_paths", "--max-step-count"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"check", "job_budget_paths", "--max-step-count", "nope"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"check", "job_budget_paths", "--unknown"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"check", "job_budget_paths", "--max-step-count", "100"}, exitCode: 0, needle: "budget=within_limits"},
		{args: []string{"check", "job_budget_paths", "--max-step-count", "1"}, exitCode: 1, needle: "E_BUDGET_EXCEEDED"},
	}

	for _, tc := range cases {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runBudget(tc.args, false, &out, &errBuf, func() time.Time { return now.Add(2 * time.Minute) })
		if code != tc.exitCode {
			t.Fatalf("args=%v expected exit %d got %d stderr=%s", tc.args, tc.exitCode, code, errBuf.String())
		}
		combined := out.String() + errBuf.String()
		if !strings.Contains(combined, tc.needle) {
			t.Fatalf("args=%v expected output to contain %q, got %q", tc.args, tc.needle, combined)
		}
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := runBudget([]string{"check", "job_budget_paths", "--max-step-count", "100"}, true, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("json budget check failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"result\": \"within_budget\"") {
		t.Fatalf("unexpected json budget output: %s", out.String())
	}
}

func TestRunStoreCoveragePaths(t *testing.T) {
	workspace, now := setupCLIWorkspace(t)

	storeRoot := filepath.Join(workspace, ".wrkr")
	outRoot := filepath.Join(workspace, "wrkr-out")
	if err := os.MkdirAll(filepath.Join(storeRoot, "jobs", "job_1"), 0o750); err != nil {
		t.Fatalf("mkdir store fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(outRoot, "jobpacks"), 0o750); err != nil {
		t.Fatalf("mkdir out fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storeRoot, "jobs", "job_1", "events.jsonl"), []byte(""), 0o600); err != nil {
		t.Fatalf("write events fixture: %v", err)
	}

	cases := []struct {
		args     []string
		exitCode int
		needle   string
	}{
		{args: []string{}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"nope"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"prune"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"prune", "--job-max-age", "bad"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"prune", "--max-jobpacks", "-1"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{
			args:     []string{"prune", "--dry-run", "--store-root", storeRoot, "--out-dir", outRoot, "--job-max-age", "1h"},
			exitCode: 0,
			needle:   "store prune dry_run=true",
		},
	}

	for _, tc := range cases {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runStore(tc.args, false, &out, &errBuf, func() time.Time { return now })
		if code != tc.exitCode {
			t.Fatalf("args=%v expected exit %d got %d stderr=%s", tc.args, tc.exitCode, code, errBuf.String())
		}
		combined := out.String() + errBuf.String()
		if !strings.Contains(combined, tc.needle) {
			t.Fatalf("args=%v expected output to contain %q, got %q", tc.args, tc.needle, combined)
		}
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := runStore(
		[]string{"prune", "--dry-run", "--store-root", storeRoot, "--out-dir", outRoot, "--job-max-age", "1h", "--max-jobpacks", "1"},
		true,
		&out,
		&errBuf,
		func() time.Time { return now },
	)
	if code != 0 {
		t.Fatalf("json store prune failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"dry_run\": true") {
		t.Fatalf("unexpected json store output: %s", out.String())
	}
}

func TestRunBridgeCoveragePaths(t *testing.T) {
	workspace, now := setupCLIWorkspace(t)
	_ = workspace

	r := setupCLIJob(t, now, "job_bridge_paths", queue.StatusRunning)
	progressCP, err := r.EmitCheckpoint("job_bridge_paths", runner.CheckpointInput{
		Type:    "progress",
		Summary: "ordinary progress",
	})
	if err != nil {
		t.Fatalf("emit progress checkpoint: %v", err)
	}
	blockedCP, err := r.EmitCheckpoint("job_bridge_paths", runner.CheckpointInput{
		Type:    "blocked",
		Summary: "waiting for missing permission",
		ReasonCodes: []string{
			"E_ADAPTER_FAIL",
		},
	})
	if err != nil {
		t.Fatalf("emit blocked checkpoint: %v", err)
	}

	cases := []struct {
		args     []string
		exitCode int
		needle   string
	}{
		{args: []string{}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"nope"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"work-item"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"work-item", "job_bridge_paths"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"work-item", "job_bridge_paths", "--checkpoint"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"work-item", "job_bridge_paths", "--checkpoint", progressCP.CheckpointID}, exitCode: 6, needle: "checkpoint type must be blocked or decision-needed"},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runBridge(tc.args, false, &out, &errBuf, func() time.Time { return now })
		if code != tc.exitCode {
			t.Fatalf("args=%v expected exit %d got %d stderr=%s", tc.args, tc.exitCode, code, errBuf.String())
		}
		combined := out.String() + errBuf.String()
		if !strings.Contains(combined, tc.needle) {
			t.Fatalf("args=%v expected output to contain %q, got %q", tc.args, tc.needle, combined)
		}
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := runBridge(
		[]string{"work-item", "job_bridge_paths", "--checkpoint", blockedCP.CheckpointID, "--dry-run"},
		false,
		&out,
		&errBuf,
		func() time.Time { return now },
	)
	if code != 0 {
		t.Fatalf("bridge dry-run failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "next_commands:") {
		t.Fatalf("expected next_commands in bridge output: %s", out.String())
	}

	out.Reset()
	errBuf.Reset()
	code = runBridge(
		[]string{"work-item", "job_bridge_paths", "--checkpoint", blockedCP.CheckpointID, "--template", "github", "--out-dir", "wrkr-out"},
		true,
		&out,
		&errBuf,
		func() time.Time { return now },
	)
	if code != 0 {
		t.Fatalf("bridge write failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "\"json_path\"") {
		t.Fatalf("expected json_path in bridge payload: %s", out.String())
	}
}

func TestRunCheckpointCoveragePaths(t *testing.T) {
	_, now := setupCLIWorkspace(t)

	r := setupCLIJob(t, now, "job_checkpoint_paths", queue.StatusRunning)
	baseSummary := strings.Repeat("summary ", 40)
	emitted, err := r.EmitCheckpoint("job_checkpoint_paths", runner.CheckpointInput{
		Type:    "blocked",
		Summary: baseSummary,
		Status:  queue.StatusBlockedError,
		ReasonCodes: []string{
			"E_ADAPTER_FAIL",
		},
		RequiredAction: &v1.RequiredAction{
			Kind:         "approval",
			Instructions: "review logs",
		},
	})
	if err != nil {
		t.Fatalf("emit checkpoint fixture: %v", err)
	}

	cases := []struct {
		args     []string
		exitCode int
		needle   string
	}{
		{args: []string{}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"nope"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"list"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"show", "job_checkpoint_paths"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"emit"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"emit", "job_checkpoint_paths", "--type"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"emit", "job_checkpoint_paths", "--summary"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"emit", "job_checkpoint_paths", "--unknown"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runCheckpoint(tc.args, false, &out, &errBuf, func() time.Time { return now })
		if code != tc.exitCode {
			t.Fatalf("args=%v expected exit %d got %d stderr=%s", tc.args, tc.exitCode, code, errBuf.String())
		}
		combined := out.String() + errBuf.String()
		if !strings.Contains(combined, tc.needle) {
			t.Fatalf("args=%v expected output to contain %q, got %q", tc.args, tc.needle, combined)
		}
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := runCheckpoint([]string{"list", "job_checkpoint_paths"}, false, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("checkpoint list failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "...") {
		t.Fatalf("expected bounded summary truncation, got %s", out.String())
	}

	out.Reset()
	errBuf.Reset()
	code = runCheckpoint([]string{"show", "job_checkpoint_paths", emitted.CheckpointID}, false, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("checkpoint show failed: %d %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "reason_codes=E_ADAPTER_FAIL") {
		t.Fatalf("expected reason codes in show output, got %s", out.String())
	}
}

func TestRunServeCoveragePaths(t *testing.T) {
	_, now := setupCLIWorkspace(t)

	cases := []struct {
		args     []string
		exitCode int
		needle   string
	}{
		{args: []string{"--listen"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"--auth-token"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"--max-body-bytes"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"--max-body-bytes", "bad"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"--unknown"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{args: []string{"--listen", "0.0.0.0:9488"}, exitCode: 8, needle: "E_UNSAFE_OPERATION"},
		{args: []string{"--listen", "0.0.0.0:9488", "--allow-non-loopback"}, exitCode: 8, needle: "E_UNSAFE_OPERATION"},
		{args: []string{"--listen", "0.0.0.0:9488", "--allow-non-loopback", "--auth-token", "token"}, exitCode: 8, needle: "E_UNSAFE_OPERATION"},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runServe(tc.args, true, &out, &errBuf, func() time.Time { return now })
		if code != tc.exitCode {
			t.Fatalf("args=%v expected exit %d got %d stderr=%s", tc.args, tc.exitCode, code, errBuf.String())
		}
		combined := out.String() + errBuf.String()
		if !strings.Contains(combined, tc.needle) {
			t.Fatalf("args=%v expected output to contain %q, got %q", tc.args, tc.needle, combined)
		}
	}
}

func TestRunJobAndReportCoveragePaths(t *testing.T) {
	_, now := setupCLIWorkspace(t)

	type commandFn func([]string, bool, io.Writer, io.Writer, func() time.Time) int

	cases := []struct {
		runFn    commandFn
		args     []string
		exitCode int
		needle   string
	}{
		{runFn: runJob, args: []string{}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runJob, args: []string{"nope"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runJob, args: []string{"inspect"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runJob, args: []string{"diff", "a"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runReport, args: []string{}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runReport, args: []string{"nope"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runReport, args: []string{"github"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runExport, args: []string{}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runExport, args: []string{"job1", "--out-dir"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runReceipt, args: []string{}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runReceipt, args: []string{"job1", "--out-dir"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runResume, args: []string{}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runResume, args: []string{"job1", "--reason"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
		{runFn: runResume, args: []string{"job1", "--approved-by"}, exitCode: 6, needle: "E_INVALID_INPUT_SCHEMA"},
	}

	for _, tc := range cases {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := tc.runFn(tc.args, true, &out, &errBuf, func() time.Time { return now })
		if code != tc.exitCode {
			t.Fatalf("args=%v expected exit %d got %d stderr=%s", tc.args, tc.exitCode, code, errBuf.String())
		}
		combined := out.String() + errBuf.String()
		if !strings.Contains(combined, tc.needle) {
			t.Fatalf("args=%v expected output to contain %q, got %q", tc.args, tc.needle, combined)
		}
	}
}

func TestRunDemoAndWrapCoveragePaths(t *testing.T) {
	_, now := setupCLIWorkspace(t)

	{
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runDemo([]string{"--nope"}, true, &out, &errBuf, func() time.Time { return now })
		if code != 6 || !strings.Contains(errBuf.String(), "E_INVALID_INPUT_SCHEMA") {
			t.Fatalf("expected demo unknown flag error, got code=%d stderr=%s", code, errBuf.String())
		}
	}
	{
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runDemo([]string{"--out-dir"}, true, &out, &errBuf, func() time.Time { return now })
		if code != 6 || !strings.Contains(errBuf.String(), "E_INVALID_INPUT_SCHEMA") {
			t.Fatalf("expected demo missing out-dir value error, got code=%d stderr=%s", code, errBuf.String())
		}
	}
	{
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runDemo([]string{"--out-dir", "wrkr-out"}, false, &out, &errBuf, func() time.Time { return now })
		if code != 0 {
			t.Fatalf("demo success failed: %d %s", code, errBuf.String())
		}
		if !strings.Contains(out.String(), "job_id=") || !strings.Contains(out.String(), "footer=") {
			t.Fatalf("unexpected demo text output: %s", out.String())
		}
	}

	cases := []struct {
		args     []string
		exitCode int
	}{
		{args: []string{}, exitCode: 6},
		{args: []string{"--job-id", "x"}, exitCode: 6},
		{args: []string{"--job-id"}, exitCode: 6},
		{args: []string{"--artifact"}, exitCode: 6},
		{args: []string{"--out-dir"}, exitCode: 6},
		{args: []string{"--nope", "--", "true"}, exitCode: 6},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runWrap(tc.args, true, &out, &errBuf, func() time.Time { return now })
		if code != tc.exitCode {
			t.Fatalf("args=%v expected exit=%d got=%d stderr=%s", tc.args, tc.exitCode, code, errBuf.String())
		}
		if !strings.Contains(errBuf.String(), "E_INVALID_INPUT_SCHEMA") {
			t.Fatalf("args=%v expected invalid input schema error, got %s", tc.args, errBuf.String())
		}
	}

	{
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runWrap([]string{"--job-id", "job_wrap_fail", "--", "sh", "-lc", "exit 1"}, true, &out, &errBuf, func() time.Time { return now })
		if code == 0 {
			t.Fatalf("expected wrap command failure, got success output=%s", out.String())
		}
		if !strings.Contains(out.String(), "\"jobpack_path\"") {
			t.Fatalf("expected jobpack payload before command error, got %s", out.String())
		}
	}
}

func TestAdditionalCommandCoveragePaths(t *testing.T) {
	workspace, now := setupCLIWorkspace(t)
	_ = workspace

	r := setupCLIJob(t, now, "job_cmd_paths", queue.StatusRunning)
	cp, err := r.EmitCheckpoint("job_cmd_paths", runner.CheckpointInput{
		Type:    "decision-needed",
		Summary: "approval required",
		Status:  queue.StatusBlockedDecision,
		RequiredAction: &v1.RequiredAction{
			Kind:         "approval",
			Instructions: "review",
		},
	})
	if err != nil {
		t.Fatalf("EmitCheckpoint: %v", err)
	}
	if _, err := r.ChangeStatus("job_cmd_paths", queue.StatusBlockedDecision); err != nil {
		t.Fatalf("ChangeStatus blocked_decision: %v", err)
	}

	errorCases := []struct {
		name     string
		runFn    func([]string, bool, io.Writer, io.Writer, func() time.Time) int
		args     []string
		exitCode int
	}{
		{name: "pause usage", runFn: runPause, args: []string{}, exitCode: 6},
		{name: "cancel usage", runFn: runCancel, args: []string{}, exitCode: 6},
		{name: "status usage", runFn: runStatus, args: []string{}, exitCode: 6},
		{name: "approve usage", runFn: runApprove, args: []string{}, exitCode: 6},
		{name: "approve checkpoint missing value", runFn: runApprove, args: []string{"job_cmd_paths", "--checkpoint"}, exitCode: 6},
		{name: "approve missing checkpoint", runFn: runApprove, args: []string{"job_cmd_paths", "--reason", "ok"}, exitCode: 6},
		{name: "approve bad flag", runFn: runApprove, args: []string{"job_cmd_paths", "--nope"}, exitCode: 6},
		{name: "submit usage", runFn: runSubmit, args: []string{}, exitCode: 6},
		{name: "submit missing job id value", runFn: runSubmit, args: []string{"jobspec.yaml", "--job-id"}, exitCode: 6},
		{name: "submit unknown flag", runFn: runSubmit, args: []string{"jobspec.yaml", "--nope"}, exitCode: 6},
		{name: "accept usage", runFn: runAccept, args: []string{}, exitCode: 6},
		{name: "accept unknown subcommand", runFn: runAccept, args: []string{"nope"}, exitCode: 6},
		{name: "accept init bad flag", runFn: runAccept, args: []string{"init", "--nope"}, exitCode: 6},
		{name: "accept init missing path value", runFn: runAccept, args: []string{"init", "--path"}, exitCode: 6},
		{name: "accept run usage", runFn: runAccept, args: []string{"run"}, exitCode: 6},
		{name: "accept run unknown flag", runFn: runAccept, args: []string{"run", "job_cmd_paths", "--nope"}, exitCode: 6},
		{name: "accept run missing config value", runFn: runAccept, args: []string{"run", "job_cmd_paths", "--config"}, exitCode: 6},
		{name: "doctor unknown flag", runFn: runDoctor, args: []string{"--nope"}, exitCode: 6},
		{name: "doctor missing serve-listen", runFn: runDoctor, args: []string{"--serve-listen"}, exitCode: 6},
		{name: "doctor bad max body", runFn: runDoctor, args: []string{"--serve-max-body-bytes", "bad"}, exitCode: 6},
	}

	for _, tc := range errorCases {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := tc.runFn(tc.args, true, &out, &errBuf, func() time.Time { return now })
		if code != tc.exitCode {
			t.Fatalf("%s expected exit=%d got=%d stderr=%s", tc.name, tc.exitCode, code, errBuf.String())
		}
		if !strings.Contains(errBuf.String(), "E_INVALID_INPUT_SCHEMA") {
			t.Fatalf("%s expected invalid input schema error, got %s", tc.name, errBuf.String())
		}
	}

	{
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runStatus([]string{"job_cmd_paths"}, false, &out, &errBuf, func() time.Time { return now })
		if code != 0 {
			t.Fatalf("status text failed: %d %s", code, errBuf.String())
		}
		if !strings.Contains(out.String(), "job=job_cmd_paths status=") {
			t.Fatalf("unexpected status text output: %s", out.String())
		}
	}

	{
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runApprove([]string{"job_cmd_paths", "--checkpoint", cp.CheckpointID, "--reason", "approved"}, false, &out, &errBuf, func() time.Time { return now })
		if code != 0 {
			t.Fatalf("approve text failed: %d %s", code, errBuf.String())
		}
		if !strings.Contains(out.String(), "approved checkpoint=") {
			t.Fatalf("unexpected approve text output: %s", out.String())
		}
	}

	{
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runPause([]string{"job_cmd_paths"}, false, &out, &errBuf, func() time.Time { return now })
		if code == 0 {
			t.Fatalf("expected pause to fail from blocked_decision status transition")
		}
		if !strings.Contains(errBuf.String(), "E_INVALID_STATE_TRANSITION") {
			t.Fatalf("expected invalid state transition for pause, got %s", errBuf.String())
		}
	}

	{
		var out bytes.Buffer
		var errBuf bytes.Buffer
		code := runDoctor([]string{"--production-readiness", "--serve-listen", "0.0.0.0:9488"}, true, &out, &errBuf, func() time.Time { return now })
		if code != 1 {
			t.Fatalf("expected doctor production-readiness failure exit 1, got %d stderr=%s", code, errBuf.String())
		}
		if !strings.Contains(out.String(), "\"ok\": false") {
			t.Fatalf("expected doctor result ok=false, got %s", out.String())
		}
	}
}
