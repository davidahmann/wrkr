package pack

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
	"github.com/davidahmann/wrkr/core/zipx"
)

func setupJob(t *testing.T, jobID string, now time.Time) {
	t.Helper()
	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.InitJob(jobID); err != nil {
		t.Fatalf("init job: %v", err)
	}
	if _, err := r.ChangeStatus(jobID, queue.StatusRunning); err != nil {
		t.Fatalf("status running: %v", err)
	}
	if _, err := r.UpdateCounters(jobID, 1, 2, 3); err != nil {
		t.Fatalf("update counters: %v", err)
	}
	if _, err := r.EmitCheckpoint(jobID, runner.CheckpointInput{Type: "progress", Summary: "step complete"}); err != nil {
		t.Fatalf("emit checkpoint: %v", err)
	}
}

func TestExportDeterministicAndVerifiable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 18, 0, 0, 0, time.UTC)
	setupJob(t, "job_pack", now)

	outDirA := filepath.Join(t.TempDir(), "out-a")
	outDirB := filepath.Join(t.TempDir(), "out-b")

	a, err := ExportJobpack("job_pack", ExportOptions{
		OutDir:          outDirA,
		ProducerVersion: "test",
		Now:             func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("export A: %v", err)
	}
	b, err := ExportJobpack("job_pack", ExportOptions{
		OutDir:          outDirB,
		ProducerVersion: "test",
		Now:             func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("export B: %v", err)
	}

	zipA, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatalf("read zip A: %v", err)
	}
	zipB, err := os.ReadFile(b.Path)
	if err != nil {
		t.Fatalf("read zip B: %v", err)
	}
	if string(zipA) != string(zipB) {
		t.Fatal("expected deterministic zip bytes")
	}

	verify, err := VerifyJobpack(a.Path)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if verify.JobID != "job_pack" {
		t.Fatalf("unexpected verify job id: %s", verify.JobID)
	}
}

