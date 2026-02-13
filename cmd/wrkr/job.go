package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/pack"
	"github.com/davidahmann/wrkr/core/runner"
	"github.com/davidahmann/wrkr/core/store"
)

func runJob(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 {
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr job <inspect|diff> ...", nil), jsonMode, stderr, now)
	}
	switch args[0] {
	case "inspect":
		return runJobInspect(args[1:], jsonMode, stdout, stderr, now)
	case "diff":
		return runJobDiff(args[1:], jsonMode, stdout, stderr, now)
	default:
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown job subcommand", map[string]any{"command": args[0]}), jsonMode, stderr, now)
	}
}

func runJobInspect(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) < 1 {
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr job inspect <job_id|path> [--out-dir <dir>]", nil), jsonMode, stderr, now)
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
			return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown inspect flag", map[string]any{"flag": args[i]}), jsonMode, stderr, now)
		}
	}

	path, isPath, err := resolveJobpackPath(target, outDir)
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}
	var result pack.InspectResult
	if isPath {
		result, err = pack.InspectJobpack(path)
	} else {
		result, err = inspectFromStore(target, now)
	}
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
	fmt.Fprintf(stdout, "job=%s status=%s events=%d checkpoints=%d approvals=%d\n", result.JobID, result.Status, result.EventCount, result.CheckpointCount, result.ApprovalCount)
	for _, item := range result.Timeline {
		fmt.Fprintf(stdout, "%s %s %s %s\n", item.At.Format(time.RFC3339), item.Kind, item.ID, boundedSummary(item.Summary))
	}
	return 0
}

func runJobDiff(args []string, jsonMode bool, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) != 2 {
		return printError(wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "usage: wrkr job diff <jobpack_a> <jobpack_b>", nil), jsonMode, stderr, now)
	}
	diff, err := pack.DiffJobpacks(args[0], args[1])
	if err != nil {
		return printError(err, jsonMode, stderr, now)
	}

	if jsonMode {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(diff); err != nil {
			return printError(err, jsonMode, stderr, now)
		}
		return 0
	}
	fmt.Fprintf(stdout, "job_a=%s job_b=%s added=%d removed=%d changed=%d\n", diff.JobIDA, diff.JobIDB, len(diff.Added), len(diff.Removed), len(diff.Changed))
	return 0
}

func inspectFromStore(jobID string, now func() time.Time) (pack.InspectResult, error) {
	s, err := store.New("")
	if err != nil {
		return pack.InspectResult{}, err
	}
	if err := ensureJobExists(s, jobID); err != nil {
		return pack.InspectResult{}, err
	}
	r, err := runner.New(s, runner.Options{Now: now})
	if err != nil {
		return pack.InspectResult{}, err
	}
	state, err := r.Recover(jobID)
	if err != nil {
		return pack.InspectResult{}, err
	}
	checkpoints, err := r.ListCheckpoints(jobID)
	if err != nil {
		return pack.InspectResult{}, err
	}
	approvals, err := r.ListApprovals(jobID)
	if err != nil {
		return pack.InspectResult{}, err
	}
	events, err := s.LoadEvents(jobID)
	if err != nil {
		return pack.InspectResult{}, err
	}

	timeline := make([]pack.TimelineItem, 0, len(events)+len(checkpoints)+len(approvals))
	for _, evt := range events {
		timeline = append(timeline, pack.TimelineItem{
			At:      evt.CreatedAt.UTC(),
			Kind:    "event",
			ID:      fmt.Sprintf("evt_%d", evt.Seq),
			Summary: evt.Type,
		})
	}
	for _, cp := range checkpoints {
		timeline = append(timeline, pack.TimelineItem{
			At:      cp.CreatedAt.UTC(),
			Kind:    "checkpoint",
			ID:      cp.CheckpointID,
			Summary: cp.Type + ":" + cp.Status,
		})
	}
	for i, approval := range approvals {
		timeline = append(timeline, pack.TimelineItem{
			At:      approval.CreatedAt.UTC(),
			Kind:    "approval",
			ID:      fmt.Sprintf("approval_%d", i+1),
			Summary: approval.CheckpointID,
		})
	}
	sort.Slice(timeline, func(i, j int) bool {
		if timeline[i].At.Equal(timeline[j].At) {
			if timeline[i].Kind == timeline[j].Kind {
				return timeline[i].ID < timeline[j].ID
			}
			return timeline[i].Kind < timeline[j].Kind
		}
		return timeline[i].At.Before(timeline[j].At)
	})

	return pack.InspectResult{
		JobID:           jobID,
		Status:          string(state.Status),
		EventCount:      len(events),
		CheckpointCount: len(checkpoints),
		ApprovalCount:   len(approvals),
		Timeline:        timeline,
	}, nil
}
