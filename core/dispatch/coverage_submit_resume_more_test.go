package dispatch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/fsx"
	"github.com/davidahmann/wrkr/core/projectconfig"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
)

func TestSubmitCoverageMore(t *testing.T) {
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

	now := time.Date(2026, 2, 14, 20, 0, 0, 0, time.UTC)
	specPath, err := projectconfig.InitJobSpec("jobspec_dispatch_more.yaml", true, now, "test")
	if err != nil {
		t.Fatalf("InitJobSpec: %v", err)
	}

	first, err := Submit(specPath, SubmitOptions{Now: func() time.Time { return now }, JobID: "job_dispatch_dup"})
	if err != nil {
		t.Fatalf("Submit first: %v", err)
	}
	if first.JobID != "job_dispatch_dup" {
		t.Fatalf("unexpected first submit job id: %s", first.JobID)
	}

	_, err = Submit(specPath, SubmitOptions{Now: func() time.Time { return now }, JobID: "job_dispatch_dup"})
	if err == nil {
		t.Fatal("expected duplicate submit failure")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EInvalidInputSchema {
		t.Fatalf("expected E_INVALID_INPUT_SCHEMA for duplicate submit, got %v", err)
	}

	auto, err := Submit(specPath, SubmitOptions{Now: func() time.Time { return now }, JobID: ""})
	if err != nil {
		t.Fatalf("Submit auto id: %v", err)
	}
	if !strings.HasPrefix(auto.JobID, "demo_refactor_job_") {
		t.Fatalf("expected inferred job id prefix, got %s", auto.JobID)
	}
}

func TestResumeCoverageMore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 20, 10, 0, 0, time.UTC)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_resume_nil_cfg"); err != nil {
		t.Fatalf("InitJob: %v", err)
	}
	if _, err := r.ChangeStatus("job_resume_nil_cfg", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus: %v", err)
	}

	result, err := Resume("job_resume_nil_cfg", ResumeOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("Resume without runtime config: %v", err)
	}
	if result.JobID != "job_resume_nil_cfg" {
		t.Fatalf("unexpected resume result: %+v", result)
	}

	if _, err := r.InitJob("job_resume_bad_cfg"); err != nil {
		t.Fatalf("InitJob bad cfg: %v", err)
	}
	if _, err := r.ChangeStatus("job_resume_bad_cfg", queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus bad cfg: %v", err)
	}
	if err := fsx.AtomicWriteFile(filepath.Join(s.JobDir("job_resume_bad_cfg"), "runtime_config.json"), []byte("{bad"), 0o600); err != nil {
		t.Fatalf("write bad runtime config: %v", err)
	}
	if _, err := Resume("job_resume_bad_cfg", ResumeOptions{Now: func() time.Time { return now }}); err == nil {
		t.Fatal("expected resume failure for malformed runtime config")
	}
}

