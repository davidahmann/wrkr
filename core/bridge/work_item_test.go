package bridge

import (
	"os"
	"testing"
	"time"

	v1 "github.com/davidahmann/wrkr/core/schema/v1"
)

func TestBuildWorkItemPayloadDeterministic(t *testing.T) {
	checkpoint := v1.Checkpoint{
		Envelope:     v1.Envelope{CreatedAt: time.Date(2026, 2, 14, 2, 20, 0, 0, time.UTC)},
		CheckpointID: "cp_10",
		Type:         "blocked",
		ReasonCodes:  []string{"E_BUDGET_EXCEEDED", "E_BUDGET_EXCEEDED"},
		ArtifactsDelta: v1.ArtifactsDelta{
			Added: []string{"reports/a.md", "reports/a.md"},
		},
	}

	a, err := BuildWorkItemPayload("job_bridge", checkpoint, BuildOptions{ProducerVersion: "test"})
	if err != nil {
		t.Fatalf("BuildWorkItemPayload A: %v", err)
	}
	b, err := BuildWorkItemPayload("job_bridge", checkpoint, BuildOptions{
		Now:             func() time.Time { return time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC) },
		ProducerVersion: "test",
	})
	if err != nil {
		t.Fatalf("BuildWorkItemPayload B: %v", err)
	}

	if a.CreatedAt != b.CreatedAt {
		t.Fatalf("expected stable created_at, got A=%s B=%s", a.CreatedAt, b.CreatedAt)
	}
	if len(a.ReasonCodes) != 1 || a.ReasonCodes[0] != "E_BUDGET_EXCEEDED" {
		t.Fatalf("unexpected reason codes: %+v", a.ReasonCodes)
	}
}

func TestWriteWorkItemPayload(t *testing.T) {
	payload := v1.WorkItemPayload{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.work_item",
			SchemaVersion:   "v1",
			CreatedAt:       time.Date(2026, 2, 14, 2, 20, 0, 0, time.UTC),
			ProducerVersion: "test",
		},
		JobID:          "job_bridge",
		CheckpointID:   "cp_9",
		CheckpointType: "decision-needed",
		RequiredAction: "approval",
		ReasonCodes:    []string{"E_CHECKPOINT_APPROVAL_REQUIRED"},
		NextCommands:   []string{"wrkr approve job_bridge --checkpoint cp_9 --reason approved"},
	}

	result, err := WriteWorkItemPayload(payload, t.TempDir(), "github")
	if err != nil {
		t.Fatalf("WriteWorkItemPayload: %v", err)
	}
	if result.JSONPath == "" || result.TemplatePath == "" {
		t.Fatalf("expected output paths, got %+v", result)
	}
	if _, err := os.Stat(result.JSONPath); err != nil {
		t.Fatalf("json output missing: %v", err)
	}
}
