package pack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/schema/validate"
	"github.com/davidahmann/wrkr/core/sign"
)

type VerifyResult struct {
	JobID          string `json:"job_id"`
	ManifestSHA256 string `json:"manifest_sha256"`
	FilesVerified  int    `json:"files_verified"`
}

func VerifyJobpack(path string) (VerifyResult, error) {
	archive, err := LoadArchive(path)
	if err != nil {
		return VerifyResult{}, err
	}

	manifestRaw, ok := archive.Files["manifest.json"]
	if !ok {
		return VerifyResult{}, wrkrerrors.New(wrkrerrors.EVerifyHashMismatch, "manifest.json missing", nil)
	}
	if err := validate.ValidateBytes(validate.JobpackManifestSchemaRel, manifestRaw); err != nil {
		return VerifyResult{}, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "manifest schema invalid", map[string]any{"error": err.Error()})
	}

	expectedManifestHash, err := ComputeManifestSHA256(archive.Manifest)
	if err != nil {
		return VerifyResult{}, err
	}
	if archive.Manifest.ManifestSHA256 != expectedManifestHash {
		return VerifyResult{}, wrkrerrors.New(
			wrkrerrors.EVerifyHashMismatch,
			"manifest_sha256 mismatch",
			map[string]any{"expected": expectedManifestHash, "actual": archive.Manifest.ManifestSHA256},
		)
	}

	for _, file := range archive.Manifest.Files {
		data, ok := archive.Files[file.Path]
		if !ok {
			return VerifyResult{}, wrkrerrors.New(
				wrkrerrors.EVerifyHashMismatch,
				"manifest references missing file",
				map[string]any{"path": file.Path},
			)
		}
		actual := sign.SHA256Hex(data)
		if actual != file.SHA256 {
			return VerifyResult{}, wrkrerrors.New(
				wrkrerrors.EVerifyHashMismatch,
				"file hash mismatch",
				map[string]any{"path": file.Path, "expected": file.SHA256, "actual": actual},
			)
		}
	}

	allowedFiles := map[string]struct{}{"manifest.json": {}}
	for _, file := range archive.Manifest.Files {
		allowedFiles[file.Path] = struct{}{}
	}
	for path := range archive.Files {
		if _, ok := allowedFiles[path]; !ok {
			return VerifyResult{}, wrkrerrors.New(
				wrkrerrors.EVerifyHashMismatch,
				"archive contains undeclared file",
				map[string]any{"path": path},
			)
		}
	}

	if err := validateSchemaFiles(archive.Files); err != nil {
		return VerifyResult{}, err
	}

	return VerifyResult{
		JobID:          archive.Manifest.JobID,
		ManifestSHA256: archive.Manifest.ManifestSHA256,
		FilesVerified:  len(archive.Manifest.Files),
	}, nil
}

func validateSchemaFiles(files map[string][]byte) error {
	if raw, ok := files["job.json"]; ok {
		if err := validate.ValidateBytes(validate.JobSchemaRel, raw); err != nil {
			return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "job schema invalid", map[string]any{"error": err.Error()})
		}
	}
	if raw, ok := files["artifacts_manifest.json"]; ok {
		if err := validate.ValidateBytes(validate.ArtifactsManifestSchemaRel, raw); err != nil {
			return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "artifacts_manifest schema invalid", map[string]any{"error": err.Error()})
		}
	}
	if raw, ok := files["accept/accept_result.json"]; ok {
		if err := validate.ValidateBytes(validate.AcceptResultSchemaRel, raw); err != nil {
			return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "accept_result schema invalid", map[string]any{"error": err.Error()})
		}
	}
	if raw, ok := files["events.jsonl"]; ok {
		lines, err := jsonlLines(raw)
		if err != nil {
			return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "events jsonl parse failed", map[string]any{"error": err.Error()})
		}
		for i, line := range lines {
			if err := validate.ValidateBytes(validate.EventSchemaRel, line); err != nil {
				return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "event schema invalid", map[string]any{"line": i + 1, "error": err.Error()})
			}
		}
	}
	if raw, ok := files["checkpoints.jsonl"]; ok {
		lines, err := jsonlLines(raw)
		if err != nil {
			return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "checkpoints jsonl parse failed", map[string]any{"error": err.Error()})
		}
		for i, line := range lines {
			if err := validate.ValidateBytes(validate.CheckpointSchemaRel, line); err != nil {
				return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "checkpoint schema invalid", map[string]any{"line": i + 1, "error": err.Error()})
			}
		}
	}
	if raw, ok := files["approvals.jsonl"]; ok {
		lines, err := jsonlLines(raw)
		if err != nil {
			return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "approvals jsonl parse failed", map[string]any{"error": err.Error()})
		}
		for i, line := range lines {
			if err := validate.ValidateBytes(validate.ApprovalRecordSchemaRel, line); err != nil {
				return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "approval schema invalid", map[string]any{"line": i + 1, "error": err.Error()})
			}
		}
	}
	return nil
}

func jsonlLines(raw []byte) ([][]byte, error) {
	parts := bytes.Split(raw, []byte{'\n'})
	out := make([][]byte, 0, 8)
	for _, part := range parts {
		line := bytes.TrimSpace(part)
		if len(line) == 0 {
			continue
		}
		copied := make([]byte, len(line))
		copy(copied, line)
		out = append(out, copied)
	}
	return out, nil
}

func DecodeJobRecord(files map[string][]byte) (*v1.JobRecord, error) {
	raw, ok := files["job.json"]
	if !ok {
		return nil, fmt.Errorf("job.json missing")
	}
	var record v1.JobRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func DecodeCheckpoints(files map[string][]byte) ([]v1.Checkpoint, error) {
	raw, ok := files["checkpoints.jsonl"]
	if !ok {
		return nil, nil
	}
	lines, err := jsonlLines(raw)
	if err != nil {
		return nil, err
	}
	out := make([]v1.Checkpoint, 0, 8)
	for i, line := range lines {
		var cp v1.Checkpoint
		if err := json.Unmarshal(line, &cp); err != nil {
			return nil, fmt.Errorf("decode checkpoint line %d: %w", i+1, err)
		}
		out = append(out, cp)
	}
	return out, nil
}

func DecodeEvents(files map[string][]byte) ([]v1.EventRecord, error) {
	raw, ok := files["events.jsonl"]
	if !ok {
		return nil, nil
	}
	lines, err := jsonlLines(raw)
	if err != nil {
		return nil, err
	}
	out := make([]v1.EventRecord, 0, 16)
	for i, line := range lines {
		var evt v1.EventRecord
		if err := json.Unmarshal(line, &evt); err != nil {
			return nil, fmt.Errorf("decode event line %d: %w", i+1, err)
		}
		out = append(out, evt)
	}
	return out, nil
}

func DecodeApprovals(files map[string][]byte) ([]v1.ApprovalRecord, error) {
	raw, ok := files["approvals.jsonl"]
	if !ok {
		return nil, nil
	}
	lines, err := jsonlLines(raw)
	if err != nil {
		return nil, err
	}
	out := make([]v1.ApprovalRecord, 0, 8)
	for i, line := range lines {
		var rec v1.ApprovalRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("decode approval line %d: %w", i+1, err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func fileHashes(manifest v1.JobpackManifest) map[string]string {
	out := make(map[string]string, len(manifest.Files))
	for _, file := range manifest.Files {
		out[file.Path] = strings.ToLower(file.SHA256)
	}
	return out
}
