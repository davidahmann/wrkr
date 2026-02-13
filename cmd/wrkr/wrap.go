package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	wrapadapter "github.com/davidahmann/wrkr/core/adapters/wrap"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/pack"
	"github.com/davidahmann/wrkr/core/projectconfig"
)

func runWrap(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr wrap [--job-id <id>] [--artifact <path>] [--out-dir <dir>] -- <command...>", nil),
			jsonMode,
			stderr,
			now,
		)
	}

	jobID := fmt.Sprintf("job_wrap_%d", now().UTC().Unix())
	jobID = projectconfig.NormalizeJobID(jobID)
	artifacts := []string{}
	outDir := ""

	split := -1
	for i, arg := range args {
		if arg == "--" {
			split = i
			break
		}
	}
	if split == -1 || split == len(args)-1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "wrap command must include -- followed by executable command", nil),
			jsonMode,
			stderr,
			now,
		)
	}

	for i := 0; i < split; i++ {
		switch args[i] {
		case "--job-id":
			i++
			if i >= split {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--job-id requires value", nil), jsonMode, stderr, now)
			}
			jobID = projectconfig.NormalizeJobID(args[i])
		case "--artifact":
			i++
			if i >= split {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--artifact requires value", nil), jsonMode, stderr, now)
			}
			artifacts = append(artifacts, args[i])
		case "--out-dir":
			i++
			if i >= split {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--out-dir requires value", nil), jsonMode, stderr, now)
			}
			outDir = args[i]
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown wrap flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}
	command := args[split+1:]

	result, runErr := wrapadapter.Run(jobID, command, wrapadapter.RunOptions{
		Now:            now,
		ExpectedOutput: artifacts,
	})

	exported, exportErr := pack.ExportJobpack(jobID, pack.ExportOptions{
		OutDir:          outDir,
		Now:             now,
		ProducerVersion: version,
	})
	if exportErr != nil {
		return printError(exportErr, jsonMode, stderr, now)
	}

	if jsonMode {
		payload := map[string]any{
			"job_id":          jobID,
			"adapter_result":  result,
			"jobpack_path":    exported.Path,
			"manifest_sha256": exported.ManifestSHA256,
			"footer":          exported.Footer,
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		if runErr != nil {
			return printError(runErr, jsonMode, stderr, now)
		}
		return 0
	}

	fmt.Fprintf(stdout, "job_id=%s\n", jobID)
	fmt.Fprintf(stdout, "jobpack=%s\n", exported.Path)
	fmt.Fprintf(stdout, "footer=%s\n", exported.Footer)
	if runErr != nil {
		return printError(runErr, jsonMode, stderr, now)
	}
	return 0
}
