package checks

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/schema/validate"
)

type PathRules struct {
	MaxArtifactPaths  int
	ForbiddenPrefixes []string
	AllowedPrefixes   []string
}

type Config struct {
	RequiredArtifacts []string
	TestCommand       string
	LintCommand       string
	PathRules         PathRules
}

type Input struct {
	StatusResponse v1.StatusResponse
	Checkpoints    []v1.Checkpoint
	Approvals      []v1.ApprovalRecord
	WorkDir        string
}

type CheckResult struct {
	Name       string          `json:"name"`
	Passed     bool            `json:"passed"`
	Message    string          `json:"message"`
	ReasonCode wrkrerrors.Code `json:"reason_code,omitempty"`
	Artifact   string          `json:"artifact,omitempty"`
}

func Run(cfg Config, in Input) ([]CheckResult, error) {
	results := make([]CheckResult, 0, 5)

	schemaResult, err := checkSchemaValidity(in)
	if err != nil {
		return nil, err
	}
	results = append(results, schemaResult)
	results = append(results, checkRequiredArtifacts(cfg, in.Checkpoints))
	results = append(results, checkPathConstraints(cfg.PathRules, in.Checkpoints))

	testResult, err := runCommandCheck("test_command", strings.TrimSpace(cfg.TestCommand), strings.TrimSpace(in.WorkDir))
	if err != nil {
		return nil, err
	}
	results = append(results, testResult)

	lintResult, err := runCommandCheck("lint_command", strings.TrimSpace(cfg.LintCommand), strings.TrimSpace(in.WorkDir))
	if err != nil {
		return nil, err
	}
	results = append(results, lintResult)

	return results, nil
}

func checkSchemaValidity(in Input) (CheckResult, error) {
	statusRaw, err := json.Marshal(in.StatusResponse)
	if err != nil {
		return CheckResult{}, fmt.Errorf("marshal status response: %w", err)
	}
	if err := validate.ValidateBytes(validate.StatusResponseSchemaRel, statusRaw); err != nil {
		return CheckResult{
			Name:       "schema_validity",
			Passed:     false,
			Message:    "status schema invalid: " + err.Error(),
			ReasonCode: wrkrerrors.EAcceptMissingArtifact,
		}, nil
	}

	for i, cp := range in.Checkpoints {
		raw, err := json.Marshal(cp)
		if err != nil {
			return CheckResult{}, fmt.Errorf("marshal checkpoint %d: %w", i+1, err)
		}
		if err := validate.ValidateBytes(validate.CheckpointSchemaRel, raw); err != nil {
			return CheckResult{
				Name:       "schema_validity",
				Passed:     false,
				Message:    fmt.Sprintf("checkpoint schema invalid at index %d: %v", i+1, err),
				ReasonCode: wrkrerrors.EAcceptMissingArtifact,
			}, nil
		}
	}

	for i, approval := range in.Approvals {
		raw, err := json.Marshal(approval)
		if err != nil {
			return CheckResult{}, fmt.Errorf("marshal approval %d: %w", i+1, err)
		}
		if err := validate.ValidateBytes(validate.ApprovalRecordSchemaRel, raw); err != nil {
			return CheckResult{
				Name:       "schema_validity",
				Passed:     false,
				Message:    fmt.Sprintf("approval schema invalid at index %d: %v", i+1, err),
				ReasonCode: wrkrerrors.EAcceptMissingArtifact,
			}, nil
		}
	}

	return CheckResult{Name: "schema_validity", Passed: true, Message: "all schemas valid"}, nil
}

