package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/pack"
	"github.com/davidahmann/wrkr/core/projectconfig"
	"github.com/davidahmann/wrkr/core/queue"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
)

func runDemo(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	outDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out-dir":
			i++
			if i >= len(args) {
				return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "--out-dir requires value", nil), jsonMode, stderr, now)
			}
			outDir = args[i]
		default:
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown demo flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	jobID := fmt.Sprintf("job_demo_%d", now().UTC().Unix())
	jobID = projectconfig.NormalizeJobID(jobID)

	r, _, err := openRunner(now)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	if _, err := r.InitJob(jobID); err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	if _, err := r.ChangeStatus(jobID, queue.StatusRunning); err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{Type: "plan", Summary: "demo plan", Status: queue.StatusRunning})
	_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{
		Type:    "progress",
		Summary: "demo progress",
		Status:  queue.StatusRunning,
		ArtifactsDelta: v1.ArtifactsDelta{
			Added: []string{"reports/demo.md"},
		},
	})
	_, _ = r.EmitCheckpoint(jobID, runner.CheckpointInput{Type: "completed", Summary: "demo complete", Status: queue.StatusCompleted})
	if _, err := r.ChangeStatus(jobID, queue.StatusCompleted); err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	exported, err := pack.ExportJobpack(jobID, pack.ExportOptions{
		OutDir:          outDir,
		Now:             now,
		ProducerVersion: version,
	})
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if _, err := pack.VerifyJobpack(exported.Path); err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		payload := map[string]any{
			"job_id":   jobID,
			"jobpack":  exported.Path,
			"footer":   exported.Footer,
			"manifest": exported.ManifestSHA256,
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}
	fmt.Fprintf(stdout, "job_id=%s\n", jobID)
	fmt.Fprintf(stdout, "jobpack=%s\n", exported.Path)
	fmt.Fprintf(stdout, "footer=%s\n", exported.Footer)
	return 0
}
