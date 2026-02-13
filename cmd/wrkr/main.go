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
	explainMode := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
			continue
		}
		if arg == "--explain" {
			explainMode = true
			continue
		}
		filtered = append(filtered, arg)
	}

	if explainMode {
		command := "version"
		if len(filtered) > 0 {
			command = filtered[0]
		}
		intent, ok := commandIntent(command)
		if !ok {
			return printError(
				wrkrerrors.New(
					wrkrerrors.EInvalidInputSchema,
					fmt.Sprintf("unknown command %q", command),
					map[string]any{"command": command},
				),
				jsonMode,
				stderr,
				now,
			)
		}
		if jsonMode {
			payload := map[string]string{
				"command": command,
				"intent":  intent,
			}
			enc := json.NewEncoder(stdout)
			if err := enc.Encode(payload); err != nil {
				fmt.Fprintf(stderr, "encode explain: %v\n", err)
				return 1
			}
			return 0
		}
		fmt.Fprintf(stdout, "wrkr %s: %s\n", command, intent)
		return 0
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
	case "demo":
		return runDemo(filtered[1:], jsonMode, stdout, stderr, now)
	case "init":
		return runInit(filtered[1:], jsonMode, stdout, stderr, now)
	case "submit":
		return runSubmit(filtered[1:], jsonMode, stdout, stderr, now)
	case "status":
		return runStatus(filtered[1:], jsonMode, stdout, stderr, now)
	case "checkpoint":
		return runCheckpoint(filtered[1:], jsonMode, stdout, stderr, now)
	case "pause":
		return runPause(filtered[1:], jsonMode, stdout, stderr, now)
	case "approve":
		return runApprove(filtered[1:], jsonMode, stdout, stderr, now)
	case "resume":
		return runResume(filtered[1:], jsonMode, stdout, stderr, now)
	case "cancel":
		return runCancel(filtered[1:], jsonMode, stdout, stderr, now)
	case "budget":
		return runBudget(filtered[1:], jsonMode, stdout, stderr, now)
	case "wrap":
		return runWrap(filtered[1:], jsonMode, stdout, stderr, now)
	case "export":
		return runExport(filtered[1:], jsonMode, stdout, stderr, now)
	case "verify":
		return runVerify(filtered[1:], jsonMode, stdout, stderr, now)
	case "job":
		return runJob(filtered[1:], jsonMode, stdout, stderr, now)
	case "receipt":
		return runReceipt(filtered[1:], jsonMode, stdout, stderr, now)
	case "accept":
		return runAccept(filtered[1:], jsonMode, stdout, stderr, now)
	case "report":
		return runReport(filtered[1:], jsonMode, stdout, stderr, now)
	case "bridge":
		return runBridge(filtered[1:], jsonMode, stdout, stderr, now)
	case "serve":
		return runServe(filtered[1:], jsonMode, stdout, stderr, now)
	case "doctor":
		return runDoctor(filtered[1:], jsonMode, stdout, stderr, now)
	case "store":
		return runStore(filtered[1:], jsonMode, stdout, stderr, now)
	case "help":
		return runHelp(stdout)
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

func commandIntent(command string) (string, bool) {
	switch command {
	case "version":
		return "show wrkr version, commit, and build date metadata", true
	case "demo":
		return "run an offline deterministic demo job and emit a verifiable jobpack", true
	case "init":
		return "generate a starter JobSpec file for dispatching a durable agent job", true
	case "submit":
		return "submit a JobSpec into durable execution and emit initial checkpoints", true
	case "status":
		return "read deterministic current job status from the durable store", true
	case "checkpoint":
		return "list or show structured checkpoint records for supervision and review", true
	case "pause":
		return "pause a running job without losing durable state", true
	case "approve":
		return "record an approval for a decision-needed checkpoint", true
	case "resume":
		return "resume a paused or blocked job from the last durable state", true
	case "cancel":
		return "cancel a job and persist terminal status and reason codes", true
	case "budget":
		return "inspect or update deterministic budget controls and stop conditions", true
	case "wrap":
		return "execute a command under wrkr durability and export job evidence", true
	case "export":
		return "assemble and write a deterministic jobpack artifact for a job", true
	case "verify":
		return "verify jobpack integrity, schema conformance, and manifest hashes", true
	case "job":
		return "inspect jobpack contents using deterministic inspect and diff surfaces", true
	case "receipt":
		return "materialize verification receipt data for external review workflows", true
	case "accept":
		return "run deterministic acceptance checks and emit CI-friendly results", true
	case "report":
		return "render deterministic supervision summaries for GitHub and review flows", true
	case "bridge":
		return "convert blocked or decision checkpoints into deterministic work-item payloads", true
	case "serve":
		return "start local wrkr API surface for submit/status/checkpoint/accept/report endpoints", true
	case "doctor":
		return "evaluate runtime, store, and hardening readiness with actionable diagnostics", true
	case "store":
		return "inspect and prune durable store data using deterministic retention controls", true
	case "help":
		return "show available wrkr command surfaces and usage hints", true
	default:
		return "", false
	}
}