func checkRequiredArtifacts(cfg Config, checkpoints []v1.Checkpoint) CheckResult {
	if len(cfg.RequiredArtifacts) == 0 {
		return CheckResult{Name: "required_artifacts", Passed: true, Message: "no required artifacts configured"}
	}

	seen := collectArtifacts(checkpoints, false)
	missing := make([]string, 0, len(cfg.RequiredArtifacts))
	for _, required := range sortedUnique(cfg.RequiredArtifacts) {
		if _, ok := seen[required]; !ok {
			missing = append(missing, required)
		}
	}
	if len(missing) == 0 {
		return CheckResult{Name: "required_artifacts", Passed: true, Message: "all required artifacts present"}
	}

	sort.Strings(missing)
	return CheckResult{
		Name:       "required_artifacts",
		Passed:     false,
		Message:    "missing required artifacts: " + strings.Join(missing, ", "),
		ReasonCode: wrkrerrors.EAcceptMissingArtifact,
		Artifact:   missing[0],
	}
}

func checkPathConstraints(rules PathRules, checkpoints []v1.Checkpoint) CheckResult {
	artifacts := sortedArtifactPaths(collectArtifacts(checkpoints, true))

	if rules.MaxArtifactPaths > 0 && len(artifacts) > rules.MaxArtifactPaths {
		return CheckResult{
			Name:       "path_constraints",
			Passed:     false,
			Message:    fmt.Sprintf("artifact path count %d exceeds max_artifact_paths=%d", len(artifacts), rules.MaxArtifactPaths),
			ReasonCode: wrkrerrors.EAcceptMissingArtifact,
		}
	}

	for _, prefix := range sortedUnique(rules.ForbiddenPrefixes) {
		for _, artifact := range artifacts {
			if strings.HasPrefix(artifact, prefix) {
				return CheckResult{
					Name:       "path_constraints",
					Passed:     false,
					Message:    fmt.Sprintf("artifact path %q matches forbidden prefix %q", artifact, prefix),
					ReasonCode: wrkrerrors.EAcceptMissingArtifact,
					Artifact:   artifact,
				}
			}
		}
	}

	for _, prefix := range sortedUnique(rules.AllowedPrefixes) {
		matched := false
		for _, artifact := range artifacts {
			if strings.HasPrefix(artifact, prefix) {
				matched = true
				break
			}
		}
		if !matched {
			return CheckResult{
				Name:       "path_constraints",
				Passed:     false,
				Message:    fmt.Sprintf("no artifact path matched allowed prefix %q", prefix),
				ReasonCode: wrkrerrors.EAcceptMissingArtifact,
			}
		}
	}

	return CheckResult{Name: "path_constraints", Passed: true, Message: "path constraints satisfied"}
}

func runCommandCheck(name, command, workDir string) (CheckResult, error) {
	if command == "" {
		return CheckResult{
			Name:       name,
			Passed:     false,
			Message:    name + " is not configured",
			ReasonCode: wrkrerrors.EAcceptTestFail,
		}, nil
	}

	// #nosec G204 -- command is explicitly configured by repository owner in accept.yaml.
	cmd := exec.Command("sh", "-lc", command)
	if workDir != "" {
		cmd.Dir = workDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return CheckResult{
			Name:       name,
			Passed:     false,
			Message:    fmt.Sprintf("command failed (exit=%d): %s", exitCode, boundedText(output)),
			ReasonCode: wrkrerrors.EAcceptTestFail,
		}, nil
	}

	return CheckResult{Name: name, Passed: true, Message: "command succeeded"}, nil
}

func collectArtifacts(checkpoints []v1.Checkpoint, includeRemoved bool) map[string]struct{} {
	out := make(map[string]struct{}, 32)
	for _, cp := range checkpoints {
		for _, path := range cp.ArtifactsDelta.Added {
			out[path] = struct{}{}
		}
		for _, path := range cp.ArtifactsDelta.Changed {
			out[path] = struct{}{}
		}
		if includeRemoved {
			for _, path := range cp.ArtifactsDelta.Removed {
				out[path] = struct{}{}
			}
		}
	}
	return out
}

func sortedArtifactPaths(paths map[string]struct{}) []string {
	out := make([]string, 0, len(paths))
	for path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func sortedUnique(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func boundedText(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "(no output)"
	}
	const max = 400
	if len(text) <= max {
		return text
	}
	return text[:max] + "..."
}
