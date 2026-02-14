package pack

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/store"
	"github.com/davidahmann/wrkr/core/zipx"
)

func TestExportJobpackCoverageMore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 19, 20, 0, 0, time.UTC)

	if _, err := ExportJobpack("job_missing_pack_cov", ExportOptions{OutDir: t.TempDir(), Now: func() time.Time { return now }, ProducerVersion: "test"}); err == nil {
		t.Fatal("expected missing job export failure")
	}

	setupJob(t, "job_pack_bad_accept", now)
	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.JobDir("job_pack_bad_accept"), "accept_result.json"), []byte("{bad"), 0o600); err != nil {
		t.Fatalf("write invalid accept_result: %v", err)
	}
	if _, err := ExportJobpack("job_pack_bad_accept", ExportOptions{OutDir: t.TempDir(), Now: func() time.Time { return now }, ProducerVersion: "test"}); err == nil {
		t.Fatal("expected invalid accept_result canonicalization failure")
	}
}

func TestVerifyJobpackCoverageMore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 19, 30, 0, 0, time.UTC)
	setupJob(t, "job_pack_missing_file", now)

	exported, err := ExportJobpack("job_pack_missing_file", ExportOptions{
		OutDir:          t.TempDir(),
		ProducerVersion: "test",
		Now:             func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("ExportJobpack: %v", err)
	}

	archive, err := LoadArchive(exported.Path)
	if err != nil {
		t.Fatalf("LoadArchive: %v", err)
	}
	archive.Manifest.Files = append(archive.Manifest.Files, v1.ManifestFile{
		Path:   "missing.txt",
		SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	manifestHash, err := ComputeManifestSHA256(archive.Manifest)
	if err != nil {
		t.Fatalf("ComputeManifestSHA256: %v", err)
	}
	archive.Manifest.ManifestSHA256 = manifestHash
	manifestRaw, err := EncodeJSONCanonical(archive.Manifest)
	if err != nil {
		t.Fatalf("EncodeJSONCanonical manifest: %v", err)
	}
	archive.Files["manifest.json"] = manifestRaw

	entries := make([]zipx.Entry, 0, len(archive.Files))
	for name, data := range archive.Files {
		entries = append(entries, zipx.Entry{Name: name, Data: data})
	}
	raw, err := zipx.BuildDeterministic(entries)
	if err != nil {
		t.Fatalf("BuildDeterministic: %v", err)
	}
	path := filepath.Join(t.TempDir(), "missing-file-manifest.zip")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	if _, err := VerifyJobpack(path); err == nil {
		t.Fatal("expected verify failure for manifest referencing missing file")
	}
}

func TestValidateSchemaFilesCoverageMore(t *testing.T) {
	t.Parallel()

	files := map[string][]byte{
		"checkpoints.jsonl": []byte("{bad}\n"),
	}
	if err := validateSchemaFiles(files); err == nil {
		t.Fatal("expected checkpoints parse/schema failure")
	}

	files = map[string][]byte{
		"approvals.jsonl": []byte("{bad}\n"),
	}
	if err := validateSchemaFiles(files); err == nil {
		t.Fatal("expected approvals parse/schema failure")
	}
}

