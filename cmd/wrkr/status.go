package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	statusview "github.com/davidahmann/wrkr/core/status"
)

func runStatus(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) != 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr status <job_id>", nil),
			jsonMode,
			stderr,
			now,
		)
	}

	jobID := args[0]
	r, s, err := openRunner(now)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	if err := ensureJobExists(s, jobID); err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	state, err := r.Recover(jobID)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	resp := statusview.FromRunnerState(state, version, now())
	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}

	if resp.Lease != nil {
		fmt.Fprintf(stdout, "job=%s status=%s lease_worker=%s lease_expires=%s\n", resp.JobID, resp.Status, resp.Lease.WorkerID, resp.Lease.ExpiresAt.Format(time.RFC3339))
		return 0
	}

	fmt.Fprintf(stdout, "job=%s status=%s\n", resp.JobID, resp.Status)
	return 0
}

func printError(err error, jsonMode bool, stderr io.Writer, now func() time.Time) int {
	if jsonMode {
		out, marshalErr := wrkrerrors.MarshalEnvelope(err, version, now().UTC())
		if marshalErr != nil {
			fmt.Fprintf(stderr, "marshal error envelope: %v\n", marshalErr)
			return 1
		}
		fmt.Fprintln(stderr, string(out))
	} else {
		fmt.Fprintf(stderr, "%v\n", err)
	}

	var werr wrkrerrors.WrkrError
	if errors.As(err, &werr) {
		return wrkrerrors.ExitCodeFor(werr.Code)
	}
	return 1
}
