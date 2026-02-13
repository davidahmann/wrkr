package bridge

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/davidahmann/wrkr/core/fsx"
	"github.com/davidahmann/wrkr/core/jcs"
	"github.com/davidahmann/wrkr/core/out"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/schema/validate"
)

type BuildOptions struct {
	Now             func() time.Time
	ProducerVersion string
}

type WriteResult struct {
	JSONPath     string `json:"json_path,omitempty"`
	TemplatePath string `json:"template_path,omitempty"`
}

func BuildWorkItemPayload(jobID string, checkpoint v1.Checkpoint, opts BuildOptions) (v1.WorkItemPayload, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	producerVersion := strings.TrimSpace(opts.ProducerVersion)
	if producerVersion == "" {
		producerVersion = "dev"
	}

	payload := v1.WorkItemPayload{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.work_item",
			SchemaVersion:   "v1",
			CreatedAt:       checkpoint.CreatedAt.UTC(),
			ProducerVersion: producerVersion,
		},
		JobID:            jobID,
		CheckpointID:     checkpoint.CheckpointID,
		CheckpointType:   checkpoint.Type,
		RequiredAction:   requiredAction(checkpoint),
		ReasonCodes:      normalizedReasonCodes(checkpoint.ReasonCodes),
		ArtifactPointers: artifactPointers(checkpoint),
		NextCommands:     nextCommands(jobID, checkpoint),
	}
	if payload.CreatedAt.IsZero() {
		payload.CreatedAt = now().UTC()
	}

	raw, err := marshalCanonical(payload)
	if err != nil {
		return v1.WorkItemPayload{}, err
	}
	if err := validate.ValidateBytes(validate.WorkItemSchemaRel, raw); err != nil {
		return v1.WorkItemPayload{}, fmt.Errorf("work-item schema invalid: %w", err)
	}

	return payload, nil
}

func WriteWorkItemPayload(payload v1.WorkItemPayload, outDir, template string) (WriteResult, error) {
	layout := out.NewLayout(outDir)
	if err := layout.Ensure(); err != nil {
		return WriteResult{}, err
	}

	fileBase := sanitize(payload.JobID) + "_" + sanitize(payload.CheckpointID)
	jsonPath := layout.ReportPath(fmt.Sprintf("work_item_%s.json", fileBase))
	raw, err := marshalCanonical(payload)
	if err != nil {
		return WriteResult{}, err
	}
	if err := fsx.AtomicWriteFile(jsonPath, raw, 0o600); err != nil {
		return WriteResult{}, err
	}

	result := WriteResult{JSONPath: jsonPath}
	template = strings.TrimSpace(strings.ToLower(template))
	if template != "" {
		templatePath := layout.ReportPath(fmt.Sprintf("work_item_%s_%s.md", fileBase, template))
		body := renderTemplate(payload, template)
		if err := fsx.AtomicWriteFile(templatePath, []byte(body+"\n"), 0o600); err != nil {
			return WriteResult{}, err
		}
		result.TemplatePath = templatePath
	}
	return result, nil
}

func requiredAction(checkpoint v1.Checkpoint) string {
	if checkpoint.RequiredAction != nil {
		if value := strings.TrimSpace(checkpoint.RequiredAction.Kind); value != "" {
			return value
		}
		if value := strings.TrimSpace(checkpoint.RequiredAction.Instructions); value != "" {
			return value
		}
	}
	if checkpoint.Type == "decision-needed" {
		return "approval"
	}
	return "resume"
}

func normalizedReasonCodes(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

func artifactPointers(checkpoint v1.Checkpoint) []string {
	all := make([]string, 0, len(checkpoint.ArtifactsDelta.Added)+len(checkpoint.ArtifactsDelta.Changed)+len(checkpoint.ArtifactsDelta.Removed))
	all = append(all, checkpoint.ArtifactsDelta.Added...)
	all = append(all, checkpoint.ArtifactsDelta.Changed...)
	all = append(all, checkpoint.ArtifactsDelta.Removed...)
	seen := make(map[string]struct{}, len(all))
	out := make([]string, 0, len(all))
	for _, item := range all {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func nextCommands(jobID string, checkpoint v1.Checkpoint) []string {
	jobID = strings.TrimSpace(jobID)
	commands := make([]string, 0, 4)
	if checkpoint.Type == "decision-needed" {
		commands = append(commands, fmt.Sprintf("wrkr approve %s --checkpoint %s --reason \"approved\"", jobID, checkpoint.CheckpointID))
	}
	commands = append(commands, fmt.Sprintf("wrkr resume %s", jobID))
	commands = append(commands, fmt.Sprintf("wrkr export %s", jobID))
	commands = append(commands, fmt.Sprintf("wrkr accept run %s --json", jobID))
	return commands
}

func renderTemplate(payload v1.WorkItemPayload, template string) string {
	switch template {
	case "jira":
		return strings.TrimSpace(fmt.Sprintf(
			"# JIRA Work Item\n\nSummary: `%s/%s` requires `%s`\n\nReason codes: %s\n\nNext commands:\n%s\n",
			payload.JobID,
			payload.CheckpointID,
			payload.RequiredAction,
			strings.Join(payload.ReasonCodes, ", "),
			bulletList(payload.NextCommands),
		))
	default:
		return strings.TrimSpace(fmt.Sprintf(
			"# GitHub Work Item\n\nJob `%s` checkpoint `%s` (`%s`) requires `%s`.\n\nReason codes: %s\n\nNext commands:\n%s\n",
			payload.JobID,
			payload.CheckpointID,
			payload.CheckpointType,
			payload.RequiredAction,
			strings.Join(payload.ReasonCodes, ", "),
			bulletList(payload.NextCommands),
		))
	}
}

func bulletList(items []string) string {
	if len(items) == 0 {
		return "- (none)"
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, "- "+item)
	}
	return strings.Join(lines, "\n")
}

func marshalCanonical(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	canonical, err := jcs.Canonicalize(raw)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}

func sanitize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "value"
	}
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "..", "_")
	return value
}
