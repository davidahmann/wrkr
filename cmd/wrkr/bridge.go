package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/davidahmann/wrkr/core/bridge"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func runBridge(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr bridge work-item <job_id> --checkpoint <id> [--dry-run] [--template github|jira] [--out-dir <dir>]", nil),
			jsonMode,
			stderr,
			now,
		)
	}
	switch args[0] {
	case "work-item":
		return runBridgeWorkItem(args[1:], jsonMode, stdout, stderr, now)
	default:
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown bridge subcommand", map[string]any{"command": args[0]}), jsonMode, stderr, now)
	}
}

func runBridgeWorkItem(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr bridge work-item <job_id> --checkpoint <id> [--dry-run] [--template github|jira] [--out-dir <dir>]", nil),
			jsonMode,
			stderr,
			now,
		)
	}
	jobID := args[0]
	checkpointID := ""
	dryRun := false
	outDir := ""
	template := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--checkpoint":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--checkpoint requires value", nil), jsonMode, stderr, now)
			}
			checkpointID = args[i]
		case "--dry-run":
			dryRun = true
		case "--template":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--template requires value", nil), jsonMode, stderr, now)
			}
			template = args[i]
		case "--out-dir":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--out-dir requires value", nil), jsonMode, stderr, now)
			}
			outDir = args[i]
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown bridge work-item flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}
	if checkpointID == "" {
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--checkpoint is required", nil), jsonMode, stderr, now)
	}

	r, s, err := openRunner(now)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	if err := ensureJobExists(s, jobID); err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	checkpoint, err := r.GetCheckpoint(jobID, checkpointID)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	if checkpoint.Type != "blocked" && checkpoint.Type != "decision-needed" {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "checkpoint type must be blocked or decision-needed", map[string]any{"type": checkpoint.Type}),
			jsonMode,
			stderr,
			now,
		)
	}

	payload, err := bridge.BuildWorkItemPayload(jobID, *checkpoint, bridge.BuildOptions{
		Now:             now,
		ProducerVersion: version,
	})
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	output := map[string]any{
		"payload": payload,
	}
	if !dryRun {
		written, err := bridge.WriteWorkItemPayload(payload, outDir, template)
		if err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		output["json_path"] = written.JSONPath
		output["template_path"] = written.TemplatePath
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}

	fmt.Fprintf(stdout, "job_id=%s checkpoint=%s type=%s\n", jobID, checkpointID, checkpoint.Type)
	if path, ok := output["json_path"].(string); ok && strings.TrimSpace(path) != "" {
		fmt.Fprintf(stdout, "work_item=%s\n", path)
	}
	fmt.Fprintln(stdout, "next_commands:")
	for _, cmd := range payload.NextCommands {
		fmt.Fprintf(stdout, "- %s\n", cmd)
	}
	return 0
}
