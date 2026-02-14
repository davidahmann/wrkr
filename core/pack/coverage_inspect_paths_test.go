package pack

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/zipx"
)

func TestInspectJobpackCoveragePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 22, 30, 0, 0, time.UTC)
	setupJob(t, "job_pack_inspect_cov", now)

	exported, err := ExportJobpack("job_pack_inspect_cov", ExportOptions{
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

	archive.Files["approvals.jsonl"] = []byte("{bad}\n")
	if err := rewriteArchiveManifest(archive); err != nil {
		t.Fatalf("rewriteArchiveManifest: %v", err)
	}
	entries := make([]zipx.Entry, 0, len(archive.Files))
	for name, data := range archive.Files {
		entries = append(entries, zipx.Entry{Name: name, Data: data})
	}
	raw, err := zipx.BuildDeterministic(entries)
	if err != nil {
		t.Fatalf("BuildDeterministic: %v", err)
	}
	path := filepath.Join(t.TempDir(), "inspect-bad-approvals.zip")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write inspect fixture zip: %v", err)
	}

	if _, err := InspectJobpack(path); err == nil {
		t.Fatal("expected inspect approval decode failure")
	}
}

