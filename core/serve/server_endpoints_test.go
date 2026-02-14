package serve

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/projectconfig"
)

func setupServeWorkspace(t *testing.T) (string, func()) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	workspace := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("chdir workspace: %v", err)
	}
	return workspace, func() {
		_ = os.Chdir(orig)
	}
}

func makeRequest(t *testing.T, srv *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func decodeJSONBody(t *testing.T, body io.Reader) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		t.Fatalf("decode json body: %v", err)
	}
	return payload
}

func writeAcceptConfig(t *testing.T, path string) {
	t.Helper()
	raw := `schema_id: wrkr.accept_config
schema_version: v1
required_artifacts: []
test_command: "true"
lint_command: "true"
path_rules:
  max_artifact_paths: 0
  forbidden_prefixes: []
  allowed_prefixes: []
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write accept config: %v", err)
	}
}

func submitTestJob(t *testing.T, srv *Server, now time.Time, jobID string) string {
	t.Helper()

	specPath, err := projectconfig.InitJobSpec("jobspec.yaml", true, now, "test")
	if err != nil {
		t.Fatalf("InitJobSpec: %v", err)
	}

	relSpec := filepath.Base(specPath)
	rec := makeRequest(
		t,
		srv,
		http.MethodPost,
		"/v1/jobs:submit",
		`{"jobspec_path":"`+relSpec+`","job_id":"`+jobID+`"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	return jobID
}

func decisionCheckpointID(t *testing.T, srv *Server, jobID string) string {
	t.Helper()
	rec := makeRequest(t, srv, http.MethodGet, "/v1/jobs/"+jobID+"/checkpoints", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list checkpoints failed: %d %s", rec.Code, rec.Body.String())
	}
	var checkpoints []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&checkpoints); err != nil {
		t.Fatalf("decode checkpoints: %v", err)
	}
	for _, cp := range checkpoints {
		if cp["type"] == "decision-needed" {
			id, _ := cp["checkpoint_id"].(string)
			if strings.TrimSpace(id) != "" {
				return id
			}
		}
	}
	t.Fatal("decision-needed checkpoint not found")
	return ""
}

func TestServerHandlerAndUnknownEndpoint(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 5, 0, 0, 0, time.UTC)
	srv := New(Config{
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
	})

	if srv.Handler() == nil {
		t.Fatal("expected non-nil handler")
	}

	rec := makeRequest(t, srv, http.MethodGet, "/v1/unknown", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "E_INVALID_INPUT_SCHEMA") {
		t.Fatalf("expected invalid input schema error, got %s", rec.Body.String())
	}
}

