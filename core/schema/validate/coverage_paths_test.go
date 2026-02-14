package validate

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateBytesCoveragePaths(t *testing.T) {
	t.Parallel()

	valid := map[string]any{
		"schema_id":        "wrkr.job",
		"schema_version":   "v1",
		"created_at":       time.Date(2026, 2, 14, 11, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"producer_version": "test",
		"job_id":           "job_schema_valid",
		"name":             "job_schema_valid",
		"status":           "queued",
		"budgets": map[string]any{
			"retry_count":     0,
			"step_count":      0,
			"tool_call_count": 0,
		},
	}
	raw, err := json.Marshal(valid)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := ValidateBytes(JobSchemaRel, raw); err != nil {
		t.Fatalf("ValidateBytes valid payload: %v", err)
	}

	if err := ValidateBytes(JobSchemaRel, []byte("{not json")); err == nil {
		t.Fatal("expected invalid json decode error")
	}

	if err := ValidateBytes(JobSchemaRel, []byte(`{"schema_id":"wrkr.job"}`)); err == nil {
		t.Fatal("expected schema validation failure")
	}
}

func TestCompileAndValidateValueCoveragePaths(t *testing.T) {
	t.Parallel()

	if _, err := Compile("missing-schema.json"); err == nil {
		t.Fatal("expected missing schema compile error")
	}

	if err := ValidateValue(JobSchemaRel, map[string]any{"schema_id": "wrkr.job"}); err == nil {
		t.Fatal("expected ValidateValue schema error")
	}
}

