package main

import (
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/davidahmann/wrkr/core/budget"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func runBudget(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 || args[0] != "check" {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr budget check <job_id> [limits...]", nil),
			jsonMode,
			stderr,
			now,
		)
	}
	if len(args) < 2 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr budget check <job_id> [limits...]", nil),
			jsonMode,
			stderr,
			now,
		)
	}

	jobID := args[1]
	limits := budget.Limits{}

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--max-wall-time-seconds":
			i++
			v, err := parseIntFlag(args, i, "--max-wall-time-seconds")
			if err != nil {
				return printError(err, jsonMode, stderr, now)
			}
			limits.MaxWallTimeSeconds = v
		case "--max-retries":
			i++
			v, err := parseIntFlag(args, i, "--max-retries")
			if err != nil {
				return printError(err, jsonMode, stderr, now)
			}
			limits.MaxRetries = v
		case "--max-step-count":
			i++
			v, err := parseIntFlag(args, i, "--max-step-count")
			if err != nil {
				return printError(err, jsonMode, stderr, now)
			}
			limits.MaxStepCount = v
		case "--max-tool-calls":
			i++
			v, err := parseIntFlag(args, i, "--max-tool-calls")
			if err != nil {
				return printError(err, jsonMode, stderr, now)
			}
			limits.MaxToolCalls = v
		default:
			return printError(
				wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown budget flag", map[string]any{"flag": args[i]}),
				jsonMode,
				stderr,
				now,
			)
		}
	}

	r, s, err := openRunner(now)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	if err := ensureJobExists(s, jobID); err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	cp, err := r.CheckBudget(jobID, limits)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if cp != nil {
			if err := enc.Encode(cp); err != nil {
				return printError(err, jsonMode, stderr, now)
			}
		} else {
			if err := enc.Encode(map[string]string{"result": "within_budget"}); err != nil {
				return printError(err, jsonMode, stderr, now)
			}
		}
		return 0
	}

	if cp == nil {
		_, _ = io.WriteString(stdout, "budget=within_limits\n")
	} else {
		_, _ = io.WriteString(stdout, "budget=exceeded checkpoint="+cp.CheckpointID+"\n")
	}
	return 0
}

func parseIntFlag(args []string, idx int, flag string) (int, error) {
	if idx >= len(args) {
		return 0, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, flag+" requires value", nil)
	}
	v, err := strconv.Atoi(args[idx])
	if err != nil {
		return 0, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid integer for "+flag, map[string]any{"value": args[idx]})
	}
	return v, nil
}
