package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/store"
)

func runStore(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr store <prune> ...", nil),
			jsonMode,
			stderr,
			now,
		)
	}
	switch args[0] {
	case "prune":
		return runStorePrune(args[1:], jsonMode, stdout, stderr, now)
	default:
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown store subcommand", map[string]any{"command": args[0]}),
			jsonMode,
			stderr,
			now,
		)
	}
}

func runStorePrune(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	opts := store.PruneOptions{
		Now:         now,
		MaxJobpacks: -1,
		MaxReports:  -1,
	}
	criteriaSet := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			opts.DryRun = true
		case "--store-root":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--store-root requires value", nil), jsonMode, stderr, now)
			}
			opts.StoreRoot = args[i]
		case "--out-dir":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--out-dir requires value", nil), jsonMode, stderr, now)
			}
			opts.OutRoot = args[i]
		case "--job-max-age":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--job-max-age requires value", nil), jsonMode, stderr, now)
			}
			parsed, err := time.ParseDuration(args[i])
			if err != nil || parsed <= 0 {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid --job-max-age", map[string]any{"value": args[i]}), jsonMode, stderr, now)
			}
			opts.JobMaxAge = parsed
			criteriaSet = true
		case "--jobpack-max-age":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--jobpack-max-age requires value", nil), jsonMode, stderr, now)
			}
			parsed, err := time.ParseDuration(args[i])
			if err != nil || parsed <= 0 {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid --jobpack-max-age", map[string]any{"value": args[i]}), jsonMode, stderr, now)
			}
			opts.JobpackMaxAge = parsed
			criteriaSet = true
		case "--report-max-age":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--report-max-age requires value", nil), jsonMode, stderr, now)
			}
			parsed, err := time.ParseDuration(args[i])
			if err != nil || parsed <= 0 {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid --report-max-age", map[string]any{"value": args[i]}), jsonMode, stderr, now)
			}
			opts.ReportMaxAge = parsed
			criteriaSet = true
		case "--integration-max-age":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--integration-max-age requires value", nil), jsonMode, stderr, now)
			}
			parsed, err := time.ParseDuration(args[i])
			if err != nil || parsed <= 0 {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid --integration-max-age", map[string]any{"value": args[i]}), jsonMode, stderr, now)
			}
			opts.IntegrationMaxAge = parsed
			criteriaSet = true
		case "--max-jobpacks":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--max-jobpacks requires value", nil), jsonMode, stderr, now)
			}
			parsed, err := strconv.Atoi(args[i])
			if err != nil || parsed < 0 {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid --max-jobpacks", map[string]any{"value": args[i]}), jsonMode, stderr, now)
			}
			opts.MaxJobpacks = parsed
			criteriaSet = true
		case "--max-reports":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--max-reports requires value", nil), jsonMode, stderr, now)
			}
			parsed, err := strconv.Atoi(args[i])
			if err != nil || parsed < 0 {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid --max-reports", map[string]any{"value": args[i]}), jsonMode, stderr, now)
			}
			opts.MaxReports = parsed
			criteriaSet = true
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown store prune flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	if !criteriaSet {
		return printError(
			wrkrerrors.New(
				wrkrerrors.EInvalidInputSchema,
				"at least one retention criterion is required",
				map[string]any{"hint": "set age and/or max-count flags"},
			),
			jsonMode,
			stderr,
			now,
		)
	}

	report, err := store.Prune(opts)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}
	fmt.Fprintf(
		stdout,
		"store prune dry_run=%t checked=%d matched=%d removed=%d freed_bytes=%d\n",
		report.DryRun,
		report.Checked,
		report.Matched,
		report.Removed,
		report.FreedBytes,
	)
	for _, entry := range report.Entries {
		fmt.Fprintf(stdout, "- kind=%s reason=%s path=%s size=%d\n", entry.Kind, entry.Reason, entry.Path, entry.SizeBytes)
	}
	return 0
}
