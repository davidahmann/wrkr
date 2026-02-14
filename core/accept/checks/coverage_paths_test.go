package checks

import (
	"strings"
	"testing"
	"time"

	v1 "github.com/davidahmann/wrkr/core/schema/v1"
)

func TestSchemaValidityCoveragePaths(t *testing.T) {
	t.Parallel()

	input := testInput("reports/out.md")

	input.StatusResponse.SchemaID = ""
	result, err := checkSchemaValidity(input)
	if err != nil {
		t.Fatalf("checkSchemaValidity status path: %v", err)
	}
	if result.Passed {
		t.Fatalf("expected schema failure for invalid status payload: %+v", result)
	}

	input = testInput("reports/out.md")
	input.Checkpoints[0].SchemaID = ""
	result, err = checkSchemaValidity(input)
	if err != nil {
		t.Fatalf("checkSchemaValidity checkpoint path: %v", err)
	}
	if result.Passed {
		t.Fatalf("expected schema failure for invalid checkpoint payload: %+v", result)
	}

	input = testInput("reports/out.md")
	input.Approvals = []v1.ApprovalRecord{
		{
			Envelope: v1.Envelope{
				SchemaID:        "",
				SchemaVersion:   "v1",
				CreatedAt:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ProducerVersion: "test",
			},
			JobID:        "job_accept",
			CheckpointID: "cp_1",
			Reason:       "ok",
			ApprovedBy:   "lead",
		},
	}
	result, err = checkSchemaValidity(input)
	if err != nil {
		t.Fatalf("checkSchemaValidity approval path: %v", err)
	}
	if result.Passed {
		t.Fatalf("expected schema failure for invalid approval payload: %+v", result)
	}
}

func TestPathConstraintAndCommandCoveragePaths(t *testing.T) {
	t.Parallel()

	base := testInput("reports/out.md")
	base.Checkpoints[0].ArtifactsDelta.Changed = []string{"reports/changed.md"}
	base.Checkpoints[0].ArtifactsDelta.Removed = []string{"tmp/old.md"}

	result := checkPathConstraints(PathRules{MaxArtifactPaths: 1}, base.Checkpoints)
	if result.Passed {
		t.Fatalf("expected max artifact path failure: %+v", result)
	}

	result = checkPathConstraints(PathRules{ForbiddenPrefixes: []string{"reports/"}}, base.Checkpoints)
	if result.Passed {
		t.Fatalf("expected forbidden prefix failure: %+v", result)
	}

	result = checkPathConstraints(PathRules{AllowedPrefixes: []string{"docs/"}}, base.Checkpoints)
	if result.Passed {
		t.Fatalf("expected allowed prefix mismatch failure: %+v", result)
	}

	cmdResult, err := runCommandCheck("test_command", "", ".")
	if err != nil {
		t.Fatalf("runCommandCheck empty command: %v", err)
	}
	if cmdResult.Passed {
		t.Fatalf("expected empty command failure: %+v", cmdResult)
	}

	cmdResult, err = runCommandCheck("test_command", "echo "+strings.Repeat("x", 500)+"; false", ".")
	if err != nil {
		t.Fatalf("runCommandCheck failing command: %v", err)
	}
	if cmdResult.Passed || !strings.Contains(cmdResult.Message, "command failed") {
		t.Fatalf("expected failing command result, got %+v", cmdResult)
	}

	artifacts := collectArtifacts(base.Checkpoints, true)
	if _, ok := artifacts["tmp/old.md"]; !ok {
		t.Fatalf("expected removed artifact when includeRemoved=true: %+v", artifacts)
	}

	paths := sortedArtifactPaths(map[string]struct{}{
		"reports/out.md": {},
		" ":              {},
	})
	if len(paths) != 1 || paths[0] != "reports/out.md" {
		t.Fatalf("unexpected sortedArtifactPaths output: %v", paths)
	}

	uniq := sortedUnique([]string{"a", " ", "a", "b"})
	if len(uniq) != 2 || uniq[0] != "a" || uniq[1] != "b" {
		t.Fatalf("unexpected sortedUnique output: %v", uniq)
	}

	short := boundedText([]byte("ok"))
	if short != "ok" {
		t.Fatalf("expected short boundedText passthrough, got %q", short)
	}
	if got := boundedText([]byte("")); got != "(no output)" {
		t.Fatalf("expected empty boundedText fallback, got %q", got)
	}
}

