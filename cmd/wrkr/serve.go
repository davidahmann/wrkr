package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/serve"
)

func runServe(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	cfg := serve.Config{
		Now:             now,
		ProducerVersion: version,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--listen":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--listen requires value", nil), jsonMode, stderr, now)
			}
			cfg.ListenAddr = args[i]
		case "--allow-non-loopback":
			cfg.AllowNonLoopback = true
		case "--auth-token":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--auth-token requires value", nil), jsonMode, stderr, now)
			}
			cfg.AuthToken = args[i]
		case "--max-body-bytes":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--max-body-bytes requires value", nil), jsonMode, stderr, now)
			}
			var parsed int64
			_, err := fmt.Sscanf(args[i], "%d", &parsed)
			if err != nil || parsed <= 0 {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid --max-body-bytes value", map[string]any{"value": args[i]}), jsonMode, stderr, now)
			}
			cfg.MaxBodyBytes = parsed
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown serve flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	validated, err := serve.ValidateConfig(cfg)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{
			"listen":             validated.ListenAddr,
			"allow_non_loopback": validated.AllowNonLoopback,
			"max_body_bytes":     validated.MaxBodyBytes,
			"auth_enabled":       validated.AuthToken != "",
		}); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		fmt.Fprintf(stderr, "wrkr serve listening on %s\n", validated.ListenAddr)
	} else {
		fmt.Fprintf(stdout, "wrkr serve listening on %s\n", validated.ListenAddr)
	}
	server := serve.New(validated)
	if err := server.ListenAndServe(); err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	return 0
}
