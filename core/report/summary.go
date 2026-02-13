package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/davidahmann/wrkr/core/fsx"
	"github.com/davidahmann/wrkr/core/jcs"
	"github.com/davidahmann/wrkr/core/out"
	"github.com/davidahmann/wrkr/core/pack"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/schema/validate"
)

type SummaryOptions struct {
	Now             func() time.Time
	ProducerVersion string
}

type WriteResult struct {
	JSONPath        string `json:"json_path"`
	MarkdownPath    string `json:"markdown_path"`
	StepSummaryPath string `json:"step_summary_path,omitempty"`
}

type artifactDelta struct {
	added   map[string]struct{}
	changed map[string]struct{}
	removed map[string]struct{}
}

func BuildGitHubSummaryFromJobpack(path string, opts SummaryOptions) (v1.GitHubSummary, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	producerVersion := opts.ProducerVersion
	if producerVersion == "" {
		producerVersion = "dev"
	}

	archive, err := pack.LoadArchive(path)
	if err != nil {
		return v1.GitHubSummary{}, err
	}
	job, err := pack.DecodeJobRecord(archive.Files)
	if err != nil {
		return v1.GitHubSummary{}, err
	}
	checkpoints, err := pack.DecodeCheckpoints(archive.Files)
	if err != nil {
		return v1.GitHubSummary{}, err
	}
	acceptResult, err := decodeAcceptResult(archive.Files)
	if err != nil {
		return v1.GitHubSummary{}, err
	}
	artifactsManifest, err := decodeArtifactsManifest(archive.Files)
	if err != nil {
		return v1.GitHubSummary{}, err
	}

	delta := collectArtifactDelta(checkpoints)
	finalSummary := latestCheckpointSummary(checkpoints)
	artifactPointers := extractArtifactPointers(artifactsManifest)
	markdown := renderMarkdown(job.JobID, job.Status, acceptResult, finalSummary, delta, artifactPointers)
	createdAt := summaryCreatedAt(archive.Manifest.CreatedAt, job.CreatedAt, now().UTC())

	summary := v1.GitHubSummary{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.github_summary",
			SchemaVersion:   "v1",
			CreatedAt:       createdAt,
			ProducerVersion: producerVersion,
		},
		JobID:  job.JobID,
		Status: job.Status,
		Acceptance: v1.GitHubSummaryAcceptance{
			ChecksRun:    acceptResult.ChecksRun,
			ChecksPassed: acceptResult.ChecksPassed,
			Failed:       len(acceptResult.Failures) > 0,
		},
		ArtifactDelta: v1.GitHubSummaryArtifactDelta{
			Added:   len(delta.added),
			Changed: len(delta.changed),
			Removed: len(delta.removed),
		},
		Markdown: markdown,
	}

	raw, err := json.Marshal(summary)
	if err != nil {
		return v1.GitHubSummary{}, fmt.Errorf("marshal summary: %w", err)
	}
	if err := validate.ValidateBytes(validate.GitHubSummarySchemaRel, raw); err != nil {
		return v1.GitHubSummary{}, fmt.Errorf("summary schema invalid: %w", err)
	}
	return summary, nil
}

func WriteGitHubSummary(summary v1.GitHubSummary, outDir string) (WriteResult, error) {
	layout, err := out.NewLayout(outDir)
	if err != nil {
		return WriteResult{}, err
	}
	if err := layout.Ensure(); err != nil {
		return WriteResult{}, err
	}

	jsonPath := layout.ReportPath(fmt.Sprintf("github_summary_%s.json", summary.JobID))
	mdPath := layout.ReportPath(fmt.Sprintf("github_summary_%s.md", summary.JobID))

	raw, err := canonicalJSON(summary)
	if err != nil {
		return WriteResult{}, err
	}
	if err := fsx.AtomicWriteFile(jsonPath, raw, 0o600); err != nil {
		return WriteResult{}, fmt.Errorf("write summary json: %w", err)
	}
	if err := fsx.AtomicWriteFile(mdPath, []byte(summary.Markdown+"\n"), 0o600); err != nil {
		return WriteResult{}, fmt.Errorf("write summary markdown: %w", err)
	}

	result := WriteResult{JSONPath: jsonPath, MarkdownPath: mdPath}
	if stepSummary := strings.TrimSpace(os.Getenv("GITHUB_STEP_SUMMARY")); stepSummary != "" {
		if err := appendStepSummary(stepSummary, summary.Markdown); err != nil {
			return WriteResult{}, err
		}
		result.StepSummaryPath = stepSummary
	}

	return result, nil
}

func decodeAcceptResult(files map[string][]byte) (v1.AcceptanceResult, error) {
	raw, ok := files["accept/accept_result.json"]
	if !ok {
		return v1.AcceptanceResult{
			Failures:    []v1.AcceptanceFailure{},
			ReasonCodes: []string{},
		}, nil
	}

	var result v1.AcceptanceResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return v1.AcceptanceResult{}, fmt.Errorf("decode accept_result: %w", err)
	}
	if result.Failures == nil {
		result.Failures = []v1.AcceptanceFailure{}
	}
	if result.ReasonCodes == nil {
		result.ReasonCodes = []string{}
	}
	return result, nil
}

