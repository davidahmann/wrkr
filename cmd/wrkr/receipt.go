package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/pack"
)

func runReceipt(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr receipt <job_id|path> [--out-dir <dir>]", nil),
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
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown receipt flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	path, isPath := resolveJobpackPath(target, outDir)
	if !isPath {
		exported, err := pack.ExportJobpack(target, pack.ExportOptions{OutDir: outDir, Now: now, ProducerVersion: version})
		if err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		path = exported.Path
	}
	archive, err := pack.LoadArchive(path)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	footer := pack.TicketFooter(archive.Manifest.JobID, archive.Manifest.ManifestSHA256)

	if jsonMode {
		payload := map[string]string{
			"job_id":          archive.Manifest.JobID,
			"manifest_sha256": archive.Manifest.ManifestSHA256,
			"footer":          footer,
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}
	fmt.Fprintln(stdout, footer)
	return 0
}
