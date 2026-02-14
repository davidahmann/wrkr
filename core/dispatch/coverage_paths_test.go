package dispatch

import (
	"errors"
	"strings"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/fsx"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
)

func setupDispatchRunner(t *testing.T, now time.Time) (*store.LocalStore, *runner.Runner) {
	t.Helper()
	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	return s, r
}

func initDispatchJob(t *testing.T, r *runner.Runner, jobID string) {
	t.Helper()
	if _, err := r.InitJob(jobID); err != nil {
		t.Fatalf("InitJob(%s): %v", jobID, err)
	}
	if _, err := r.ChangeStatus(jobID, queue.StatusRunning); err != nil {
		t.Fatalf("ChangeStatus(%s): %v", jobID, err)
	}
}

func TestRuntimeConfigCoveragePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	now := time.Date(2026, 2, 14, 13, 0, 0, 0, time.UTC)
	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	if err := SaveRuntimeConfig(nil, "job_cfg", RuntimeConfig{}, now); err == nil {
		t.Fatal("expected nil store error")
	}
	if _, err := LoadRuntimeConfig(nil, "job_cfg"); err == nil {
		t.Fatal("expected nil store error")
	}

	cfg := RuntimeConfig{
		Adapter:       "reference",
		Inputs:        nil,
		NextStepIndex: -5,
	}
	if err := SaveRuntimeConfig(s, "job_cfg", cfg, now); err != nil {
		t.Fatalf("SaveRuntimeConfig: %v", err)
	}

	loaded, err := LoadRuntimeConfig(s, "job_cfg")
	if err != nil {
		t.Fatalf("LoadRuntimeConfig: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected runtime config")
	}
	if loaded.SchemaID == "" || loaded.SchemaVersion == "" {
		t.Fatalf("expected schema defaults, got %+v", loaded)
	}
	if loaded.NextStepIndex != 0 {
		t.Fatalf("expected normalized next_step_index=0, got %d", loaded.NextStepIndex)
	}
	if loaded.Inputs == nil {
		t.Fatal("expected normalized non-nil inputs")
	}

	if err := s.EnsureJob("job_bad_cfg"); err != nil {
		t.Fatalf("EnsureJob: %v", err)
	}
	if err := fsx.AtomicWriteFile(runtimeConfigPath(s, "job_bad_cfg"), []byte("{bad-json"), 0o600); err != nil {
		t.Fatalf("write bad runtime config: %v", err)
	}
	if _, err := LoadRuntimeConfig(s, "job_bad_cfg"); err == nil {
		t.Fatal("expected decode runtime config failure")
	}

	missing, err := LoadRuntimeConfig(s, "job_missing_cfg")
	if err != nil {
		t.Fatalf("LoadRuntimeConfig missing: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil missing runtime config, got %+v", missing)
	}
}

func TestDispatchHelperCoveragePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 13, 10, 0, 0, time.UTC)
	if got := inferJobID("", now); !strings.HasPrefix(got, "job_") {
		t.Fatalf("expected inferred fallback prefix, got %q", got)
	}
	if got := adapterNameOrDefault("  "); got != "reference" {
		t.Fatalf("expected reference default adapter, got %q", got)
	}
	if got := adapterNameOrDefault(" NoOp "); got != "noop" {
		t.Fatalf("expected normalized adapter noop, got %q", got)
	}

	maxCost := 12.5
	maxTokens := 1024
	spec := v1.BudgetSpec{
		MaxWallTimeSeconds: 100,
		MaxRetries:         3,
		MaxStepCount:       5,
		MaxToolCalls:       9,
		MaxEstimatedCost:   &maxCost,
		MaxTokens:          &maxTokens,
	}
	limits := budgetFromSpec(spec)
	maxCost = 0
	maxTokens = 0
	if limits.MaxEstimatedCost == nil || *limits.MaxEstimatedCost != 12.5 {
		t.Fatalf("expected copied max estimated cost, got %+v", limits.MaxEstimatedCost)
	}
	if limits.MaxTokens == nil || *limits.MaxTokens != 1024 {
		t.Fatalf("expected copied max tokens, got %+v", limits.MaxTokens)
	}
}

func TestRunAdapterCoveragePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	now := time.Date(2026, 2, 14, 13, 20, 0, 0, time.UTC)
	s, r := setupDispatchRunner(t, now)

	initDispatchJob(t, r, "job_adapter_noop")
	res, err := runAdapter("noop", "job_adapter_noop", &RuntimeConfig{}, r, s, func() time.Time { return now })
	if err != nil {
		t.Fatalf("runAdapter noop: %v", err)
	}
	if res.Status != queue.StatusCompleted {
		t.Fatalf("expected completed noop status, got %s", res.Status)
	}

	_, err = runAdapter("unsupported", "job_adapter_noop", &RuntimeConfig{}, r, s, func() time.Time { return now })
	if err == nil {
		t.Fatal("expected unsupported adapter error")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EInvalidInputSchema {
		t.Fatalf("expected E_INVALID_INPUT_SCHEMA, got %v", err)
	}

	initDispatchJob(t, r, "job_adapter_ref_bad")
	_, err = runAdapter("reference", "job_adapter_ref_bad", &RuntimeConfig{Inputs: map[string]any{}}, r, s, func() time.Time { return now })
	if err == nil {
		t.Fatal("expected missing steps error")
	}

	initDispatchJob(t, r, "job_adapter_ref_ok")
	cfg := &RuntimeConfig{
		Adapter: "reference",
		Inputs: map[string]any{
			"steps": []any{
				map[string]any{
					"id":       "one",
					"summary":  "step one",
					"command":  "true",
					"executed": true,
				},
			},
		},
	}
	refRes, err := runAdapter("reference", "job_adapter_ref_ok", cfg, r, s, func() time.Time { return now })
	if err != nil {
		t.Fatalf("runAdapter reference: %v", err)
	}
	if refRes.Status != queue.StatusCompleted || refRes.NextStepIndex != 1 {
		t.Fatalf("unexpected reference result: %+v", refRes)
	}
}

func TestExecuteWithLeaseRunErrorCoverage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	now := time.Date(2026, 2, 14, 13, 30, 0, 0, time.UTC)
	_, r := setupDispatchRunner(t, now)
	initDispatchJob(t, r, "job_lease_error")

	_, err := executeWithLease(
		r,
		"job_lease_error",
		func() time.Time { return now },
		func() (adapterRunResult, error) {
			return adapterRunResult{Status: queue.StatusRunning}, errors.New("forced failure")
		},
	)
	if err == nil {
		t.Fatal("expected executeWithLease to return run error")
	}
}
