package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/pack"
	ghreport "github.com/davidahmann/wrkr/core/report"
)

type githubReportOutput struct {
	Summary         any    `json:"summary"`
	SummaryJSONPath string `json:"summary_json_path"`
	SummaryMDPath   string `json:"summary_markdown_path"`
	StepSummaryPath string `json:"step_summary_path,omitempty"`
}

func runReport(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 {
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr report <github> ...", nil), jsonMode, stderr, now)
	}
	switch args[0] {
	case "github":
		return runReportGitHub(args[1:], jsonMode, stdout, stderr, now)
	default:
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown report subcommand", map[string]any{"command": args[0]}), jsonMode, stderr, now)
	}
}

func runReportGitHub(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr report github <job_id|path> [--out-dir <dir>]", nil), jsonMode, stderr, now)
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
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown report github flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	jobpackPath, isPath := resolveJobpackPath(target, outDir)
	if !isPath {
		exported, err := pack.ExportJobpack(target, pack.ExportOptions{OutDir: outDir, Now: now, ProducerVersion: version})
		if err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		jobpackPath = exported.Path
	}

	summary, err := ghreport.BuildGitHubSummaryFromJobpack(jobpackPath, ghreport.SummaryOptions{Now: now, ProducerVersion: version})
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	written, err := ghreport.WriteGitHubSummary(summary, outDir)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		payload := githubReportOutput{
			Summary:         summary,
			SummaryJSONPath: written.JSONPath,
			SummaryMDPath:   written.MarkdownPath,
			StepSummaryPath: written.StepSummaryPath,
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}

	fmt.Fprintf(stdout, "summary_json=%s\n", written.JSONPath)
	fmt.Fprintf(stdout, "summary_markdown=%s\n", written.MarkdownPath)
	if written.StepSummaryPath != "" {
		fmt.Fprintf(stdout, "step_summary=%s\n", written.StepSummaryPath)
	}
	return 0
}
