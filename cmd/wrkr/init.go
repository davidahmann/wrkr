package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/projectconfig"
)

func runInit(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	path := ""
	force := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--path requires value", nil), jsonMode, stderr, now)
			}
			path = args[i]
		case "--force":
			force = true
		default:
			if path == "" && args[i] != "" && args[i][0] != '-' {
				path = args[i]
				continue
			}
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown init flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	written, err := projectconfig.InitJobSpec(path, force, now(), version)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]string{"jobspec_path": written}); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}
	fmt.Fprintf(stdout, "jobspec=%s\n", written)
	return 0
}
