package dispatch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/davidahmann/wrkr/core/budget"
	"github.com/davidahmann/wrkr/core/fsx"
	"github.com/davidahmann/wrkr/core/store"
)

type RuntimeConfig struct {
	SchemaID        string         `json:"schema_id"`
	SchemaVersion   string         `json:"schema_version"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	ProducerVersion string         `json:"producer_version"`
	Adapter         string         `json:"adapter"`
	Inputs          map[string]any `json:"inputs"`
	Budgets         budget.Limits  `json:"budgets"`
	NextStepIndex   int            `json:"next_step_index"`
}

func runtimeConfigPath(s *store.LocalStore, jobID string) string {
	return filepath.Join(s.JobDir(jobID), "runtime_config.json")
}

func SaveRuntimeConfig(s *store.LocalStore, jobID string, cfg RuntimeConfig, now time.Time) error {
	if s == nil {
		return fmt.Errorf("store is required")
	}
	if err := s.EnsureJob(jobID); err != nil {
		return err
	}
	if cfg.SchemaID == "" {
		cfg.SchemaID = "wrkr.runtime_config"
	}
	if cfg.SchemaVersion == "" {
		cfg.SchemaVersion = "v1"
	}
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now.UTC()
	}
	cfg.UpdatedAt = now.UTC()
	if cfg.Inputs == nil {
		cfg.Inputs = map[string]any{}
	}
	if cfg.NextStepIndex < 0 {
		cfg.NextStepIndex = 0
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal runtime config: %w", err)
	}
	if err := fsx.AtomicWriteFile(runtimeConfigPath(s, jobID), raw, 0o600); err != nil {
		return err
	}
	return nil
}

func LoadRuntimeConfig(s *store.LocalStore, jobID string) (*RuntimeConfig, error) {
	if s == nil {
		return nil, fmt.Errorf("store is required")
	}
	if err := s.EnsureJob(jobID); err != nil {
		return nil, err
	}
	// #nosec G304 -- runtime config path is store-scoped and job_id-validated.
	raw, err := os.ReadFile(runtimeConfigPath(s, jobID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read runtime config: %w", err)
	}
	var cfg RuntimeConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("decode runtime config: %w", err)
	}
	if cfg.Inputs == nil {
		cfg.Inputs = map[string]any{}
	}
	if cfg.NextStepIndex < 0 {
		cfg.NextStepIndex = 0
	}
	return &cfg, nil
}

