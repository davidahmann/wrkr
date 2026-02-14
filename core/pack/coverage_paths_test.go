package pack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/zipx"
)

func TestEncodeAndJSONLHelpersCoveragePaths(t *testing.T) {
	t.Parallel()

	if _, err := EncodeJSONCanonical(make(chan int)); err == nil {
		t.Fatal("expected marshal failure for unsupported type")
	}

	records := []map[string]any{
		{"a": 1},
		{"b": "two"},
	}
	raw, err := MarshalJSONLCanonical(records)
	if err != nil {
		t.Fatalf("MarshalJSONLCanonical: %v", err)
	}
	if !strings.Contains(string(raw), "\n") {
		t.Fatalf("expected jsonl newlines, got %q", string(raw))
	}
}

func TestDecodeHelpersCoveragePaths(t *testing.T) {
	t.Parallel()

	if _, err := DecodeJobRecord(map[string][]byte{}); err == nil {
		t.Fatal("expected missing job.json error")
	}
	if _, err := DecodeJobRecord(map[string][]byte{"job.json": []byte("{bad")}); err == nil {
		t.Fatal("expected invalid job.json decode error")
	}
	if _, err := DecodeCheckpoints(map[string][]byte{"checkpoints.jsonl": []byte("{bad}\n")}); err == nil {
		t.Fatal("expected invalid checkpoint jsonl decode error")
	}
	if _, err := DecodeEvents(map[string][]byte{"events.jsonl": []byte("{bad}\n")}); err == nil {
		t.Fatal("expected invalid event jsonl decode error")
	}
	if _, err := DecodeApprovals(map[string][]byte{"approvals.jsonl": []byte("{bad}\n")}); err == nil {
		t.Fatal("expected invalid approval jsonl decode error")
	}
}

func TestManifestAndFileHashHelpersCoveragePaths(t *testing.T) {
	t.Parallel()

	manifest := v1.JobpackManifest{
		Files: []v1.ManifestFile{
			{Path: "A.txt", SHA256: "ABCDEF"},
			{Path: "b.txt", SHA256: "1234"},
		},
	}
	hashes := fileHashes(manifest)
	if hashes["A.txt"] != "abcdef" {
		t.Fatalf("expected lowercase file hash, got %+v", hashes)
	}

	canonical, err := canonicalizeRawJSON([]byte(`{"b":2,"a":1}`))
	if err != nil {
		t.Fatalf("canonicalizeRawJSON: %v", err)
	}
	if string(canonical) != `{"a":1,"b":2}` {
		t.Fatalf("expected canonical json ordering, got %s", string(canonical))
	}
	if _, err := canonicalizeRawJSON([]byte("{bad")); err == nil {
		t.Fatal("expected canonicalizeRawJSON parse failure")
	}
}

func TestLoadArchiveCoveragePaths(t *testing.T) {
	t.Parallel()

	plain := filepath.Join(t.TempDir(), "not-a-zip.txt")
	if err := os.WriteFile(plain, []byte("plain"), 0o600); err != nil {
		t.Fatalf("write plain file: %v", err)
	}
	if _, err := LoadArchive(plain); err == nil {
		t.Fatal("expected non-zip load failure")
	}

	zipBytes, err := zipx.BuildDeterministic([]zipx.Entry{
		{Name: "job.json", Data: []byte(`{"job_id":"job_x"}`)},
	})
	if err != nil {
		t.Fatalf("build zip: %v", err)
	}
	missingManifest := filepath.Join(t.TempDir(), "missing-manifest.zip")
	if err := os.WriteFile(missingManifest, zipBytes, 0o600); err != nil {
		t.Fatalf("write zip: %v", err)
	}
	if _, err := LoadArchive(missingManifest); err == nil {
		t.Fatal("expected missing manifest failure")
	}
}

func TestVerifySchemaValidationCoveragePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 14, 0, 0, 0, time.UTC)
	acceptRaw, err := json.Marshal(v1.AcceptanceResult{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.accept_result",
			SchemaVersion:   "v1",
			CreatedAt:       now,
			ProducerVersion: "test",
		},
		JobID:       "job_schema",
		ChecksRun:   1,
		ChecksPassed: 1,
		Failures:    []v1.AcceptanceFailure{},
		ReasonCodes: []string{},
	})
	if err != nil {
		t.Fatalf("marshal accept result: %v", err)
	}
	files := map[string][]byte{
		"accept/accept_result.json": acceptRaw,
	}
	if err := validateSchemaFiles(files); err != nil {
		t.Fatalf("validateSchemaFiles valid accept result: %v", err)
	}

	files["accept/accept_result.json"] = []byte(`{"bad":true}`)
	if err := validateSchemaFiles(files); err == nil {
		t.Fatal("expected invalid accept schema failure")
	}

	files = map[string][]byte{
		"events.jsonl": []byte(`{"bad":true}` + "\n"),
	}
	if err := validateSchemaFiles(files); err == nil {
		t.Fatal("expected invalid event schema failure")
	}

	lines, err := jsonlLines([]byte("\n {\"x\":1} \n\n{\"y\":2}\n"))
	if err != nil {
		t.Fatalf("jsonlLines: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 jsonl lines, got %d", len(lines))
	}
}

func TestBuildArtifactsManifestCoveragePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 14, 14, 5, 0, 0, time.UTC)
	manifest := buildArtifactsManifest("job_artifacts", []v1.Checkpoint{
		{
			ArtifactsDelta: v1.ArtifactsDelta{
				Added:   []string{"reports/a.md"},
				Changed: []string{"reports/b.md"},
			},
		},
		{
			ArtifactsDelta: v1.ArtifactsDelta{
				Added: []string{"reports/a.md"},
			},
		},
	}, "test", now)

	if manifest.JobID != "job_artifacts" || manifest.CaptureMode != "reference-only" {
		t.Fatalf("unexpected manifest header: %+v", manifest)
	}
	if len(manifest.Artifacts) != 2 {
		t.Fatalf("expected deduped artifacts, got %+v", manifest.Artifacts)
	}
}

