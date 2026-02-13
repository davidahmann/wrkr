package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, time.Now))
}

func run(args []string, stdout, stderr io.Writer, now func() time.Time) int {
	jsonMode := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
			continue
		}
		filtered = append(filtered, arg)
	}

	if len(filtered) == 0 || filtered[0] == "version" {
		if jsonMode {
			payload := map[string]string{
				"version": version,
				"commit":  commit,
				"date":    date,
			}
			enc := json.NewEncoder(stdout)
			if err := enc.Encode(payload); err != nil {
				fmt.Fprintf(stderr, "encode version: %v\n", err)
				return 1
			}
			return 0
		}
		fmt.Fprintf(stdout, "wrkr %s (commit=%s date=%s)\n", version, commit, date)
		return 0
	}

	switch filtered[0] {
	case "status":
		return runStatus(filtered[1:], jsonMode, stdout, stderr, now)
	}

	return printError(
		wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			fmt.Sprintf("unknown command %q", filtered[0]),
			map[string]any{"command": filtered[0]},
		),
		jsonMode,
		stderr,
		now,
	)
}
