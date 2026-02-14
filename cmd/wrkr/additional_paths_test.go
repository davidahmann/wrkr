package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
)

func seedStorePruneFixture(t *testing.T, storeRoot, outRoot string, now time.Time) {
	t.Helper()
	for _, dir := range []string{
		filepath.Join(storeRoot, "jobs", "job_old"),
		filepath.Join(outRoot, "jobpacks"),
		filepath.Join(outRoot, "reports"),
		filepath.Join(outRoot, "integrations", "lane"),
	} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeAged := func(path string, age time.Duration) {
		t.Helper()
		if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		mod := now.Add(-age)
		if err := os.Chtimes(path, mod, mod); err != nil {
			t.Fatalf("chtimes %s: %v", path, err)
		}
	}
	writeAged(filepath.Join(storeRoot, "jobs", "job_old", "events.jsonl"), 48*time.Hour)
	writeAged(filepath.Join(outRoot, "jobpacks", "jobpack_a.zip"), 48*time.Hour)
	writeAged(filepath.Join(outRoot, "reports", "report_a.json"), 48*time.Hour)
	writeAged(filepath.Join(outRoot, "integrations", "lane", "event.log"), 48*time.Hour)
}

func TestCLIAdditionalCoveragePaths(t *testing.T) {
	workspace, now := setupCLIWorkspace(t)
	nowFn := func() time.Time { return now }

	var out bytes.Buffer
	var errBuf bytes.Buffer

	if code := runInit([]string{"jobspec_cli_more.yaml"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runInit positional path failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runInit([]string{"--path", "jobspec_cli_more.yaml", "--force"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runInit force/json failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), `"jobspec_path"`) {
		t.Fatalf("unexpected runInit json output: %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	jobID := "job_cli_more"
	if code := runSubmit([]string{"jobspec_cli_more.yaml", "--job-id", jobID}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runSubmit failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runExport([]string{jobID, "--out-dir", "wrkr-out"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runExport json failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), `"manifest_sha256"`) {
		t.Fatalf("unexpected runExport json output: %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	// Cover wrap adapter failure branch after jobpack export.
	if code := runWrap([]string{"--job-id", "job_wrap_fail_cov", "--out-dir", "wrkr-out", "--", "false"}, false, &out, &errBuf, nowFn); code == 0 {
		t.Fatalf("expected runWrap to fail for command=false, stdout=%s stderr=%s", out.String(), errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	acceptPath := filepath.Join(workspace, "accept_more.yaml")
	writeAcceptConfigForCLI(t, acceptPath)
	if code := runAccept([]string{"init", "--path", acceptPath, "--force"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runAccept init failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runAccept([]string{"run", jobID, "--config", acceptPath, "--junit", "wrkr-out/reports/accept_more.junit.xml", "--out-dir", "wrkr-out"}, true, &out, &errBuf, nowFn); code != 5 {
		t.Fatalf("runAccept run with junit/json expected acceptance failure code=5, got code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), `"junit_path"`) {
		t.Fatalf("unexpected runAccept run output: %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	storeRoot := filepath.Join(workspace, ".wrkr")
	outRoot := filepath.Join(workspace, "wrkr-out-prune")
	seedStorePruneFixture(t, storeRoot, outRoot, now)

	if code := runStore([]string{
		"prune",
		"--dry-run",
		"--store-root", storeRoot,
		"--out-dir", outRoot,
		"--job-max-age", "1h",
		"--jobpack-max-age", "1h",
		"--report-max-age", "1h",
		"--integration-max-age", "1h",
		"--max-jobpacks", "0",
		"--max-reports", "0",
	}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runStore prune full criteria/json failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), `"dry_run": true`) {
		t.Fatalf("unexpected runStore prune json output: %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	// Exercise pause/cancel JSON success paths on separate running jobs.
	setupCLIJob(t, now, "job_pause_more", queue.StatusRunning)
	if code := runPause([]string{"job_pause_more"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runPause text failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	setupCLIJob(t, now, "job_cancel_more", queue.StatusRunning)
	if code := runCancel([]string{"job_cancel_more"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runCancel json failed: code=%d err=%s", code, errBuf.String())
	}
}

func TestCLICheckpointResumeStatusReportExtraCoverage(t *testing.T) {
	workspace, now := setupCLIWorkspace(t)
	nowFn := func() time.Time { return now }

	var out bytes.Buffer
	var errBuf bytes.Buffer

	jobID := "job_cli_extra_cov"
	if code := runInit([]string{"--path", "jobspec_extra.yaml"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runInit failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()
	if code := runSubmit([]string{"jobspec_extra.yaml", "--job-id", jobID}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runSubmit failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runCheckpoint([]string{"emit", jobID, "--type", "decision-needed", "--summary", "manual decision", "--status", "blocked_decision", "--required-kind", "approval", "--required-instructions", "review", "--reason-code", string(wrkrerrors.ECheckpointApprovalRequired)}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runCheckpoint emit decision-needed failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	r, _, err := openRunner(nowFn)
	if err != nil {
		t.Fatalf("openRunner: %v", err)
	}

	// Build an env-fingerprint mismatch on a dedicated paused job and cover --force-env resume path.
	setupCLIJob(t, now, "job_resume_env", queue.StatusRunning)
	if _, err := r.ChangeStatus("job_resume_env", queue.StatusPaused); err != nil {
		t.Fatalf("ChangeStatus job_resume_env paused: %v", err)
	}
	s, err := openStore()
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	if _, err := s.AppendEvent("job_resume_env", "env_fingerprint_set", map[string]any{
		"rules":       []string{"os"},
		"values":      map[string]string{"os": "bogus-os"},
		"hash":        "deadbeef",
		"captured_at": now.UTC(),
	}, now); err != nil {
		t.Fatalf("append env mismatch event: %v", err)
	}

	if code := runResume([]string{"job_resume_env", "--force-env", "--reason", "known drift", "--approved-by", "lead"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runResume --force-env failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	// Create a leased running job to cover status lease text branch.
	if _, err := r.InitJob("job_status_lease"); err != nil {
		t.Fatalf("InitJob status lease: %v", err)
	}
	if _, err := r.ChangeStatus("job_status_lease", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus status lease: %v", err)
	}
	if _, err := r.AcquireLease("job_status_lease", "worker-test", "lease-test"); err != nil {
		t.Fatalf("AcquireLease: %v", err)
	}
	if code := runStatus([]string{"job_status_lease"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runStatus lease branch failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "lease_worker=") {
		t.Fatalf("expected lease status output, got %s", out.String())
	}
	out.Reset()
	errBuf.Reset()

	if code := runExport([]string{jobID, "--out-dir", "wrkr-out"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runExport failed: code=%d err=%s", code, errBuf.String())
	}
	out.Reset()
	errBuf.Reset()

	stepSummary := filepath.Join(workspace, "gh", "step_summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", stepSummary)
	if code := runReport([]string{"github", jobID, "--out-dir", "wrkr-out"}, true, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runReport github with step summary failed: code=%d err=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), `"step_summary_path"`) {
		t.Fatalf("expected step summary path in output, got %s", out.String())
	}
}
