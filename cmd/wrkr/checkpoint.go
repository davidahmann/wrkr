package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
)

func runCheckpoint(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr checkpoint <list|show|emit> ...", nil),
			jsonMode,
			stderr,
			now,
		)
	}

	switch args[0] {
	case "list":
		return runCheckpointList(args[1:], jsonMode, stdout, stderr, now)
	case "show":
		return runCheckpointShow(args[1:], jsonMode, stdout, stderr, now)
	case "emit":
		return runCheckpointEmit(args[1:], jsonMode, stdout, stderr, now)
	default:
		return printError(
			wrkrerrors.New(
				wrkrerrors.EInvalidInputSchema,
				fmt.Sprintf("unknown checkpoint command %q", args[0]),
				map[string]any{"command": args[0]},
			),
			jsonMode,
			stderr,
			now,
		)
	}
}

func runCheckpointList(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) != 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr checkpoint list <job_id>", nil),
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

	checkpoints, err := r.ListCheckpoints(jobID)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(checkpoints); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}

	for _, cp := range checkpoints {
		fmt.Fprintf(stdout, "checkpoint=%s type=%s status=%s summary=%s\n", cp.CheckpointID, cp.Type, cp.Status, boundedSummary(cp.Summary))
	}
	return 0
}

func runCheckpointShow(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) != 2 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr checkpoint show <job_id> <checkpoint_id>", nil),
			jsonMode,
			stderr,
			now,
		)
	}
	jobID := args[0]
	checkpointID := args[1]

	r, s, err := openRunner(now)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	if err := ensureJobExists(s, jobID); err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	cp, err := r.GetCheckpoint(jobID, checkpointID)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(cp); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}

	fmt.Fprintf(stdout, "checkpoint=%s type=%s status=%s\n", cp.CheckpointID, cp.Type, cp.Status)
	fmt.Fprintf(stdout, "summary=%s\n", boundedSummary(cp.Summary))
	if len(cp.ReasonCodes) > 0 {
		fmt.Fprintf(stdout, "reason_codes=%s\n", strings.Join(cp.ReasonCodes, ","))
	}
	return 0
}

func runCheckpointEmit(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(
			wrkrerrors.New(
				wrkrerrors.EInvalidInputSchema,
				"usage: wrkr checkpoint emit <job_id> --type <plan|progress|decision-needed|blocked|completed> --summary <text>",
				nil,
			),
			jsonMode,
			stderr,
			now,
		)
	}
	jobID := args[0]

	var (
		cpType               string
		summary              string
		statusValue          string
		requiredKind         string
		requiredInstructions string
		reasonCodes          []string
	)

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--type":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--type requires a value", nil), jsonMode, stderr, now)
			}
			cpType = args[i]
		case "--summary":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--summary requires a value", nil), jsonMode, stderr, now)
			}
			summary = args[i]
		case "--status":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--status requires a value", nil), jsonMode, stderr, now)
			}
			statusValue = args[i]
		case "--required-kind":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--required-kind requires a value", nil), jsonMode, stderr, now)
			}
			requiredKind = args[i]
		case "--required-instructions":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--required-instructions requires a value", nil), jsonMode, stderr, now)
			}
			requiredInstructions = args[i]
		case "--reason-code":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--reason-code requires a value", nil), jsonMode, stderr, now)
			}
			reasonCodes = append(reasonCodes, strings.TrimSpace(args[i]))
		default:
			return printError(
				wrkrerrors.New(
					wrkrerrors.EInvalidInputSchema,
					fmt.Sprintf("unknown flag %q", args[i]),
					map[string]any{"flag": args[i]},
				),
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

	var required *v1.RequiredAction
	if strings.TrimSpace(requiredKind) != "" || strings.TrimSpace(requiredInstructions) != "" {
		required = &v1.RequiredAction{
			Kind:         strings.TrimSpace(requiredKind),
			Instructions: strings.TrimSpace(requiredInstructions),
		}
	}

	var status queue.Status
	if strings.TrimSpace(statusValue) != "" {
		status = queue.Status(strings.TrimSpace(statusValue))
	}

	cp, err := r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:           strings.TrimSpace(cpType),
		Summary:        summary,
		Status:         status,
		RequiredAction: required,
		ReasonCodes:    reasonCodes,
	})
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(cp); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}
	fmt.Fprintf(stdout, "checkpoint=%s type=%s status=%s\n", cp.CheckpointID, cp.Type, cp.Status)
	return 0
}

func boundedSummary(v string) string {
	s := strings.TrimSpace(v)
	const max = 160
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
