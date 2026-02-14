package bridge

import (
	"strings"
	"testing"

	v1 "github.com/davidahmann/wrkr/core/schema/v1"
)

func TestBridgeHelperCoveragePaths(t *testing.T) {
	t.Parallel()

	if got := requiredAction(v1.Checkpoint{Type: "decision-needed"}); got != "approval" {
		t.Fatalf("expected decision-needed fallback action approval, got %q", got)
	}
	if got := requiredAction(v1.Checkpoint{Type: "blocked"}); got != "resume" {
		t.Fatalf("expected blocked fallback action resume, got %q", got)
	}
	if got := requiredAction(v1.Checkpoint{
		RequiredAction: &v1.RequiredAction{Instructions: "custom instructions"},
	}); got != "custom instructions" {
		t.Fatalf("expected required_action instructions fallback, got %q", got)
	}

	reasons := normalizedReasonCodes([]string{"A", " ", "A", "B"})
	if len(reasons) != 2 || reasons[0] != "A" || reasons[1] != "B" {
		t.Fatalf("unexpected normalized reason codes: %v", reasons)
	}

	artifacts := artifactPointers(v1.Checkpoint{
		ArtifactsDelta: v1.ArtifactsDelta{
			Added:   []string{"a", "a"},
			Changed: []string{"b"},
			Removed: []string{" "},
		},
	})
	if len(artifacts) != 2 || artifacts[0] != "a" || artifacts[1] != "b" {
		t.Fatalf("unexpected artifact pointers: %v", artifacts)
	}

	commands := nextCommands("job_x", v1.Checkpoint{Type: "decision-needed", CheckpointID: "cp_9"})
	if len(commands) < 4 || !strings.Contains(commands[0], "wrkr approve") {
		t.Fatalf("unexpected next commands: %v", commands)
	}

	jira := renderTemplate(v1.WorkItemPayload{
		JobID:          "job_x",
		CheckpointID:   "cp_9",
		CheckpointType: "decision-needed",
		RequiredAction: "approval",
		ReasonCodes:    []string{"E_CHECKPOINT_APPROVAL_REQUIRED"},
		NextCommands:   []string{"wrkr resume job_x"},
	}, "jira")
	if !strings.Contains(jira, "JIRA Work Item") {
		t.Fatalf("expected jira template, got %s", jira)
	}

	github := renderTemplate(v1.WorkItemPayload{
		JobID:          "job_x",
		CheckpointID:   "cp_9",
		CheckpointType: "decision-needed",
		RequiredAction: "approval",
		ReasonCodes:    []string{"E_CHECKPOINT_APPROVAL_REQUIRED"},
		NextCommands:   []string{"wrkr resume job_x"},
	}, "github")
	if !strings.Contains(github, "GitHub Work Item") {
		t.Fatalf("expected github template, got %s", github)
	}

	if got := bulletList(nil); got != "- (none)" {
		t.Fatalf("unexpected empty bullet list output: %q", got)
	}
	if got := sanitize(" ../job/id "); got != "__job_id" {
		t.Fatalf("unexpected sanitize output: %q", got)
	}
	if got := sanitize(" "); got != "value" {
		t.Fatalf("unexpected sanitize empty output: %q", got)
	}

	if _, err := marshalCanonical(make(chan int)); err == nil {
		t.Fatal("expected marshalCanonical marshal error")
	}
}

