package main

import (
	"encoding/json"
	"io"
	"time"

	"github.com/davidahmann/wrkr/core/approve"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func runApprove(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr approve <job_id> --checkpoint <id> --reason <text> [--approved-by <user>]", nil),
			jsonMode,
			stderr,
			now,
		)
	}
	jobID := args[0]

	var checkpointID string
	var reason string
	var approvedBy string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--checkpoint":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--checkpoint requires value", nil), jsonMode, stderr, now)
			}
			checkpointID = args[i]
		case "--reason":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--reason requires value", nil), jsonMode, stderr, now)
			}
			reason = args[i]
		case "--approved-by":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--approved-by requires value", nil), jsonMode, stderr, now)
			}
			approvedBy = args[i]
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown flag for approve", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	if checkpointID == "" {
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "checkpoint id is required", nil), jsonMode, stderr, now)
	}
	if err := approve.ValidateReason(reason); err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	approvedBy = approve.ResolveApprovedBy(approvedBy)

	r, s, err := openRunner(now)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	if err := ensureJobExists(s, jobID); err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	record, err := r.ApproveCheckpoint(jobID, checkpointID, reason, approvedBy)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(record); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}

	_, _ = io.WriteString(stdout, "approved checkpoint="+record.CheckpointID+"\n")
	return 0
}
