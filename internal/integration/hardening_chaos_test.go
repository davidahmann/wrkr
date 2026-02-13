package integration

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/pack"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
)

func TestLeaseHeartbeatExpiryAndSafeReclaim(t *testing.T) {
	current := time.Date(2026, 2, 14, 5, 0, 0, 0, time.UTC)
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{
		Now:      func() time.Time { return current },
		LeaseTTL: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_lease_chaos"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.AcquireLease("job_lease_chaos", "worker-a", "lease-a"); err != nil {
		t.Fatalf("acquire lease-a: %v", err)
	}

	if _, err := r.HeartbeatLease("job_lease_chaos", "worker-b", "lease-b"); err == nil {
		t.Fatal("expected heartbeat conflict for wrong worker/lease")
	} else {
		var werr wrkrerrors.WrkrError
		if !errors.As(err, &werr) || werr.Code != wrkrerrors.ELeaseConflict {
			t.Fatalf("expected E_LEASE_CONFLICT heartbeat failure, got %v", err)
		}
	}

	current = current.Add(31 * time.Second)
	state, err := r.AcquireLease("job_lease_chaos", "worker-b", "lease-b")
	if err != nil {
		t.Fatalf("expected reclaim after expiry: %v", err)
	}
	if state.Lease == nil || state.Lease.WorkerID != "worker-b" || state.Lease.LeaseID != "lease-b" {
		t.Fatalf("expected reclaimed lease-b, got %+v", state.Lease)
	}
}

func TestResumeEnvMismatchOverrideWritesAuditTrail(t *testing.T) {
	now := time.Date(2026, 2, 14, 5, 30, 0, 0, time.UTC)
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_env_audit"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_env_audit", queue.StatusRunning); err != nil {
		t.Fatalf("running: %v", err)
	}
	if _, err := r.ChangeStatus("job_env_audit", queue.StatusPaused); err != nil {
		t.Fatalf("paused: %v", err)
	}

	if _, err := s.AppendEvent("job_env_audit", "env_fingerprint_set", map[string]any{
		"rules":       []string{"os"},
		"values":      map[string]string{"os": "bogus-os"},
		"hash":        "deadbeef",
		"captured_at": now.UTC(),
	}, now); err != nil {
		t.Fatalf("inject mismatch fingerprint: %v", err)
	}

	if _, err := r.Resume("job_env_audit", runner.ResumeInput{}); err == nil {
		t.Fatal("expected env mismatch error")
	}

	if _, err := r.Resume("job_env_audit", runner.ResumeInput{
		OverrideEnvMismatch: true,
		OverrideReason:      "approved drift",
		ApprovedBy:          "ops",
	}); err != nil {
		t.Fatalf("resume override: %v", err)
	}

	events, err := s.LoadEvents("job_env_audit")
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	found := false
	for _, event := range events {
		if event.Type != "env_override_recorded" {
			continue
		}
		found = true
		var payload map[string]any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("unmarshal override payload: %v", err)
		}
		if payload["reason"] != "approved drift" || payload["approved_by"] != "ops" {
			t.Fatalf("unexpected override payload: %+v", payload)
		}
	}
	if !found {
		t.Fatal("expected env_override_recorded event")
	}
}

func TestTraceAndArtifactUniqueness(t *testing.T) {
	now := time.Date(2026, 2, 14, 5, 45, 0, 0, time.UTC)
	home := t.TempDir()
	t.Setenv("HOME", home)
	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob("job_uniqueness"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := r.ChangeStatus("job_uniqueness", queue.StatusRunning); err != nil {
		t.Fatalf("running: %v", err)
	}
	if _, err := r.EmitCheckpoint("job_uniqueness", runner.CheckpointInput{
		Type:    "progress",
		Summary: "first artifact write",
		ArtifactsDelta: v1.ArtifactsDelta{
			Added: []string{"reports/unique.md"},
		},
	}); err != nil {
		t.Fatalf("emit checkpoint 1: %v", err)
	}
	if _, err := r.EmitCheckpoint("job_uniqueness", runner.CheckpointInput{
		Type:    "progress",
		Summary: "second artifact touch",
		ArtifactsDelta: v1.ArtifactsDelta{
			Changed: []string{"reports/unique.md"},
		},
	}); err != nil {
		t.Fatalf("emit checkpoint 2: %v", err)
	}

	exported, err := pack.ExportJobpack("job_uniqueness", pack.ExportOptions{
		OutDir:          t.TempDir(),
		Now:             func() time.Time { return now },
		ProducerVersion: "test",
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	archive, err := pack.LoadArchive(exported.Path)
	if err != nil {
		t.Fatalf("load archive: %v", err)
	}
	events, err := pack.DecodeEvents(archive.Files)
	if err != nil {
		t.Fatalf("decode events: %v", err)
	}
	seenEventIDs := map[string]bool{}
	for _, evt := range events {
		if evt.EventID == "" {
			t.Fatalf("event_id must be set: %+v", evt)
		}
		if seenEventIDs[evt.EventID] {
			t.Fatalf("duplicate event_id found: %s", evt.EventID)
		}
		seenEventIDs[evt.EventID] = true
	}

	rawArtifacts, ok := archive.Files["artifacts_manifest.json"]
	if !ok {
		t.Fatal("artifacts_manifest.json missing")
	}
	var artifacts v1.ArtifactsManifest
	if err := json.Unmarshal(rawArtifacts, &artifacts); err != nil {
		t.Fatalf("decode artifacts manifest: %v", err)
	}
	seenPaths := map[string]bool{}
	for _, item := range artifacts.Artifacts {
		if seenPaths[item.Path] {
			t.Fatalf("duplicate artifact path in manifest: %s", item.Path)
		}
		seenPaths[item.Path] = true
	}
}
