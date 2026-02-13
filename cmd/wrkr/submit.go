package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/davidahmann/wrkr/core/dispatch"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func runSubmit(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr submit <jobspec.yaml|json> [--job-id <id>]", nil),
			jsonMode,
			stderr,
			now,
		)
	}
	specPath := args[0]
	jobID := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--job-id":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--job-id requires value", nil), jsonMode, stderr, now)
			}
			jobID = args[i]
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown submit flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	result, err := dispatch.Submit(specPath, dispatch.SubmitOptions{
		Now:   now,
		JobID: jobID,
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
	fmt.Fprintf(stdout, "job_id=%s status=%s adapter=%s\n", result.JobID, result.Status, result.Adapter)
	return 0
}