func TestVerifyDetectsTampering(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 18, 0, 0, 0, time.UTC)
	setupJob(t, "job_tamper", now)

	exported, err := ExportJobpack("job_tamper", ExportOptions{
		OutDir:          t.TempDir(),
		ProducerVersion: "test",
		Now:             func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	archive, err := LoadArchive(exported.Path)
	if err != nil {
		t.Fatalf("load archive: %v", err)
	}
	archive.Files["job.json"] = []byte(`{"tampered":true}`)

	entries := make([]zipx.Entry, 0, len(archive.Files))
	for name, data := range archive.Files {
		entries = append(entries, zipx.Entry{Name: name, Data: data})
	}
	tamperedZip, err := zipx.BuildDeterministic(entries)
	if err != nil {
		t.Fatalf("build tampered zip: %v", err)
	}
	tamperedPath := filepath.Join(t.TempDir(), "tampered.zip")
	if err := os.WriteFile(tamperedPath, tamperedZip, 0o600); err != nil {
		t.Fatalf("write tampered zip: %v", err)
	}

	_, err = VerifyJobpack(tamperedPath)
	if err == nil {
		t.Fatal("expected tampered verify to fail")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EVerifyHashMismatch {
		t.Fatalf("expected E_VERIFY_HASH_MISMATCH, got %v", err)
	}
}

func TestVerifyRejectsUndeclaredArchiveEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 18, 0, 0, 0, time.UTC)
	setupJob(t, "job_extra_entry", now)

	exported, err := ExportJobpack("job_extra_entry", ExportOptions{
		OutDir:          t.TempDir(),
		ProducerVersion: "test",
		Now:             func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	archive, err := LoadArchive(exported.Path)
	if err != nil {
		t.Fatalf("load archive: %v", err)
	}
	archive.Files["smuggled.txt"] = []byte("malicious extra file")

	entries := make([]zipx.Entry, 0, len(archive.Files))
	for name, data := range archive.Files {
		entries = append(entries, zipx.Entry{Name: name, Data: data})
	}
	tamperedZip, err := zipx.BuildDeterministic(entries)
	if err != nil {
		t.Fatalf("build tampered zip: %v", err)
	}
	tamperedPath := filepath.Join(t.TempDir(), "tampered-extra-entry.zip")
	if err := os.WriteFile(tamperedPath, tamperedZip, 0o600); err != nil {
		t.Fatalf("write tampered zip: %v", err)
	}

	_, err = VerifyJobpack(tamperedPath)
	if err == nil {
		t.Fatal("expected undeclared file verify failure")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EVerifyHashMismatch {
		t.Fatalf("expected E_VERIFY_HASH_MISMATCH, got %v", err)
	}
}

func TestVerifyRejectsOversizedInvalidJSONLRecord(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 18, 0, 0, 0, time.UTC)
	setupJob(t, "job_jsonl_overflow", now)

	exported, err := ExportJobpack("job_jsonl_overflow", ExportOptions{
		OutDir:          t.TempDir(),
		ProducerVersion: "test",
		Now:             func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	archive, err := LoadArchive(exported.Path)
	if err != nil {
		t.Fatalf("load archive: %v", err)
	}
	archive.Files["events.jsonl"] = []byte(
		`{"schema_id":"wrkr.event","schema_version":"v1","created_at":"2026-02-13T18:00:00Z","producer_version":"test","event_id":"evt_big","job_id":"job_jsonl_overflow","type":"big","executed":true,"payload":{"blob":"` +
			strings.Repeat("x", 70_000) +
			`"` + "\n",
	)
	if err := rewriteArchiveManifest(archive); err != nil {
		t.Fatalf("rewrite manifest: %v", err)
	}

	entries := make([]zipx.Entry, 0, len(archive.Files))
	for name, data := range archive.Files {
		entries = append(entries, zipx.Entry{Name: name, Data: data})
	}
	tamperedZip, err := zipx.BuildDeterministic(entries)
	if err != nil {
		t.Fatalf("build tampered zip: %v", err)
	}
	tamperedPath := filepath.Join(t.TempDir(), "tampered-jsonl-overflow.zip")
	if err := os.WriteFile(tamperedPath, tamperedZip, 0o600); err != nil {
		t.Fatalf("write tampered zip: %v", err)
	}

	_, err = VerifyJobpack(tamperedPath)
	if err == nil {
		t.Fatal("expected invalid jsonl verify failure")
	}
	var werr wrkrerrors.WrkrError
	if !errors.As(err, &werr) || werr.Code != wrkrerrors.EInvalidInputSchema {
		t.Fatalf("expected E_INVALID_INPUT_SCHEMA, got %v", err)
	}
}

func TestInspectAndDiffDeterministic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 18, 0, 0, 0, time.UTC)
	setupJob(t, "job_a", now)
	setupJob(t, "job_b", now)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	r, err := runner.New(s, runner.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	if _, err := r.EmitCheckpoint("job_b", runner.CheckpointInput{Type: "progress", Summary: "extra checkpoint"}); err != nil {
		t.Fatalf("emit extra checkpoint: %v", err)
	}

	outDir := t.TempDir()
	a, err := ExportJobpack("job_a", ExportOptions{OutDir: outDir, ProducerVersion: "test", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("export job_a: %v", err)
	}
	b, err := ExportJobpack("job_b", ExportOptions{OutDir: outDir, ProducerVersion: "test", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("export job_b: %v", err)
	}

	inspectA, err := InspectJobpack(a.Path)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if inspectA.JobID != "job_a" || inspectA.CheckpointCount == 0 {
		t.Fatalf("unexpected inspect result: %+v", inspectA)
	}

	diff, err := DiffJobpacks(a.Path, b.Path)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if len(diff.Changed) == 0 && len(diff.Added) == 0 && len(diff.Removed) == 0 {
		t.Fatalf("expected non-empty diff between jobpacks: %+v", diff)
	}
}

func TestExportPreservesExecutedFlagFromEventPayload(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 13, 18, 0, 0, 0, time.UTC)
	setupJob(t, "job_executed_flag", now)

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	if _, err := s.AppendEvent("job_executed_flag", "adapter_step", map[string]any{
		"step_id":  "review",
		"executed": false,
	}, now); err != nil {
		t.Fatalf("append event: %v", err)
	}

	exported, err := ExportJobpack("job_executed_flag", ExportOptions{
		OutDir:          t.TempDir(),
		ProducerVersion: "test",
		Now:             func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	archive, err := LoadArchive(exported.Path)
	if err != nil {
		t.Fatalf("load archive: %v", err)
	}
	events, err := DecodeEvents(archive.Files)
	if err != nil {
		t.Fatalf("decode events: %v", err)
	}
	found := false
	for _, event := range events {
		if event.Type == "adapter_step" && event.EventID != "" {
			found = true
			if event.Executed {
				t.Fatalf("expected executed=false for adapter_step event, got true: %+v", event)
			}
		}
	}
	if !found {
		t.Fatal("expected adapter_step event in exported jobpack")
	}
}

func rewriteArchiveManifest(archive *Archive) error {
	manifest := archive.Manifest
	files := make(map[string][]byte, len(archive.Files))
	for path, data := range archive.Files {
		if path == "manifest.json" {
			continue
		}
		files[path] = data
	}

	manifest.Files = SortedFileList(files)
	manifestSHA, err := ComputeManifestSHA256(manifest)
	if err != nil {
		return err
	}
	manifest.ManifestSHA256 = manifestSHA

	manifestBytes, err := EncodeJSONCanonical(manifest)
	if err != nil {
		return err
	}
	archive.Files["manifest.json"] = manifestBytes
	archive.Manifest = manifest
	return nil
}