func TestServeSubmitStatusCheckpointsApproveFlow(t *testing.T) {
	workspace, cleanup := setupServeWorkspace(t)
	t.Cleanup(cleanup)

	now := time.Date(2026, 2, 14, 5, 5, 0, 0, time.UTC)
	srv := New(Config{
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
		MaxBodyBytes:    1 << 20,
	})

	jobID := submitTestJob(t, srv, now, "job_serve_flow")

	rec := makeRequest(t, srv, http.MethodGet, "/v1/jobs/"+jobID+":status", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	statusPayload := decodeJSONBody(t, rec.Body)
	if statusPayload["job_id"] != jobID {
		t.Fatalf("expected job_id %q, got %#v", jobID, statusPayload["job_id"])
	}

	decisionID := decisionCheckpointID(t, srv, jobID)
	rec = makeRequest(t, srv, http.MethodGet, "/v1/jobs/"+jobID+"/checkpoints/"+decisionID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("checkpoint show failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	cpPayload := decodeJSONBody(t, rec.Body)
	if cpPayload["checkpoint_id"] != decisionID {
		t.Fatalf("expected checkpoint_id %q, got %#v", decisionID, cpPayload["checkpoint_id"])
	}

	rec = makeRequest(
		t,
		srv,
		http.MethodPost,
		"/v1/jobs/"+jobID+":approve",
		`{"checkpoint_id":"`+decisionID+`","reason":"looks good","approved_by":"lead"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	approvePayload := decodeJSONBody(t, rec.Body)
	if approvePayload["checkpoint_id"] != decisionID {
		t.Fatalf("expected approved checkpoint %q, got %#v", decisionID, approvePayload["checkpoint_id"])
	}

	configPath := filepath.Join(workspace, "accept.yaml")
	writeAcceptConfig(t, configPath)
	rec = makeRequest(
		t,
		srv,
		http.MethodPost,
		"/v1/jobs/"+jobID+":accept",
		`{"config_path":"accept.yaml"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("accept failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	acceptPayload := decodeJSONBody(t, rec.Body)
	if _, ok := acceptPayload["accept_result_path"]; !ok {
		t.Fatalf("expected accept_result_path in payload: %#v", acceptPayload)
	}
}

func TestServeExportVerifyAndReportGitHubFlow(t *testing.T) {
	_, cleanup := setupServeWorkspace(t)
	t.Cleanup(cleanup)

	now := time.Date(2026, 2, 14, 5, 10, 0, 0, time.UTC)
	srv := New(Config{
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
		MaxBodyBytes:    1 << 20,
	})

	jobID := submitTestJob(t, srv, now, "job_serve_pack")
	outDir := "wrkr-out"

	rec := makeRequest(
		t,
		srv,
		http.MethodPost,
		"/v1/jobs/"+jobID+":export",
		`{"out_dir":"`+outDir+`"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("export failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	exportPayload := decodeJSONBody(t, rec.Body)
	if exportPayload["job_id"] != jobID {
		t.Fatalf("expected export job_id %q, got %#v", jobID, exportPayload["job_id"])
	}

	rec = makeRequest(
		t,
		srv,
		http.MethodPost,
		"/v1/jobs/"+jobID+":verify",
		`{"out_dir":"`+outDir+`"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	verifyPayload := decodeJSONBody(t, rec.Body)
	if verifyPayload["job_id"] != jobID {
		t.Fatalf("expected verify job_id %q, got %#v", jobID, verifyPayload["job_id"])
	}

	rec = makeRequest(
		t,
		srv,
		http.MethodPost,
		"/v1/jobs/"+jobID+":report-github",
		`{"out_dir":"`+outDir+`"}`,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("report-github failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	reportPayload := decodeJSONBody(t, rec.Body)
	if _, ok := reportPayload["summary_json_path"]; !ok {
		t.Fatalf("expected summary_json_path in report payload: %#v", reportPayload)
	}
	if _, ok := reportPayload["summary_markdown_path"]; !ok {
		t.Fatalf("expected summary_markdown_path in report payload: %#v", reportPayload)
	}
}

func TestServeRejectsUnsafePathInputs(t *testing.T) {
	_, cleanup := setupServeWorkspace(t)
	t.Cleanup(cleanup)

	now := time.Date(2026, 2, 14, 5, 15, 0, 0, time.UTC)
	srv := New(Config{
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
		MaxBodyBytes:    128,
	})

	rec := makeRequest(t, srv, http.MethodPost, "/v1/jobs:submit", `{"jobspec_path":"../bad.yaml"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for jobspec traversal, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodGet, "/v1/jobs/../evil:status", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsafe status job_id, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodGet, "/v1/jobs/bad!id/checkpoints", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid checkpoints job_id, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/job_x:approve", `{"checkpoint_id":"..","reason":"ok"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsafe checkpoint_id, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/job_x:export", `{"out_dir":"/tmp"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for absolute out_dir, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/job_x:verify", `{"out_dir":"../tmp"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for traversal out_dir, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/job_x:accept", `{"config_path":"/tmp/accept.yaml"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for absolute config_path, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/job_x:report-github", `{"out_dir":"/tmp"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for absolute report out_dir, got %d", rec.Code)
	}

	oversized := bytes.Repeat([]byte("x"), 1024)
	req := httptest.NewRequest(http.MethodPost, "/v1/jobs/job_x:export", bytes.NewReader(oversized))
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized request body, got %d", rec.Code)
	}
}

func TestServeHandlerValidationErrors(t *testing.T) {
	_, cleanup := setupServeWorkspace(t)
	t.Cleanup(cleanup)

	now := time.Date(2026, 2, 14, 5, 18, 0, 0, time.UTC)
	srv := New(Config{
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
		MaxBodyBytes:    1 << 20,
	})

	rec := makeRequest(t, srv, http.MethodPost, "/v1/jobs:submit", `{"jobspec_path":"jobspec.yaml","job_id":"bad!id"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid job id format, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs:submit", `{"jobspec_path":"/tmp/jobspec.yaml"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for absolute jobspec path, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs:submit", `{"jobspec_path":"jobspec.yaml","unknown":1}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field in submit payload, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodGet, "/v1/jobs/missing_job:status", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing job status, got %d", rec.Code)
	}

	jobID := submitTestJob(t, srv, now, "job_serve_bad_path")
	rec = makeRequest(t, srv, http.MethodGet, "/v1/jobs/"+jobID+"/checkpoints/a/b", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid checkpoints path, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/job_x:verify", `{"out_dir":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed verify payload, got %d", rec.Code)
	}

	rec = makeRequest(t, srv, http.MethodPost, "/v1/jobs/job_x:accept", `{"config_path":"accept.yaml","extra":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown accept payload field, got %d", rec.Code)
	}
}

func TestServeApproveAndCheckpointNotFoundPaths(t *testing.T) {
	_, cleanup := setupServeWorkspace(t)
	t.Cleanup(cleanup)

	now := time.Date(2026, 2, 14, 5, 19, 0, 0, time.UTC)
	srv := New(Config{
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
		MaxBodyBytes:    1 << 20,
	})

	jobID := submitTestJob(t, srv, now, "job_serve_validation")

	rec := makeRequest(t, srv, http.MethodGet, "/v1/jobs/"+jobID+"/checkpoints/cp_99999", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing checkpoint, got %d", rec.Code)
	}

	decisionID := decisionCheckpointID(t, srv, jobID)

	rec = makeRequest(
		t,
		srv,
		http.MethodPost,
		"/v1/jobs/"+jobID+":approve",
		`{"checkpoint_id":"`+decisionID+`","reason":"","approved_by":"lead"}`,
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty approval reason, got %d", rec.Code)
	}

	listRec := makeRequest(t, srv, http.MethodGet, "/v1/jobs/"+jobID+"/checkpoints", "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected checkpoints list success, got %d", listRec.Code)
	}
	var checkpoints []map[string]any
	if err := json.NewDecoder(listRec.Body).Decode(&checkpoints); err != nil {
		t.Fatalf("decode checkpoints: %v", err)
	}
	progressID := ""
	for _, cp := range checkpoints {
		if cp["type"] == "progress" {
			progressID, _ = cp["checkpoint_id"].(string)
			break
		}
	}
	if strings.TrimSpace(progressID) == "" {
		t.Fatal("expected a progress checkpoint in fixture job")
	}

	rec = makeRequest(
		t,
		srv,
		http.MethodPost,
		"/v1/jobs/"+jobID+":approve",
		`{"checkpoint_id":"`+progressID+`","reason":"ok","approved_by":"lead"}`,
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for approving non decision-needed checkpoint, got %d", rec.Code)
	}
}

func TestListenAndServeConfigValidationAndListenFailure(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 5, 20, 0, 0, time.UTC)

	srv := New(Config{
		ListenAddr: "0.0.0.0:9488",
		Now:        func() time.Time { return now },
	})
	if err := srv.ListenAndServe(); err == nil {
		t.Fatal("expected non-loopback guardrail error")
	}

	srv = New(Config{
		ListenAddr:       "127.0.0.1:-1",
		AllowNonLoopback: false,
		Now:              func() time.Time { return now },
	})
	if err := srv.ListenAndServe(); err == nil {
		t.Fatal("expected listen failure for invalid port")
	}
}
