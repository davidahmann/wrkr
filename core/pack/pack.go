package pack

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/davidahmann/wrkr/core/fsx"
	"github.com/davidahmann/wrkr/core/out"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/sign"
	"github.com/davidahmann/wrkr/core/store"
	"github.com/davidahmann/wrkr/core/zipx"
)

type ExportOptions struct {
	OutDir          string
	Now             func() time.Time
	ProducerVersion string
}

type ExportResult struct {
	JobID          string `json:"job_id"`
	Path           string `json:"path"`
	ManifestSHA256 string `json:"manifest_sha256"`
	Footer         string `json:"footer"`
}

func ExportJobpack(jobID string, opts ExportOptions) (ExportResult, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	producerVersion := opts.ProducerVersion
	if producerVersion == "" {
		producerVersion = "dev"
	}

	s, err := store.New("")
	if err != nil {
		return ExportResult{}, err
	}
	exists, err := s.JobExists(jobID)
	if err != nil {
		return ExportResult{}, err
	}
	if !exists {
		return ExportResult{}, fmt.Errorf("job not found: %s", jobID)
	}
	r, err := runner.New(s, runner.Options{Now: now})
	if err != nil {
		return ExportResult{}, err
	}

	state, err := r.Recover(jobID)
	if err != nil {
		return ExportResult{}, err
	}
	events, err := s.LoadEvents(jobID)
	if err != nil {
		return ExportResult{}, err
	}
	checkpoints, err := r.ListCheckpoints(jobID)
	if err != nil {
		return ExportResult{}, err
	}
	approvals, err := r.ListApprovals(jobID)
	if err != nil {
		return ExportResult{}, err
	}

	files := map[string][]byte{}

	jobRecord := v1.JobRecord{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.job",
			SchemaVersion:   "v1",
			CreatedAt:       now().UTC(),
			ProducerVersion: producerVersion,
		},
		JobID:  state.JobID,
		Name:   state.JobID,
		Status: string(state.Status),
		Budgets: map[string]any{
			"retry_count":     state.RetryCount,
			"step_count":      state.StepCount,
			"tool_call_count": state.ToolCallCount,
		},
	}
	jobBytes, err := EncodeJSONCanonical(jobRecord)
	if err != nil {
		return ExportResult{}, err
	}
	files["job.json"] = jobBytes

	projectedEvents := make([]v1.EventRecord, 0, len(events))
	for _, event := range events {
		payload := map[string]any{}
		executed := true
		if len(event.Payload) > 0 {
			var decoded any
			if err := json.Unmarshal(event.Payload, &decoded); err == nil {
				if asMap, ok := decoded.(map[string]any); ok {
					payload = asMap
					if explicitExecuted, ok := asMap["executed"].(bool); ok {
						executed = explicitExecuted
					}
				} else {
					payload = map[string]any{"value": decoded}
				}
			}
		}
		projectedEvents = append(projectedEvents, v1.EventRecord{
			Envelope: v1.Envelope{
				SchemaID:        "wrkr.event",
				SchemaVersion:   "v1",
				CreatedAt:       event.CreatedAt.UTC(),
				ProducerVersion: producerVersion,
			},
			EventID:  fmt.Sprintf("evt_%d", event.Seq),
			JobID:    jobID,
			Type:     event.Type,
			Executed: executed,
			Payload:  payload,
		})
	}
	eventBytes, err := MarshalJSONLCanonical(projectedEvents)
	if err != nil {
		return ExportResult{}, err
	}
	files["events.jsonl"] = eventBytes

	checkpointBytes, err := MarshalJSONLCanonical(checkpoints)
	if err != nil {
		return ExportResult{}, err
	}
	files["checkpoints.jsonl"] = checkpointBytes

	artifactsManifest := buildArtifactsManifest(jobID, checkpoints, producerVersion, now())
	artifactBytes, err := EncodeJSONCanonical(artifactsManifest)
	if err != nil {
		return ExportResult{}, err
	}
	files["artifacts_manifest.json"] = artifactBytes

	if len(approvals) > 0 {
		approvalBytes, err := MarshalJSONLCanonical(approvals)
		if err != nil {
			return ExportResult{}, err
		}
		files["approvals.jsonl"] = approvalBytes
	}

	acceptPath := s.JobDir(jobID) + "/accept_result.json"
	// #nosec G304 -- acceptPath is derived from store job dir with validated job_id.
	acceptBytes, err := os.ReadFile(acceptPath)
	if err == nil {
		canonicalAccept, err := canonicalizeRawJSON(acceptBytes)
		if err != nil {
			return ExportResult{}, err
		}
		files["accept/accept_result.json"] = canonicalAccept
	} else if !os.IsNotExist(err) {
		return ExportResult{}, err
	}

	fileList := SortedFileList(files)
	manifest := v1.JobpackManifest{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.jobpack_manifest",
			SchemaVersion:   "v1",
			CreatedAt:       now().UTC(),
			ProducerVersion: producerVersion,
		},
		JobID: jobID,
		Files: fileList,
	}
	manifestHash, err := ComputeManifestSHA256(manifest)
	if err != nil {
		return ExportResult{}, err
	}
	manifest.ManifestSHA256 = manifestHash
	manifestBytes, err := EncodeJSONCanonical(manifest)
	if err != nil {
		return ExportResult{}, err
	}
	files["manifest.json"] = manifestBytes

	entries := make([]zipx.Entry, 0, len(files))
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		entries = append(entries, zipx.Entry{Name: path, Data: files[path]})
	}
	zipBytes, err := zipx.BuildDeterministic(entries)
	if err != nil {
		return ExportResult{}, err
	}

	layout, err := out.NewLayout(opts.OutDir)
	if err != nil {
		return ExportResult{}, err
	}
	if err := layout.Ensure(); err != nil {
		return ExportResult{}, err
	}
	path := layout.JobpackPath(jobID)
	if err := fsx.AtomicWriteFile(path, zipBytes, 0o600); err != nil {
		return ExportResult{}, err
	}

	return ExportResult{
		JobID:          jobID,
		Path:           path,
		ManifestSHA256: manifest.ManifestSHA256,
		Footer:         TicketFooter(jobID, manifest.ManifestSHA256),
	}, nil
}

func buildArtifactsManifest(jobID string, checkpoints []v1.Checkpoint, producerVersion string, now time.Time) v1.ArtifactsManifest {
	seen := map[string]struct{}{}
	for _, cp := range checkpoints {
		for _, path := range cp.ArtifactsDelta.Added {
			seen[path] = struct{}{}
		}
		for _, path := range cp.ArtifactsDelta.Changed {
			seen[path] = struct{}{}
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	artifacts := make([]v1.ArtifactRecord, 0, len(paths))
	for _, path := range paths {
		sha := sign.SHA256Hex([]byte("reference:" + path))
		artifacts = append(artifacts, v1.ArtifactRecord{
			Path:   path,
			SHA256: sha,
		})
	}

	return v1.ArtifactsManifest{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.artifacts_manifest",
			SchemaVersion:   "v1",
			CreatedAt:       now.UTC(),
			ProducerVersion: producerVersion,
		},
		JobID:       jobID,
		CaptureMode: "reference-only",
		Artifacts:   artifacts,
	}
}

func canonicalizeRawJSON(raw []byte) ([]byte, error) {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	return EncodeJSONCanonical(data)
}
