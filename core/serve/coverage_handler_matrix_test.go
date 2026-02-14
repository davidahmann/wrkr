package serve

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/store"
)

func TestServeHandlerMatrixCoverage(t *testing.T) {
	_, cleanup := setupServeWorkspace(t)
	t.Cleanup(cleanup)

	now := time.Date(2026, 2, 14, 22, 0, 0, 0, time.UTC)
	srv := New(Config{
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
		MaxBodyBytes:    1 << 20,
	})
	jobID := submitTestJob(t, srv, now, "job_serve_matrix")

	// Checkpoint path validation branches.
	rec := makeRequest(t, srv, http.MethodGet, "/v1/jobs/"+jobID+"/checkpoints/..", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected unsafe checkpoint_id 400, got %d", rec.Code)
	}
	rec = makeRequest(t, srv, http.MethodGet, "/v1/jobs/"+jobID+"/checkpoints/bad!id", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid checkpoint_id format 400, got %d", rec.Code)
	}

	// Approve validation branches.
	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/bad!id:approve", `{"checkpoint_id":"cp_1","reason":"ok","approved_by":"lead"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid approve job_id format 400, got %d", rec.Code)
	}
	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/missing_job:approve", `{"checkpoint_id":"cp_1","reason":"ok","approved_by":"lead"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected missing approve job 404, got %d", rec.Code)
	}
	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/"+jobID+":approve", `{"checkpoint_id":"bad!id","reason":"ok","approved_by":"lead"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid approve checkpoint_id format 400, got %d", rec.Code)
	}
	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/"+jobID+":approve", `{"checkpoint_id":"cp_1","reason":"ok","approved_by":"lead","extra":1}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected unknown-field approve payload 400, got %d", rec.Code)
	}

	// decodeJSONOptional empty-body branch for export/verify/report.
	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/"+jobID+":export", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected empty-body export success, got %d body=%s", rec.Code, rec.Body.String())
	}
	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/"+jobID+":verify", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected empty-body verify success, got %d body=%s", rec.Code, rec.Body.String())
	}
	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/"+jobID+":report-github", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected empty-body report-github success, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/missing_job:report-github", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing report-github job bad request, got %d", rec.Code)
	}

	// status recover error branch via malformed events.jsonl.
	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.JobDir(jobID), "events.jsonl"), []byte("{bad-json\n"), 0o600); err != nil {
		t.Fatalf("write malformed events: %v", err)
	}
	rec = makeRequest(t, srv, http.MethodGet, "/v1/jobs/"+jobID+":status", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status recover error 500, got %d", rec.Code)
	}

	// verify missing jobpack branch.
	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/"+jobID+":verify", `{"out_dir":"missing-out"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected verify missing jobpack 400, got %d", rec.Code)
	}

	// accept missing job branch.
	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/missing_job:accept", `{"config_path":"accept.yaml"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected accept missing job 400, got %d", rec.Code)
	}

	// report write failure branch by pointing step summary to invalid parent.
	reportJob := submitTestJob(t, srv, now, "job_serve_matrix_report_error")
	blockingParent := filepath.Join(t.TempDir(), "blocked-parent")
	if err := os.WriteFile(blockingParent, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking parent file: %v", err)
	}
	t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(blockingParent, "summary.md"))
	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/"+reportJob+":report-github", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected report write error 500, got %d", rec.Code)
	}
}
