package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/pack"
)

func runExport(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr export <job_id> [--out-dir <dir>]", nil),
			jsonMode,
			stderr,
			now,
		)
	}
	jobID := args[0]
	outDir := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--out-dir":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--out-dir requires value", nil), jsonMode, stderr, now)
			}
			outDir = args[i]
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown export flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	result, err := pack.ExportJobpack(jobID, pack.ExportOptions{
		OutDir:          outDir,
		Now:             now,
		ProducerVersion: version,
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

	fmt.Fprintf(stdout, "job_id=%s\n", result.JobID)
	fmt.Fprintf(stdout, "jobpack=%s\n", result.Path)
	fmt.Fprintf(stdout, "footer=%s\n", result.Footer)
	return 0
}
