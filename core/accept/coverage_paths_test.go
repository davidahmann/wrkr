package accept

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/accept/checks"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func TestRunCoveragePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	workspace := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	now := time.Date(2026, 2, 14, 18, 30, 0, 0, time.UTC)

	_, err = Run("job_missing_accept", RunOptions{Now: func() time.Time { return now }, WorkDir: workspace})
	if err == nil {
		t.Fatal("expected missing job error")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EInvalidInputSchema {
		t.Fatalf("expected E_INVALID_INPUT_SCHEMA, got %v", err)
	}

	setupAcceptJob(t, "job_accept_bad_config", now)
	if err := os.WriteFile(filepath.Join(workspace, "bad_accept.yaml"), []byte(":\n"), 0o600); err != nil {
		t.Fatalf("write bad config: %v", err)
	}
	if _, err := Run("job_accept_bad_config", RunOptions{
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
		ConfigPath:      "bad_accept.yaml",
		WorkDir:         workspace,
	}); err == nil {
		t.Fatal("expected bad config decode failure")
	}
}

func TestAcceptanceHelperCoveragePaths(t *testing.T) {
	t.Parallel()

	result := buildAcceptanceResult("job_accept_cov", "test", time.Date(2026, 2, 14, 18, 40, 0, 0, time.UTC), []checks.CheckResult{
		{Name: "test_command", Passed: false, Message: "failed", ReasonCode: wrkrerrors.EAcceptTestFail},
		{Name: "required_artifacts", Passed: false, Message: "missing", ReasonCode: wrkrerrors.EAcceptMissingArtifact},
		{Name: "required_artifacts_dup", Passed: false, Message: "missing", ReasonCode: wrkrerrors.EAcceptMissingArtifact},
		{Name: "lint_command", Passed: true, Message: "ok"},
	})
	if result.ChecksRun != 4 || result.ChecksPassed != 1 {
		t.Fatalf("unexpected checks summary: %+v", result)
	}
	if len(result.ReasonCodes) != 2 {
		t.Fatalf("expected deduped reason codes, got %+v", result.ReasonCodes)
	}
	if !Failed(result) {
		t.Fatalf("expected result with failures to be marked failed: %+v", result)
	}
	if code := FailureCode(result); code != wrkrerrors.EAcceptTestFail {
		t.Fatalf("expected E_ACCEPT_TEST_FAIL precedence, got %s", code)
	}

	if _, err := canonicalizeJSON(make(chan int)); err == nil {
		t.Fatal("expected canonicalizeJSON marshal failure")
	}
}

