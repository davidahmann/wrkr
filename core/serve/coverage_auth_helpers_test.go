package serve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestServeAuthCoveragePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 20, 30, 0, 0, time.UTC)
	srv := New(Config{
		AuthToken:     "secret",
		MaxBodyBytes:  1 << 20,
		Now:           func() time.Time { return now },
		ProducerVersion: "test",
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/jobs:submit", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/jobs:submit", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer wrong")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong auth, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/unknown", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected auth-pass unknown endpoint 404, got %d", rec.Code)
	}
}

func TestServeHelperCoveragePaths(t *testing.T) {
	t.Parallel()

	if host := hostPart("127.0.0.1:9488"); host != "127.0.0.1" {
		t.Fatalf("unexpected hostPart parsed host: %q", host)
	}
	if host := hostPart("invalid-listen"); host != "invalid-listen" {
		t.Fatalf("unexpected hostPart fallback host: %q", host)
	}

	if !isLoopbackHost("localhost") || !isLoopbackHost("127.0.0.1") || isLoopbackHost("0.0.0.0") {
		t.Fatal("unexpected loopback host classification")
	}

	var payload struct {
		OutDir string `json:"out_dir"`
	}
	if err := decodeJSONOptional(io.NopCloser(strings.NewReader("")), &payload); err != nil {
		t.Fatalf("decodeJSONOptional empty body: %v", err)
	}
	if err := decodeJSONOptional(io.NopCloser(strings.NewReader(`{"out_dir":"x","extra":1}`)), &payload); err == nil {
		t.Fatal("expected decodeJSONOptional unknown-field failure")
	}
	if err := decodeJSON(io.NopCloser(strings.NewReader(`{"out_dir":"x","extra":1}`)), &payload); err == nil {
		t.Fatal("expected decodeJSON unknown-field failure")
	}
}

func TestServeCheckpointsInvalidPathCoverage(t *testing.T) {
	_, cleanup := setupServeWorkspace(t)
	t.Cleanup(cleanup)

	now := time.Date(2026, 2, 14, 20, 40, 0, 0, time.UTC)
	srv := New(Config{
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
		MaxBodyBytes:    1 << 20,
	})
	jobID := submitTestJob(t, srv, now, "job_serve_bad_cp_path_extra")

	rec := makeRequest(t, srv, http.MethodGet, "/v1/jobs/"+jobID+"/checkpoints/extra/path", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid checkpoints path 400, got %d", rec.Code)
	}
}
