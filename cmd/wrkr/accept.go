package main

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/davidahmann/wrkr/core/accept"
	"github.com/davidahmann/wrkr/core/accept/checks"
	acceptreport "github.com/davidahmann/wrkr/core/accept/report"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/out"
	"github.com/davidahmann/wrkr/core/pack"
	ghreport "github.com/davidahmann/wrkr/core/report"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
)

type acceptRunOutput struct {
	JobID            string               `json:"job_id"`
	ConfigPath       string               `json:"config_path"`
	AcceptResultPath string               `json:"accept_result_path"`
	AcceptResult     v1.AcceptanceResult  `json:"accept_result"`
	Checks           []checks.CheckResult `json:"checks"`
	JUnitPath        string               `json:"junit_path,omitempty"`
	JobpackPath      string               `json:"jobpack_path,omitempty"`
	SummaryJSONPath  string               `json:"summary_json_path,omitempty"`
	SummaryMDPath    string               `json:"summary_markdown_path,omitempty"`
	StepSummaryPath  string               `json:"step_summary_path,omitempty"`
}

func runAccept(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr accept <init|run> ...", nil),
			jsonMode,
			stderr,
			now,
		)
	}

	switch args[0] {
	case "init":
		return runAcceptInit(args[1:], jsonMode, stdout, stderr, now)
	case "run":
		return runAcceptRun(args[1:], jsonMode, stdout, stderr, now)
	default:
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown accept subcommand", map[string]any{"command": args[0]}),
			jsonMode,
			stderr,
			now,
		)
	}
}

func runAcceptInit(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	path := accept.DefaultConfigPath
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
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown accept init flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	writtenPath, err := accept.InitConfig(path, force)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		payload := map[string]string{"config_path": writtenPath}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}

	fmt.Fprintf(stdout, "config=%s\n", writtenPath)
	return 0
}

func runAcceptRun(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(
			wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr accept run <job_id> [--ci] [--junit <path>] [--config <path>] [--out-dir <dir>]", nil),
			jsonMode,
			stderr,
			now,
		)
	}

	jobID := args[0]
	configPath := ""
	outDir := ""
	junitPath := ""
	ciMode := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--ci":
			ciMode = true
		case "--config":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--config requires value", nil), jsonMode, stderr, now)
			}
			configPath = args[i]
		case "--out-dir":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--out-dir requires value", nil), jsonMode, stderr, now)
			}
			outDir = args[i]
		case "--junit":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--junit requires value", nil), jsonMode, stderr, now)
			}
			junitPath = args[i]
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown accept run flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	runResult, err := accept.Run(jobID, accept.RunOptions{
		Now:             now,
		ProducerVersion: version,
		ConfigPath:      configPath,
		WorkDir:         ".",
	})
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	layout := out.NewLayout(outDir)
	if ciMode && junitPath == "" {
		junitPath = layout.ReportPath(fmt.Sprintf("accept_%s.junit.xml", jobID))
	}
	if junitPath != "" {
		junitPath = filepath.Clean(junitPath)
		if err := acceptreport.WriteJUnit(junitPath, runResult.CheckResult); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
	}

	output := acceptRunOutput{
		JobID:            jobID,
		ConfigPath:       runResult.ConfigPath,
		AcceptResultPath: runResult.ResultPath,
		AcceptResult:     runResult.Result,
		Checks:           runResult.CheckResult,
		JUnitPath:        junitPath,
	}

	if ciMode {
		exported, err := pack.ExportJobpack(jobID, pack.ExportOptions{OutDir: outDir, Now: now, ProducerVersion: version})
		if err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		summary, err := ghreport.BuildGitHubSummaryFromJobpack(exported.Path, ghreport.SummaryOptions{Now: now, ProducerVersion: version})
		if err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		written, err := ghreport.WriteGitHubSummary(summary, outDir)
		if err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		output.JobpackPath = exported.Path
		output.SummaryJSONPath = written.JSONPath
		output.SummaryMDPath = written.MarkdownPath
		output.StepSummaryPath = written.StepSummaryPath
	}

	exitCode := 0
	if accept.Failed(runResult.Result) {
		exitCode = wrkrerrors.ExitCodeFor(accept.FailureCode(runResult.Result))
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return exitCode
	}

	fmt.Fprintf(stdout, "job=%s checks=%d passed=%d failures=%d\n", output.JobID, output.AcceptResult.ChecksRun, output.AcceptResult.ChecksPassed, len(output.AcceptResult.Failures))
	fmt.Fprintf(stdout, "accept_result=%s\n", output.AcceptResultPath)
	if output.JUnitPath != "" {
		fmt.Fprintf(stdout, "junit=%s\n", output.JUnitPath)
	}
	if output.SummaryMDPath != "" {
		fmt.Fprintf(stdout, "summary=%s\n", output.SummaryMDPath)
	}
	for _, failure := range output.AcceptResult.Failures {
		fmt.Fprintf(stdout, "failure check=%s message=%s\n", failure.Check, boundedSummary(failure.Message))
	}
	return exitCode
}
