package report

import (
	"strings"
	"testing"
	"time"

	v1 "github.com/davidahmann/wrkr/core/schema/v1"
)

func TestReportHelperCoveragePaths(t *testing.T) {
	t.Parallel()

	files := map[string][]byte{}
	accept, err := decodeAcceptResult(files)
	if err != nil {
		t.Fatalf("decodeAcceptResult missing: %v", err)
	}
	if accept.Failures == nil || accept.ReasonCodes == nil {
		t.Fatalf("expected default non-nil slices, got %+v", accept)
	}

	files["accept/accept_result.json"] = []byte(`{"failures":null,"reason_codes":null}`)
	accept, err = decodeAcceptResult(files)
	if err != nil {
		t.Fatalf("decodeAcceptResult explicit null slices: %v", err)
	}
	if accept.Failures == nil || accept.ReasonCodes == nil {
		t.Fatalf("expected normalized non-nil slices, got %+v", accept)
	}

	files["accept/accept_result.json"] = []byte("{bad")
	if _, err := decodeAcceptResult(files); err == nil {
		t.Fatal("expected decodeAcceptResult parse error")
	}

	files = map[string][]byte{"artifacts_manifest.json": []byte("{bad")}
	if _, err := decodeArtifactsManifest(files); err == nil {
		t.Fatal("expected decodeArtifactsManifest parse error")
	}
}

func TestArtifactDeltaAndMarkdownCoveragePaths(t *testing.T) {
	t.Parallel()

	checkpoints := []v1.Checkpoint{
		{
			CheckpointID: "cp_2",
			Envelope:     v1.Envelope{CreatedAt: time.Date(2026, 2, 14, 1, 0, 0, 0, time.UTC)},
			Summary:      " latest ",
			ArtifactsDelta: v1.ArtifactsDelta{
				Added:   []string{"a.md", " ", "a.md"},
				Changed: []string{"b.md"},
				Removed: []string{"c.md"},
			},
		},
		{
			CheckpointID: "cp_1",
			Envelope:     v1.Envelope{CreatedAt: time.Date(2026, 2, 14, 1, 0, 0, 0, time.UTC)},
			Summary:      "older",
		},
	}
	delta := collectArtifactDelta(checkpoints)
	if len(delta.added) != 1 || len(delta.changed) != 1 || len(delta.removed) != 1 {
		t.Fatalf("unexpected artifact delta: %+v", delta)
	}
	if summary := latestCheckpointSummary(checkpoints); summary != "latest" {
		t.Fatalf("expected trimmed latest summary, got %q", summary)
	}

	if summary := latestCheckpointSummary(nil); summary != "(none)" {
		t.Fatalf("expected empty fallback summary, got %q", summary)
	}

	if got := summaryCreatedAt(time.Time{}, time.Time{}, time.Date(2026, 2, 14, 1, 2, 0, 0, time.UTC)); got.IsZero() {
		t.Fatal("expected fallback summary created_at")
	}
	if _, ok := checkpointOrdinal("cp_12"); !ok {
		t.Fatal("expected cp_12 to parse")
	}
	if _, ok := checkpointOrdinal("bad"); ok {
		t.Fatal("expected invalid checkpoint id parse to fail")
	}

	pointers := extractArtifactPointers(&v1.ArtifactsManifest{
		Artifacts: []v1.ArtifactRecord{
			{Path: "z.md"},
			{Path: "a.md"},
			{Path: "m.md"},
			{Path: "k.md"},
			{Path: "x.md"},
			{Path: "b.md"},
			{Path: ""},
		},
	})
	if len(pointers) != 5 || pointers[0] != "a.md" {
		t.Fatalf("expected sorted pointer cap=5, got %v", pointers)
	}

	md := renderMarkdown(
		"job_report_cov",
		"completed",
		v1.AcceptanceResult{
			ChecksRun:    2,
			ChecksPassed: 1,
			Failures: []v1.AcceptanceFailure{
				{Check: "lint", Message: "failed", Artifact: "logs/lint.txt"},
			},
		},
		"",
		delta,
		pointers,
	)
	if !strings.Contains(md, "Top Failures") || !strings.Contains(md, "lint") {
		t.Fatalf("unexpected markdown: %s", md)
	}
}

func TestCanonicalAndStepSummaryCoveragePaths(t *testing.T) {
	t.Parallel()

	if _, err := canonicalJSON(make(chan int)); err == nil {
		t.Fatal("expected canonicalJSON marshal failure")
	}

	stepPath := t.TempDir() + "/step/summary.md"
	if err := appendStepSummary(stepPath, "summary body"); err != nil {
		t.Fatalf("appendStepSummary: %v", err)
	}
	if err := appendStepSummary(stepPath, "second line"); err != nil {
		t.Fatalf("appendStepSummary second write: %v", err)
	}
}