func decodeArtifactsManifest(files map[string][]byte) (*v1.ArtifactsManifest, error) {
	raw, ok := files["artifacts_manifest.json"]
	if !ok {
		return nil, nil
	}
	var manifest v1.ArtifactsManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("decode artifacts manifest: %w", err)
	}
	return &manifest, nil
}

func collectArtifactDelta(checkpoints []v1.Checkpoint) artifactDelta {
	out := artifactDelta{
		added:   make(map[string]struct{}, 16),
		changed: make(map[string]struct{}, 16),
		removed: make(map[string]struct{}, 16),
	}
	for _, cp := range checkpoints {
		for _, path := range cp.ArtifactsDelta.Added {
			if strings.TrimSpace(path) != "" {
				out.added[path] = struct{}{}
			}
		}
		for _, path := range cp.ArtifactsDelta.Changed {
			if strings.TrimSpace(path) != "" {
				out.changed[path] = struct{}{}
			}
		}
		for _, path := range cp.ArtifactsDelta.Removed {
			if strings.TrimSpace(path) != "" {
				out.removed[path] = struct{}{}
			}
		}
	}
	return out
}

func latestCheckpointSummary(checkpoints []v1.Checkpoint) string {
	if len(checkpoints) == 0 {
		return "(none)"
	}
	latest := checkpoints[0]
	for _, cp := range checkpoints[1:] {
		if cp.CreatedAt.After(latest.CreatedAt) {
			latest = cp
			continue
		}
		if cp.CreatedAt.Equal(latest.CreatedAt) {
			cpOrdinal, cpOK := checkpointOrdinal(cp.CheckpointID)
			latestOrdinal, latestOK := checkpointOrdinal(latest.CheckpointID)
			switch {
			case cpOK && latestOK && cpOrdinal > latestOrdinal:
				latest = cp
			case cpOK && !latestOK:
				latest = cp
			case !cpOK && !latestOK && cp.CheckpointID > latest.CheckpointID:
				latest = cp
			}
		}
	}
	return strings.TrimSpace(latest.Summary)
}

func summaryCreatedAt(manifestCreatedAt, jobCreatedAt, fallback time.Time) time.Time {
	if !manifestCreatedAt.IsZero() {
		return manifestCreatedAt.UTC()
	}
	if !jobCreatedAt.IsZero() {
		return jobCreatedAt.UTC()
	}
	return fallback.UTC()
}

func checkpointOrdinal(id string) (int64, bool) {
	if !strings.HasPrefix(id, "cp_") {
		return 0, false
	}
	value, err := strconv.ParseInt(strings.TrimPrefix(id, "cp_"), 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func extractArtifactPointers(manifest *v1.ArtifactsManifest) []string {
	if manifest == nil || len(manifest.Artifacts) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		path := strings.TrimSpace(artifact.Path)
		if path == "" {
			continue
		}
		out = append(out, path)
	}
	sort.Strings(out)
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func renderMarkdown(jobID, status string, acceptResult v1.AcceptanceResult, finalCheckpoint string, delta artifactDelta, artifactPointers []string) string {
	if strings.TrimSpace(finalCheckpoint) == "" {
		finalCheckpoint = "(none)"
	}
	failed := len(acceptResult.Failures) > 0
	statusLine := "passed"
	if failed {
		statusLine = "failed"
	}

	var b strings.Builder
	b.WriteString("# Wrkr GitHub Summary\n\n")
	b.WriteString("- Job: `" + jobID + "`\n")
	b.WriteString("- Status: `" + status + "`\n")
	b.WriteString(fmt.Sprintf("- Acceptance: `%s` (%d/%d checks)\n", statusLine, acceptResult.ChecksPassed, acceptResult.ChecksRun))
	b.WriteString(fmt.Sprintf("- Artifact Delta: added=%d changed=%d removed=%d\n\n", len(delta.added), len(delta.changed), len(delta.removed)))

	b.WriteString("## Final Checkpoint\n\n")
	b.WriteString(finalCheckpoint + "\n\n")

	b.WriteString("## Top Failures\n\n")
	if len(acceptResult.Failures) == 0 {
		b.WriteString("- None\n\n")
	} else {
		limit := len(acceptResult.Failures)
		if limit > 5 {
			limit = 5
		}
		for _, failure := range acceptResult.Failures[:limit] {
			line := "- `" + failure.Check + "`: " + failure.Message
			if strings.TrimSpace(failure.Artifact) != "" {
				line += " (`" + failure.Artifact + "`)"
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Artifact Pointers\n\n")
	if len(artifactPointers) == 0 {
		b.WriteString("- None\n")
	} else {
		for _, pointer := range artifactPointers {
			b.WriteString("- `" + pointer + "`\n")
		}
	}

	return strings.TrimSpace(b.String())
}

func canonicalJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	canonical, err := jcs.Canonicalize(raw)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}

func appendStepSummary(path, markdown string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create step summary dir: %w", err)
	}
	// #nosec G304 -- path comes from GitHub-provided GITHUB_STEP_SUMMARY env var in CI context.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open step summary: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.WriteString(markdown + "\n"); err != nil {
		return fmt.Errorf("write step summary: %w", err)
	}
	return nil
}
