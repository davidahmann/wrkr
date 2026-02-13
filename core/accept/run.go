package accept

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/davidahmann/wrkr/core/accept/checks"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/fsx"
	"github.com/davidahmann/wrkr/core/jcs"
	"github.com/davidahmann/wrkr/core/runner"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/schema/validate"
	statusview "github.com/davidahmann/wrkr/core/status"
	"github.com/davidahmann/wrkr/core/store"
)

type RunOptions struct {
	Now             func() time.Time
	ProducerVersion string
	ConfigPath      string
	WorkDir         string
}

type RunResult struct {
	ConfigPath  string               `json:"config_path"`
	ResultPath  string               `json:"accept_result_path"`
	Result      v1.AcceptanceResult  `json:"accept_result"`
	CheckResult []checks.CheckResult `json:"checks"`
}

func Run(jobID string, opts RunOptions) (RunResult, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	producerVersion := opts.ProducerVersion
	if producerVersion == "" {
		producerVersion = "dev"
	}

	s, err := store.New("")
	if err != nil {
		return RunResult{}, err
	}
	exists, err := s.JobExists(jobID)
	if err != nil {
		return RunResult{}, err
	}
	if !exists {
		return RunResult{}, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "job not found", map[string]any{"job_id": jobID})
	}

	r, err := runner.New(s, runner.Options{Now: now})
	if err != nil {
		return RunResult{}, err
	}
	state, err := r.Recover(jobID)
	if err != nil {
		return RunResult{}, err
	}
	checkpoints, err := r.ListCheckpoints(jobID)
	if err != nil {
		return RunResult{}, err
	}
	approvals, err := r.ListApprovals(jobID)
	if err != nil {
		return RunResult{}, err
	}

	cfg, configPath, err := LoadConfigOrDefault(opts.ConfigPath)
	if err != nil {
		return RunResult{}, err
	}

	checkResults, err := checks.Run(cfg.ToChecksConfig(), checks.Input{
		StatusResponse: statusview.FromRunnerState(state, producerVersion, now()),
		Checkpoints:    checkpoints,
		Approvals:      approvals,
		WorkDir:        opts.WorkDir,
	})
	if err != nil {
		return RunResult{}, err
	}

	acceptResult := buildAcceptanceResult(jobID, producerVersion, now().UTC(), checkResults)
	raw, err := json.Marshal(acceptResult)
	if err != nil {
		return RunResult{}, fmt.Errorf("marshal acceptance result: %w", err)
	}
	if err := validate.ValidateBytes(validate.AcceptResultSchemaRel, raw); err != nil {
		return RunResult{}, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "accept_result schema invalid", map[string]any{"error": err.Error()})
	}

	canonical, err := canonicalizeJSON(acceptResult)
	if err != nil {
		return RunResult{}, err
	}

	resultPath := filepath.Join(s.JobDir(jobID), "accept_result.json")
	if err := fsx.AtomicWriteFile(resultPath, canonical, 0o600); err != nil {
		return RunResult{}, fmt.Errorf("write acceptance result: %w", err)
	}

	return RunResult{
		ConfigPath:  configPath,
		ResultPath:  resultPath,
		Result:      acceptResult,
		CheckResult: checkResults,
	}, nil
}

func Failed(result v1.AcceptanceResult) bool {
	return len(result.Failures) > 0
}

func FailureCode(result v1.AcceptanceResult) wrkrerrors.Code {
	for _, code := range result.ReasonCodes {
		if wrkrerrors.Code(code) == wrkrerrors.EAcceptTestFail {
			return wrkrerrors.EAcceptTestFail
		}
	}
	return wrkrerrors.EAcceptMissingArtifact
}

func buildAcceptanceResult(jobID, producerVersion string, createdAt time.Time, checkResults []checks.CheckResult) v1.AcceptanceResult {
	failures := make([]v1.AcceptanceFailure, 0, len(checkResults))
	reasonCodes := make([]string, 0, len(checkResults))
	seenReason := map[string]struct{}{}
	passed := 0

	for _, check := range checkResults {
		if check.Passed {
			passed++
			continue
		}

		failures = append(failures, v1.AcceptanceFailure{
			Check:    check.Name,
			Message:  check.Message,
			Artifact: check.Artifact,
		})

		if check.ReasonCode != "" {
			code := string(check.ReasonCode)
			if _, ok := seenReason[code]; !ok {
				seenReason[code] = struct{}{}
				reasonCodes = append(reasonCodes, code)
			}
		}
	}

	if reasonCodes == nil {
		reasonCodes = []string{}
	}
	if failures == nil {
		failures = []v1.AcceptanceFailure{}
	}

	return v1.AcceptanceResult{
		Envelope: v1.Envelope{
			SchemaID:        "wrkr.accept_result",
			SchemaVersion:   "v1",
			CreatedAt:       createdAt.UTC(),
			ProducerVersion: producerVersion,
		},
		JobID:        jobID,
		ChecksRun:    len(checkResults),
		ChecksPassed: passed,
		Failures:     failures,
		ReasonCodes:  reasonCodes,
	}
}

func canonicalizeJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	canonical, err := jcs.Canonicalize(raw)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}
