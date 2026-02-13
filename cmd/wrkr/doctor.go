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
	if len(args) > 0 {
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr doctor", nil), jsonMode, stderr, now)
	}
	result, err := doctor.Run(now)
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
	fmt.Fprintf(stdout, "doctor ok=%t checks=%d\n", result.OK, len(result.Checks))
	for _, check := range result.Checks {
		fmt.Fprintf(stdout, "- %s ok=%t %s\n", check.Name, check.OK, check.Details)
	}
	return 0
}
