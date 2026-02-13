package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/davidahmann/wrkr/core/approve"
	"github.com/davidahmann/wrkr/core/dispatch"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func runResume(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr resume <job_id> [--force-env] [--reason <text>] [--approved-by <user>]", nil),
			jsonMode,
			stderr,
			now,
		)
	}

	jobID := args[0]
	override := false
	overrideReason := ""
	approvedBy := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--force-env":
			override = true
		case "--reason":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--reason requires value", nil), jsonMode, stderr, now)
			}
			overrideReason = args[i]
		case "--approved-by":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--approved-by requires value", nil), jsonMode, stderr, now)
			}
			approvedBy = args[i]
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, fmt.Sprintf("unknown flag %q", args[i]), nil), jsonMode, stderr, now)
		}
	}

	if override {
		approvedBy = approve.ResolveApprovedBy(approvedBy)
	}

	s, err := openStore()
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	if err := ensureJobExists(s, jobID); err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	result, err := dispatch.Resume(jobID, dispatch.ResumeOptions{
		Now:                 now,
		OverrideEnvMismatch: override,
		OverrideReason:      overrideReason,
		ApprovedBy:          approvedBy,
	})
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}
	fmt.Fprintf(stdout, "job=%s status=%s\n", result.JobID, result.Status)
	return 0
}
