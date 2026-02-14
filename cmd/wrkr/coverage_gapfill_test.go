package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/pack"
	"github.com/davidahmann/wrkr/core/queue"
)

type failingWriterCoverage struct{}

func (failingWriterCoverage) Write(_ []byte) (int, error) {
	return 0, errors.New("forced write failure")
}

func exportCoverageJobpack(t *testing.T, now time.Time, jobID, outDir string) string {
	t.Helper()

	r := setupCLIJob(t, now, jobID, queue.StatusRunning)
	if _, err := r.ChangeStatus(jobID, queue.StatusCompleted); err != nil {
		t.Fatalf("ChangeStatus completed: %v", err)
	}
	exported, err := pack.ExportJobpack(jobID, pack.ExportOptions{
		OutDir:          outDir,
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
	})
	if err != nil {
		t.Fatalf("ExportJobpack: %v", err)
	}
	return exported.Path
}

func TestCLIMainEncodeFailureCoverage(t *testing.T) {
	now := time.Date(2026, 2, 15, 2, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }
	var stderr bytes.Buffer

	if code := run([]string{"--json", "--explain", "status"}, failingWriterCoverage{}, &stderr, nowFn); code != 1 {
		t.Fatalf("expected explain encode failure exit=1, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "encode explain") {
		t.Fatalf("expected explain encode error, got %s", stderr.String())
	}

	stderr.Reset()
	if code := run([]string{"--json"}, failingWriterCoverage{}, &stderr, nowFn); code != 1 {
		t.Fatalf("expected version encode failure exit=1, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "encode version") {
		t.Fatalf("expected version encode error, got %s", stderr.String())
	}

	stderr.Reset()
	if code := run([]string{"--json", "--explain", "not_a_real_command"}, &bytes.Buffer{}, &stderr, nowFn); code != 6 {
		t.Fatalf("expected unknown explain command exit=6, got %d stderr=%s", code, stderr.String())
	}
}

func TestCLIResolveJobpackPathCoverage(t *testing.T) {
	workspace, _ := setupCLIWorkspace(t)

	path, isPath, err := resolveJobpackPath("", "")
	if err != nil || path != "" || isPath {
		t.Fatalf("expected empty target to return empty/not-path/nil, got path=%q isPath=%t err=%v", path, isPath, err)
	}

	if err := os.MkdirAll(filepath.Join(workspace, "fixtures"), 0o750); err != nil {
		t.Fatalf("mkdir fixtures: %v", err)
	}
	relativeFile := filepath.Join("fixtures", "jobpack_fixture.zip")
	if err := os.WriteFile(relativeFile, []byte("zip"), 0o600); err != nil {
		t.Fatalf("write relative fixture: %v", err)
	}

	path, isPath, err = resolveJobpackPath(relativeFile, "")
	if err != nil || !isPath || path != relativeFile {
		t.Fatalf("expected relative file path match, got path=%q isPath=%t err=%v", path, isPath, err)
	}

	absoluteFile, err := filepath.Abs(relativeFile)
	if err != nil {
		t.Fatalf("abs fixture: %v", err)
	}
	path, isPath, err = resolveJobpackPath(absoluteFile, "")
	if err != nil || !isPath || path != absoluteFile {
		t.Fatalf("expected absolute file path match, got path=%q isPath=%t err=%v", path, isPath, err)
	}

	path, isPath, err = resolveJobpackPath("job_resolve_cov", "wrkr-out")
	if err != nil || isPath || !strings.Contains(path, "jobpack_job_resolve_cov.zip") {
		t.Fatalf("expected derived jobpack path, got path=%q isPath=%t err=%v", path, isPath, err)
	}

	if _, _, err := resolveJobpackPath("job_resolve_cov", "\x00bad"); err == nil {
		t.Fatal("expected invalid outDir to fail")
	}
}

func TestCLIStorePruneValidationAndEncodeCoverage(t *testing.T) {
	workspace, now := setupCLIWorkspace(t)
	nowFn := func() time.Time { return now }
	storeRoot := filepath.Join(workspace, ".wrkr")
	outRoot := filepath.Join(workspace, "wrkr-out")
	if err := os.MkdirAll(filepath.Join(storeRoot, "jobs"), 0o750); err != nil {
		t.Fatalf("mkdir store root: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(outRoot, "jobpacks"), 0o750); err != nil {
		t.Fatalf("mkdir out root: %v", err)
	}

	cases := [][]string{
		{"prune", "--store-root"},
		{"prune", "--out-dir"},
		{"prune", "--jobpack-max-age"},
		{"prune", "--jobpack-max-age", "0"},
		{"prune", "--report-max-age"},
		{"prune", "--report-max-age", "bad"},
		{"prune", "--integration-max-age"},
		{"prune", "--integration-max-age", "bad"},
		{"prune", "--max-reports"},
		{"prune", "--max-reports", "-1"},
		{"prune", "--unknown"},
		{"prune", "--dry-run", "--store-root", storeRoot, "--out-dir", outRoot},
	}
	for _, args := range cases {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		if code := runStore(args, false, &out, &errBuf, nowFn); code != 6 {
			t.Fatalf("args=%v expected exit=6 got=%d stderr=%s", args, code, errBuf.String())
		}
	}

	var stderr bytes.Buffer
	code := runStore(
		[]string{"prune", "--dry-run", "--store-root", storeRoot, "--out-dir", outRoot, "--job-max-age", "1h"},
		true,
		failingWriterCoverage{},
		&stderr,
		nowFn,
	)
	if code != 1 {
		t.Fatalf("expected json encode failure exit=1, got %d stderr=%s", code, stderr.String())
	}
}

func TestCLIVerifyReportJobStatusAndWrapCoverage(t *testing.T) {
	_, now := setupCLIWorkspace(t)
	nowFn := func() time.Time { return now }
	jobpackPath := exportCoverageJobpack(t, now, "job_cov_verify_report", "wrkr-out")

	// verify flag-validation branches
	for _, args := range [][]string{
		{},
		{"target", "--out-dir"},
		{"target", "--bad-flag"},
	} {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		if code := runVerify(args, false, &out, &errBuf, nowFn); code != 6 {
			t.Fatalf("runVerify args=%v expected exit=6 got=%d stderr=%s", args, code, errBuf.String())
		}
	}

	// report flag-validation branches
	for _, args := range [][]string{
		{},
		{"github"},
		{"github", jobpackPath, "--out-dir"},
		{"github", jobpackPath, "--bad"},
	} {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		if code := runReport(args, false, &out, &errBuf, nowFn); code != 6 {
			t.Fatalf("runReport args=%v expected exit=6 got=%d stderr=%s", args, code, errBuf.String())
		}
	}

	// json encoding error branches
	var stderr bytes.Buffer
	if code := runVerify([]string{jobpackPath}, true, failingWriterCoverage{}, &stderr, nowFn); code != 1 {
		t.Fatalf("expected runVerify json encode failure exit=1, got %d stderr=%s", code, stderr.String())
	}
	stderr.Reset()
	if code := runReport([]string{"github", jobpackPath, "--out-dir", "wrkr-out"}, true, failingWriterCoverage{}, &stderr, nowFn); code != 1 {
		t.Fatalf("expected runReport json encode failure exit=1, got %d stderr=%s", code, stderr.String())
	}

	// runJob inspect/diff json encoding error branches
	stderr.Reset()
	if code := runJob([]string{"inspect", jobpackPath}, true, failingWriterCoverage{}, &stderr, nowFn); code != 1 {
		t.Fatalf("expected runJob inspect json encode failure exit=1, got %d stderr=%s", code, stderr.String())
	}
	stderr.Reset()
	if code := runJob([]string{"diff", jobpackPath, jobpackPath}, true, failingWriterCoverage{}, &stderr, nowFn); code != 1 {
		t.Fatalf("expected runJob diff json encode failure exit=1, got %d stderr=%s", code, stderr.String())
	}

	// runStatus json encode error path
	setupCLIJob(t, now, "job_cov_status_json", queue.StatusRunning)
	stderr.Reset()
	if code := runStatus([]string{"job_cov_status_json"}, true, failingWriterCoverage{}, &stderr, nowFn); code != 1 {
		t.Fatalf("expected runStatus json encode failure exit=1, got %d stderr=%s", code, stderr.String())
	}

	// runWrap missing-value and unknown-flag branches plus non-json success.
	for _, args := range [][]string{
		{"--job-id", "--", "true"},
		{"--artifact", "--", "true"},
		{"--out-dir", "--", "true"},
		{"--unknown", "x", "--", "true"},
	} {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		if code := runWrap(args, false, &out, &errBuf, nowFn); code != 6 {
			t.Fatalf("runWrap args=%v expected exit=6 got=%d stderr=%s", args, code, errBuf.String())
		}
	}
	var out bytes.Buffer
	var errBuf bytes.Buffer
	if code := runWrap([]string{"--job-id", "job_wrap_text_cov", "--out-dir", "wrkr-out", "--", "true"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runWrap non-json success expected exit=0 got=%d stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "job_id=job_wrap_text_cov") {
		t.Fatalf("expected non-json wrap output, got %s", out.String())
	}
}

func TestCLIPrintErrorMarshalFailureCoverage(t *testing.T) {
	now := time.Date(2026, 2, 15, 2, 30, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }
	var stderr bytes.Buffer

	err := wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "bad details", map[string]any{
		"unserializable": make(chan int),
	})
	if code := printError(err, true, &stderr, nowFn); code != 1 {
		t.Fatalf("expected marshal-envelope failure exit=1, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "marshal error envelope") {
		t.Fatalf("expected marshal envelope fallback error, got %s", stderr.String())
	}
}

func TestCLIFinalGapfillCoverage(t *testing.T) {
	_, now := setupCLIWorkspace(t)
	nowFn := func() time.Time { return now }

	var out bytes.Buffer
	var errBuf bytes.Buffer

	if code := runAccept([]string{"init", "--path", "accept_cov_init.yaml", "--force"}, false, &out, &errBuf, nowFn); code != 0 {
		t.Fatalf("runAccept init text expected exit=0 got=%d stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "config=") {
		t.Fatalf("expected accept init text output, got %s", out.String())
	}

	out.Reset()
	errBuf.Reset()
	if code := runVerify([]string{"job_missing_cov", "--out-dir", "\x00bad"}, false, &out, &errBuf, nowFn); code == 0 {
		t.Fatalf("expected runVerify resolve error, stdout=%s stderr=%s", out.String(), errBuf.String())
	}

	out.Reset()
	errBuf.Reset()
	if code := runVerify([]string{"does-not-exist.zip"}, false, &out, &errBuf, nowFn); code == 0 {
		t.Fatalf("expected runVerify missing archive failure, stdout=%s stderr=%s", out.String(), errBuf.String())
	}

	out.Reset()
	errBuf.Reset()
	if code := runExport([]string{"job_missing_export_cov"}, false, &out, &errBuf, nowFn); code == 0 {
		t.Fatalf("expected runExport missing job failure, stdout=%s stderr=%s", out.String(), errBuf.String())
	}

	errBuf.Reset()
	if code := runInit([]string{"--path", "jobspec_encode_fail.yaml", "--force"}, true, failingWriterCoverage{}, &errBuf, nowFn); code != 1 {
		t.Fatalf("expected runInit json encode failure exit=1 got=%d stderr=%s", code, errBuf.String())
	}

	out.Reset()
	errBuf.Reset()
	if code := runWrap([]string{"--out-dir", "\x00bad", "--", "true"}, false, &out, &errBuf, nowFn); code == 0 {
		t.Fatalf("expected runWrap export failure, stdout=%s stderr=%s", out.String(), errBuf.String())
	}

	errBuf.Reset()
	if code := runWrap([]string{"--job-id", "job_wrap_json_cov", "--out-dir", "wrkr-out", "--", "true"}, true, failingWriterCoverage{}, &errBuf, nowFn); code != 1 {
		t.Fatalf("expected runWrap json encode failure exit=1 got=%d stderr=%s", code, errBuf.String())
	}
}
