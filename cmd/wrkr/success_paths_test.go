package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
)

func writeAcceptConfigForCLI(t *testing.T, path string) {
	t.Helper()
	raw := `schema_id: wrkr.accept_config
schema_version: v1
required_artifacts: []
test_command: "true"
lint_command: "true"
path_rules:
  max_artifact_paths: 0
  forbidden_prefixes: []
  allowed_prefixes: []
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write accept config: %v", err)
	}
}

func checkpointByType(t *testing.T, r *runner.Runner, jobID string, cpType string) string {
	t.Helper()
	checkpoints, err := r.ListCheckpoints(jobID)
	if err != nil {
		t.Fatalf("ListCheckpoints: %v", err)
	}
	for _, cp := range checkpoints {
		if cp.Type == cpType {
			return cp.CheckpointID
		}
	}
	t.Fatalf("checkpoint type %q not found", cpType)
	return ""
}

func TestCLIRealSuccessFlowsCoverage(t *testing.T) {
	workspace, now := setupCLIWorkspace(t)
	_ = workspace

	var out bytes.Buffer
	var errBuf bytes.Buffer

	jobID := "job_cli_matrix"
	nowFn := func() time.Time { return now }

	if code := runInit([]string{"--path", "jobspec.yaml"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runInit failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runSubmit([]string{"jobspec.yaml", "--job-id", jobID}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runSubmit failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), `"job_id": "job_cli_matrix"`) {
		t.Fatalf("unexpected submit output: %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runStatus([]string{jobID}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runStatus failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "job=job_cli_matrix") {
		t.Fatalf("unexpected status output: %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	r, _, err := openRunner(nowFn)
	if err != nil {
		t.Fatalf("openRunner: %v", err)
	}
	decisionID := checkpointByType(t, r, jobID, "decision-needed")

	if code := runCheckpoint([]string{"emit", jobID, "--type", "progress", "--summary", "manual checkpoint", "--reason-code", "E_ADAPTER_FAIL"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runCheckpoint emit failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runCheckpoint([]string{"show", jobID, decisionID}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runCheckpoint show failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "checkpoint="+decisionID) {
		t.Fatalf("unexpected checkpoint show output: %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runApprove([]string{jobID, "--checkpoint", decisionID, "--reason", "looks good", "--approved-by", "lead"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runApprove failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runResume([]string{jobID}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runResume failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "status=completed") {
		t.Fatalf("unexpected resume output: %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runBudget(
		[]string{
			"check", jobID,
			"--max-wall-time-seconds", "1000",
			"--max-retries", "10",
			"--max-step-count", "100",
			"--max-tool-calls", "100",
		},
		false, &out, &errBuf, nowFn,
	); code != 0 {
		t.Fatalf("runBudget failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runExport([]string{jobID, "--out-dir", "wrkr-out"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runExport failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "jobpack=") {
		t.Fatalf("unexpected export output: %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	jobpackPath := filepath.Join("wrkr-out", "jobpacks", fmt.Sprintf("jobpack_%s.zip", jobID))
	if code := runVerify([]string{jobpackPath}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runVerify(path) failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runVerify([]string{jobID, "--out-dir", "wrkr-out"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runVerify(jobID) failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runJob([]string{"inspect", jobID, "--out-dir", "wrkr-out"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runJob inspect(jobID) failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runJob([]string{"inspect", jobpackPath}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runJob inspect(path) failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runJob([]string{"diff", jobpackPath, jobpackPath}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runJob diff failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runReceipt([]string{jobID, "--out-dir", "wrkr-out"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runReceipt(jobID) failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runReceipt([]string{jobpackPath}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runReceipt(path) failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runReport([]string{"github", jobID, "--out-dir", "wrkr-out"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runReport github(jobID) failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runReport([]string{"github", jobpackPath, "--out-dir", "wrkr-out"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runReport github(path) failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	writeAcceptConfigForCLI(t, "accept.yaml")
	if code := runAccept([]string{"run", jobID, "--config", "accept.yaml", "--ci", "--out-dir", "wrkr-out"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runAccept run failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runStore([]string{"prune", "--dry-run", "--store-root", filepath.Join(os.Getenv("HOME"), ".wrkr"), "--out-dir", "wrkr-out", "--job-max-age", "1h"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runStore prune failed: code=%d err=%s", code, errBuf.String())
	}
}

func TestCLIWrapPauseCancelDoctorSuccessCoverage(t *testing.T) {
	_, now := setupCLIWorkspace(t)
	nowFn := func() time.Time { return now }

	var out bytes.Buffer
	var errBuf bytes.Buffer

	r := setupCLIJob(t, now, "job_pause_cancel", queue.StatusRunning)
	if code := runPause([]string{"job_pause_cancel"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runPause failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if _, err := r.ChangeStatus("job_pause_cancel", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus back to running: %v", err)
	}
	if code := runCancel([]string{"job_pause_cancel"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runCancel failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runWrap([]string{"--job-id", "job_wrap_cov", "--artifact", "reports/out.md", "--out-dir", "wrkr-out", "--", "true"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runWrap failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), `"job_id": "job_wrap_cov"`) {
		t.Fatalf("unexpected wrap output: %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runDoctor([]string{}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runDoctor default failed: code=%d err=%s", code, errBuf.String())
	}
}

