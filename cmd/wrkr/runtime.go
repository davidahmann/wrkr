package main

import (
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
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
