package projectconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/fsx"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/schema/validate"
	"gopkg.in/yaml.v3"
)

const DefaultJobSpecPath = "jobspec.yaml"

var safeJobIDChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func InitJobSpec(path string, force bool, now time.Time, producerVersion string) (string, error) {
	target := strings.TrimSpace(path)
	if target == "" {
		target = DefaultJobSpecPath
	}
	resolved, err := fsx.ResolveWithinWorkingDir(filepath.Clean(target))
	if err != nil {
		return "", wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"jobspec path must stay within working directory",
			map[string]any{"path": target, "error": err.Error()},
		)
	}
	target = resolved

	if !force {
		if _, err := os.Stat(target); err == nil {
			return "", wrkrerrors.New(
				wrkrerrors.EInvalidInputSchema,
				"jobspec already exists (use --force to overwrite)",
				map[string]any{"path": target},
			)
		}
	}

	if strings.TrimSpace(producerVersion) == "" {
		producerVersion = "dev"
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	spec := defaultJobSpec(now.UTC(), producerVersion)
	normalized, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshal jobspec json: %w", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(normalized, &asMap); err != nil {
		return "", fmt.Errorf("normalize jobspec map: %w", err)
	}
	raw, err := yaml.Marshal(asMap)
	if err != nil {
		return "", fmt.Errorf("marshal jobspec yaml: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return "", fmt.Errorf("create jobspec dir: %w", err)
	}
	if err := fsx.AtomicWriteFile(target, raw, 0o600); err != nil {
		return "", err
	}
	return target, nil
}

func LoadJobSpec(path string) (*v1.JobSpec, error) {
	target := strings.TrimSpace(path)
	if target == "" {
		return nil, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "jobspec path is required", nil)
	}
	resolved, err := fsx.ResolveWithinWorkingDir(filepath.Clean(target))
	if err != nil {
		return nil, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"jobspec path must stay within working directory",
			map[string]any{"path": target, "error": err.Error()},
		)
	}
	root, err := os.OpenRoot(filepath.Dir(resolved))
	if err != nil {
		return nil, fmt.Errorf("open jobspec dir: %w", err)
	}
	defer func() { _ = root.Close() }()
	raw, err := root.ReadFile(filepath.Base(resolved))
	if err != nil {
		return nil, fmt.Errorf("read jobspec: %w", err)
	}

	var spec v1.JobSpec
	switch strings.ToLower(filepath.Ext(resolved)) {
	case ".json":
		if err := json.Unmarshal(raw, &spec); err != nil {
			return nil, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "decode jobspec json failed", map[string]any{"error": err.Error()})
		}
	default:
		var generic any
		if err := yaml.Unmarshal(raw, &generic); err != nil {
			return nil, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "decode jobspec yaml failed", map[string]any{"error": err.Error()})
		}
		normalizedYAML, err := json.Marshal(normalizeYAMLValue(generic))
		if err != nil {
			return nil, fmt.Errorf("normalize jobspec yaml: %w", err)
		}
		if err := json.Unmarshal(normalizedYAML, &spec); err != nil {
			return nil, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "decode jobspec yaml failed", map[string]any{"error": err.Error()})
		}
	}

	if spec.Inputs == nil {
		spec.Inputs = map[string]any{}
	}
	if spec.ExpectedArtifacts == nil {
		spec.ExpectedArtifacts = []string{}
	}
	if spec.CheckpointPolicy.RequiredTypes == nil {
		spec.CheckpointPolicy.RequiredTypes = []string{}
	}

	normalized, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("normalize jobspec json: %w", err)
	}
	if err := validate.ValidateBytes(validate.JobspecSchemaRel, normalized); err != nil {
		return nil, wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"jobspec schema invalid",
			map[string]any{"error": err.Error()},
		)
	}
	return &spec, nil
}

func NormalizeJobID(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	clean = strings.ReplaceAll(clean, " ", "_")
	clean = safeJobIDChars.ReplaceAllString(clean, "_")
	clean = strings.Trim(clean, "_.-")
	if clean == "" {
		return "job"
	}
	return clean
}

func defaultJobSpec(now time.Time, producerVersion string) v1.JobSpec {
	return v1.JobSpec{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.jobspec",
			SchemaVersion:   "v1",
			CreatedAt:       now,
			ProducerVersion: producerVersion,
		},
		Name:              "demo_refactor_job",
		Objective:         "Run a deterministic reference adapter workflow.",
		Inputs:            defaultInputs(),
		ExpectedArtifacts: []string{"reports/result.md"},
		Adapter: v1.AdapterConfig{
			Name: "reference",
			Config: map[string]any{
				"lane": "blessed",
			},
		},
		Budgets: v1.BudgetSpec{
			MaxWallTimeSeconds: 3600,
			MaxRetries:         2,
			MaxStepCount:       20,
			MaxToolCalls:       200,
		},
		CheckpointPolicy: v1.CheckpointPolicy{
			MinIntervalSeconds: 10,
			RequiredTypes:      []string{"plan", "progress", "decision-needed", "blocked", "completed"},
		},
		EnvironmentFingerprint: v1.EnvironmentFingerprint{
			Rules: []string{"go_version", "os", "arch"},
		},
	}
}

func defaultInputs() map[string]any {
	return map[string]any{
		"workspace": ".",
		"steps": []map[string]any{
			{
				"id":         "plan",
				"summary":    "analyze repository and draft plan",
				"command":    "true",
				"artifacts":  []string{"reports/plan.md"},
				"executed":   true,
				"checkpoint": "progress",
			},
			{
				"id":              "review",
				"summary":         "request manager approval before final write",
				"command":         "",
				"artifacts":       []string{},
				"decision_needed": true,
				"required_action": "approve",
				"executed":        false,
				"checkpoint":      "decision-needed",
			},
			{
				"id":         "finalize",
				"summary":    "write final artifact",
				"command":    "true",
				"artifacts":  []string{"reports/result.md"},
				"executed":   true,
				"checkpoint": "completed",
			},
		},
	}
}

func normalizeYAMLValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, v := range typed {
			out[key] = normalizeYAMLValue(v)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, v := range typed {
			out[fmt.Sprint(key)] = normalizeYAMLValue(v)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = normalizeYAMLValue(typed[i])
		}
		return out
	default:
		return value
	}
}
