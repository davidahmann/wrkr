package main

import (
	"os"
	"path/filepath"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/out"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
)

func openRunner(now func() time.Time) (*runner.Runner, *store.LocalStore, error) {
	s, err := store.New("")
	if err != nil {
		return nil, nil, err
	}
	r, err := runner.New(s, runner.Options{Now: now})
	if err != nil {
		return nil, nil, err
	}
	return r, s, nil
}

func ensureJobExists(s *store.LocalStore, jobID string) error {
	exists, err := s.JobExists(jobID)
	if err != nil {
		return err
	}
	if !exists {
		return wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"job not found",
			map[string]any{"job_id": jobID},
		)
	}
	return nil
}

func resolveJobpackPath(target, outDir string) (path string, isPath bool) {
	if target == "" {
		return "", false
	}
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		return target, true
	}
	if abs, err := filepath.Abs(target); err == nil {
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			return abs, true
		}
	}

	layout := out.NewLayout(outDir)
	return layout.JobpackPath(target), false
}
