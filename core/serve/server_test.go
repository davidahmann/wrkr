package serve

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateConfigNonLoopbackGuardrails(t *testing.T) {
	_, err := ValidateConfig(Config{ListenAddr: "0.0.0.0:9488"})
	if err == nil {
		t.Fatal("expected non-loopback guardrail error")
	}

	cfg, err := ValidateConfig(Config{
		ListenAddr:       "0.0.0.0:9488",
		AllowNonLoopback: true,
		AuthToken:        "token",
		MaxBodyBytes:     1024,
	})
	if err != nil {
		t.Fatalf("ValidateConfig: %v", err)
	}
	if cfg.ListenAddr != "0.0.0.0:9488" {
		t.Fatalf("unexpected listen addr: %s", cfg.ListenAddr)
	}
}

func TestAuthAndBodyLimit(t *testing.T) {
	srv := New(Config{
		Now:          func() time.Time { return time.Date(2026, 2, 14, 2, 40, 0, 0, time.UTC) },
		AuthToken:    "secret",
		MaxBodyBytes: 16,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/jobs:submit", strings.NewReader(`{"jobspec_path":"x"}`))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/jobs:submit", bytes.NewBuffer(bytes.Repeat([]byte("x"), 128)))
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("expected body-limit handling, got unauthorized")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 from oversized invalid json, got %d", rec.Code)
	}
}

func TestPathTraversalRejected(t *testing.T) {
	srv := New(Config{
		Now:          func() time.Time { return time.Date(2026, 2, 14, 2, 40, 0, 0, time.UTC) },
		MaxBodyBytes: 1024,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/jobs/job_1:export", strings.NewReader(`{"out_dir":"../bad"}`))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for traversal, got %d", rec.Code)
	}
}

func TestJobIDTraversalRejectedOnMutationEndpoints(t *testing.T) {
	srv := New(Config{
		Now:          func() time.Time { return time.Date(2026, 2, 14, 2, 40, 0, 0, time.UTC) },
		MaxBodyBytes: 1024,
	})

	cases := []struct {
		path string
		body string
	}{
		{path: "/v1/jobs/..:export", body: `{}`},
		{path: "/v1/jobs/..:verify", body: `{}`},
		{path: "/v1/jobs/..:accept", body: `{"config_path":"accept.yaml"}`},
		{path: "/v1/jobs/..:report-github", body: `{}`},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", tc.path, rec.Code)
		}
	}
}
