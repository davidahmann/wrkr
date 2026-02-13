package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/davidahmann/wrkr/core/doctor"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func runDoctor(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	opts := doctor.Options{
		Now: now,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--production-readiness":
			opts.ProductionReadiness = true
		case "--serve-listen":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--serve-listen requires value", nil), jsonMode, stderr, now)
			}
			opts.ServeListenAddr = args[i]
			opts.ServeListenAddrSet = true
		case "--serve-auth-token":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--serve-auth-token requires value", nil), jsonMode, stderr, now)
			}
			opts.ServeAuthToken = args[i]
			opts.ServeAuthTokenSet = true
		case "--serve-max-body-bytes":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--serve-max-body-bytes requires value", nil), jsonMode, stderr, now)
			}
			parsed, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil || parsed <= 0 {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid --serve-max-body-bytes", map[string]any{"value": args[i]}), jsonMode, stderr, now)
			}
			opts.ServeMaxBodyBytes = parsed
			opts.ServeMaxBodyBytesSet = true
		case "--serve-allow-non-loopback":
			opts.ServeAllowNonLoopback = true
			opts.ServeAllowNonLoopbackSet = true
		default:
			return printError(wrkrerrors.New(
				wrkrerrors.EInvalidInputSchema,
				"usage: wrkr doctor [--production-readiness] [--serve-listen <addr>] [--serve-allow-non-loopback] [--serve-auth-token <token>] [--serve-max-body-bytes <n>]",
				nil,
			), jsonMode, stderr, now)
		}
	}
	result, err := doctor.RunWithOptions(opts)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		if !result.OK {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "doctor profile=%s ok=%t checks=%d\n", result.Profile, result.OK, len(result.Checks))
	for _, check := range result.Checks {
		fmt.Fprintf(stdout, "- %s ok=%t severity=%s %s\n", check.Name, check.OK, check.Severity, check.Details)
		if !check.OK && check.Remediation != "" {
			fmt.Fprintf(stdout, "  remediation=%s\n", check.Remediation)
		}
	}
	if !result.OK {
		return 1
	}
	return 0
}
