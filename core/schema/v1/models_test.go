package v1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJobSpecUnmarshalFromFixture(t *testing.T) {
	t.Parallel()

	fixturePath := filepath.Join("..", "testdata", "valid", "jobspec.json")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var wrapped struct {
		Schema string          `json:"schema"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(content, &wrapped); err != nil {
		t.Fatalf("unmarshal wrapper: %v", err)
	}

	var spec JobSpec
	if err := json.Unmarshal(wrapped.Data, &spec); err != nil {
		t.Fatalf("unmarshal jobspec: %v", err)
	}

	if spec.SchemaID != "wrkr.jobspec" {
		t.Fatalf("unexpected schema id: %s", spec.SchemaID)
	}
	if spec.Adapter.Name == "" {
		t.Fatal("expected adapter name")
	}
}
