package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/pack"
)

func runVerify(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr verify <job_id|path> [--out-dir <dir>]", nil),
			jsonMode,
			stderr,
			now,
		)
	}
	target := args[0]
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
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown verify flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	path, _, err := resolveJobpackPath(target, outDir)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	result, err := pack.VerifyJobpack(path)
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

	fmt.Fprintf(stdout, "verified job_id=%s manifest=sha256:%s files=%d\n", result.JobID, result.ManifestSHA256, result.FilesVerified)
	return 0
}
