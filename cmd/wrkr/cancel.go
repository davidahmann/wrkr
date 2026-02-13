package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
)

func runCancel(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) != 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr cancel <job_id>", nil),
			jsonMode,
			stderr,
			now,
		)
	}

	r, s, err := openRunner(now)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	jobID := args[0]
	if err := ensureJobExists(s, jobID); err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	state, err := r.ChangeStatus(jobID, queue.StatusCanceled)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(state); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}
	fmt.Fprintf(stdout, "job=%s status=%s\n", state.JobID, state.Status)
	return 0
}
