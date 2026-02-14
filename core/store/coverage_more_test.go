package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateJobIDCoveragePaths(t *testing.T) {
	t.Parallel()

	invalid := []string{"", " ", "../bad", "bad/name", `bad\name`, "bad..name", "bad*name"}
	for _, jobID := range invalid {
		if err := validateJobID(jobID); err == nil {
			t.Fatalf("expected invalid job id to fail: %q", jobID)
		}
	}
	if err := validateJobID("job_ok-1.2_3"); err != nil {
		t.Fatalf("expected valid job id: %v", err)
	}
}

func TestSnapshotCoveragePathsAdditional(t *testing.T) {
	t.Parallel()

	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Date(2026, 2, 14, 18, 0, 0, 0, time.UTC)

	if err := s.SaveSnapshot("job_snapshot_cov", 7, map[string]any{"status": "running"}, now); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}
	snap, err := s.LoadSnapshot("job_snapshot_cov")
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if snap == nil || snap.LastSeq != 7 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}

	if err := s.SaveSnapshot("job_snapshot_bad_state", 1, make(chan int), now); err == nil {
		t.Fatal("expected snapshot marshal state error")
	}

	if err := s.EnsureJob("job_snapshot_bad_decode"); err != nil {
		t.Fatalf("EnsureJob: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.JobDir("job_snapshot_bad_decode"), "snapshot.json"), []byte("{bad"), 0o600); err != nil {
		t.Fatalf("write malformed snapshot: %v", err)
	}
	if _, err := s.LoadSnapshot("job_snapshot_bad_decode"); err == nil {
		t.Fatal("expected snapshot decode error")
	}
}
