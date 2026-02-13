package checks

import (
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
)

func TestRunAllChecksPass(t *testing.T) {
	t.Parallel()

	results, err := Run(
		Config{
			RequiredArtifacts: []string{"reports/out.md"},
			TestCommand:       "true",
			LintCommand:       "true",
			PathRules: PathRules{
				AllowedPrefixes: []string{"reports/"},
			},
		},
		testInput("reports/out.md"),
	)
	if err != nil {
		t.Fatalf("run checks: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 checks, got %d", len(results))
	}
	for _, result := range results {
		if !result.Passed {
			t.Fatalf("expected check %s to pass: %+v", result.Name, result)
		}
	}
}

func TestRunMissingArtifactFails(t *testing.T) {
	t.Parallel()

	results, err := Run(
		Config{RequiredArtifacts: []string{"reports/missing.md"}, TestCommand: "true", LintCommand: "true"},
		testInput("reports/out.md"),
	)
	if err != nil {
		t.Fatalf("run checks: %v", err)
	}

	found := false
	for _, result := range results {
		if result.Name == "required_artifacts" {
			found = true
			if result.Passed {
				t.Fatalf("expected required_artifacts failure, got %+v", result)
			}
			if result.ReasonCode != wrkrerrors.EAcceptMissingArtifact {
				t.Fatalf("expected E_ACCEPT_MISSING_ARTIFACT, got %s", result.ReasonCode)
			}
		}
	}
	if !found {
		t.Fatal("required_artifacts check missing")
	}
}

func TestRunCommandFailureFails(t *testing.T) {
	t.Parallel()

	results, err := Run(
		Config{TestCommand: "false", LintCommand: "true"},
		testInput("reports/out.md"),
	)
	if err != nil {
		t.Fatalf("run checks: %v", err)
	}

	found := false
	for _, result := range results {
		if result.Name == "test_command" {
			found = true
			if result.Passed {
				t.Fatalf("expected test_command failure, got %+v", result)
			}
			if result.ReasonCode != wrkrerrors.EAcceptTestFail {
				t.Fatalf("expected E_ACCEPT_TEST_FAIL, got %s", result.ReasonCode)
			}
		}
	}
	if !found {
		t.Fatal("test_command check missing")
	}
}

func testInput(artifact string) Input {
	now := time.Date(2026, 2, 13, 20, 0, 0, 0, time.UTC)
	return Input{
		StatusResponse: v1.StatusResponse{
			Envelope: v1.Envelope{
				SchemaID:        "wrkr.status_response",
				SchemaVersion:   "v1",
				CreatedAt:       now,
				ProducerVersion: "test",
			},
			JobID:   "job_accept",
			Status:  "running",
			Summary: "ok",
		},
		Checkpoints: []v1.Checkpoint{
			{
				Envelope: v1.Envelope{
					SchemaID:        "wrkr.checkpoint",
					SchemaVersion:   "v1",
					CreatedAt:       now,
					ProducerVersion: "test",
				},
				CheckpointID: "cp_1",
				JobID:        "job_accept",
				Type:         "progress",
				Summary:      "artifact captured",
				Status:       "running",
				BudgetState: v1.BudgetState{
					WallTimeSeconds: 5,
					RetryCount:      0,
					StepCount:       1,
					ToolCallCount:   1,
				},
				ArtifactsDelta: v1.ArtifactsDelta{
					Added:   []string{artifact},
					Changed: []string{},
					Removed: []string{},
				},
				ReasonCodes: []string{},
			},
		},
		Approvals: []v1.ApprovalRecord{},
		WorkDir:   ".",
	}
}
