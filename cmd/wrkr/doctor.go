package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/davidahmann/wrkr/core/doctor"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
)

func runDoctor(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	productionReadiness := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--production-readiness":
			productionReadiness = true
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr doctor [--production-readiness]", nil), jsonMode, stderr, now)
		}
	}

	result, err := doctor.RunWithOptions(doctor.Options{
		Now:                 now,
		ProductionReadiness: productionReadiness,
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
