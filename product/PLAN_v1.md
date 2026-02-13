# PLAN v1: Wrkr (Unified OSS Execution Plan, Gait-Grade)

Date: 2026-02-13
Source of truth: `product/PRD.md`
Parity reference: `/Users/davidahmann/Projects/gait` (docs map, repo standards, CI/CD, hardening cadence)
Scope: single unified v1 plan for OSS (no ladders)

This plan is written for top-to-bottom execution with minimal interpretation. Every story includes concrete tasks, repo paths, and acceptance criteria. The objective is to land Wrkr at the same standards and operating maturity as the current Gait repo while implementing Wrkr primitives (`dispatch`, `checkpoint`, `accept`, `jobpack`).

---

## Global Decisions (Locked for v1)

- Core runtime and CLI are Go in a single static binary (`cmd/wrkr`).
- Python SDK is a thin adoption layer only and never owns policy, state, or artifact logic.
- Persisted artifact contracts are JSON Schema Draft 2020-12 and are append-only within `v1.x`.
- Canonicalization for any digest-bearing JSON uses RFC 8785 (JCS).
- Digests use `sha256`; signing uses `ed25519`.
- Default posture is offline-first, deterministic outputs, and reference-only artifact capture.
- Deterministic operator-facing outputs are written under `./wrkr-out/` by default (jobpacks, integrations, reports); output layout is a contract and is append-only within `v1.x`.
- Raw artifact capture and other unsafe behaviors require explicit `--unsafe-*` flags and are surfaced in `--json` output.
- Job execution claims use lease + heartbeat semantics; conflicting claims fail closed with `E_LEASE_CONFLICT` and never double-execute.
- Resume is gated by `environment_fingerprint`; mismatches block with `E_ENV_FINGERPRINT_MISMATCH` unless explicitly overridden (override is recorded in the job event ledger).
- `wrkr serve` is a local transport surface: loopback-only by default; non-loopback requires explicit auth token and request-size limits; no network listener by default.
- GitHub-native summaries are deterministic markdown generated from jobpacks/checkpoints/acceptance results and are written under `./wrkr-out/reports/`.
- "Wrkr-compatible" is a real, testable claim enforced by conformance + parity suites; it is release-blocking.
- CI/CD governance must be single-admin operable; no release or protection step may require two-admin approval.
- Major commands support `--json` and `--explain` with bounded summaries.
- Coverage thresholds are enforced in CI via `perf/coverage_thresholds.json` (current: Go `>=63%`, Python `>=85%`) with a ratchet plan to raise Go coverage without destabilizing release cadence.
- Documentation ownership mirrors Gait: `docs/contracts/*` are normative; `README.md` is onboarding only.
- CI cadence mirrors Gait: fast PR lane, broad mainline lane, nightly deep validation lanes.
- Release integrity mirrors Gait: checksums, SBOM, vulnerability scan, signatures, provenance, reproducible artifacts.

---

## Repository Layout (Target Mirror)

```
.
|-- AGENTS.md
|-- README.md
|-- CHANGELOG.md
|-- CONTRIBUTING.md
|-- SECURITY.md
|-- CODE_OF_CONDUCT.md
|-- CODEOWNERS
|-- Makefile
|-- .tool-versions
|-- .golangci.yml
|-- .pre-commit-config.yaml
|-- .goreleaser.yaml
|-- cmd/wrkr/
|-- core/
|   |-- accept/
|   |-- adapters/
|   |-- approve/
|   |-- bridge/
|   |-- budget/
|   |-- doctor/
|   |-- envfp/
|   |-- errors/
|   |-- export/
|   |-- fsx/
|   |-- jcs/
|   |-- lease/
|   |-- out/
|   |-- pack/
|   |-- projectconfig/
|   |-- queue/
|   |-- report/
|   |-- runner/
|   |-- schema/
|   |   `-- v1/
|   |-- serve/
|   |-- sign/
|   |-- status/
|   |-- store/
|   `-- zipx/
|-- schemas/v1/
|   |-- accept/
|   |-- bridge/
|   |-- checkpoint/
|   |-- jobspec/
|   |-- jobpack/
|   |-- report/
|   |-- serve/
|   `-- status/
|-- sdk/python/wrkr/
|-- internal/
|   |-- e2e/
|   |-- integration/
|   `-- testutil/
|-- examples/
|   |-- integrations/
|   |-- policy/
|   |-- ci/
|   `-- scenarios/
|-- scripts/
|-- docs/
|   |-- concepts/
|   |-- contracts/
|   |-- hardening/
|   |-- security/
|   |-- slo/
|   |-- launch/
|   |-- deployment/
|   |-- ecosystem/
|   `-- wiki/
|-- docs-site/
|-- perf/
|-- product/
|   |-- PRD.md
|   `-- PLAN_v1.md
`-- .github/
    |-- workflows/
    |-- ISSUE_TEMPLATE/
    `-- actions/
```

---

## v1 Exit Criteria (Release-Blocking)

v1 is complete only when all are true:

- `wrkr demo` works offline in under 60 seconds and produces a verifiable `jobpack`.
- `wrkr submit -> status -> checkpoint -> approve -> resume -> export -> accept` works deterministically from a fresh checkout.
- Crash/restart recovery preserves committed state; resume is deterministic from last durable checkpoint.
- Lease + heartbeat semantics prevent double execution; concurrent runners fail closed with `E_LEASE_CONFLICT`.
- Resume blocks safely on environment mismatch with `E_ENV_FINGERPRINT_MISMATCH` and requires explicit override to continue.
- `jobpack` verify catches tampering and schema mismatch with stable reason and exit codes.
- Acceptance harness runs locally and in CI with stable JSON and optional JUnit output.
- `wrkr-out` output layout is stable and validated (jobpacks, reports, integration artifacts).
- `wrkr bridge work-item` produces deterministic payloads (with `--dry-run`) from `blocked` and `decision-needed` checkpoints.
- GitHub-native summaries are produced deterministically and the GitHub Actions kit publishes them via `GITHUB_STEP_SUMMARY`.
- `wrkr serve` passes loopback-default hardening tests (auth required for non-loopback, request limits, no path traversal).
- Local security checks (`make sast-fast`) and CI CodeQL are both wired and documented, with deterministic triage flow.
- Required check policy and branch protection bootstrap are in place and manageable by a single admin account.
- PR fast lane, mainline CI lane, and nightly deep lanes are implemented and green.
- Docs map, contracts, architecture, and runbooks are complete and non-duplicative.
- Release workflow emits signed multi-platform artifacts, checksums, SBOM, scan output, and provenance.

---

## Epic 0: Program Foundations and Parity Scaffold

Objective: bootstrap Wrkr repo operations to Gait-grade standards before deep feature work.

### Story 0.1: Repository scaffold and mandatory metadata

Tasks:
- Create all directories in "Repository Layout".
- Add baseline files: `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CODEOWNERS`, `AGENTS.md`.
- Add GitHub issue templates and PR template.

Repo paths:
- repo root
- `.github/ISSUE_TEMPLATE/`
- `.github/pull_request_template.md`

Acceptance criteria:
- Repo tree exists and is committed.
- All new issues/PRs can be created with templates.

### Story 0.2: Toolchain and module initialization

Tasks:
- Pin tool versions in `.tool-versions` (Go, Python, Node).
- Initialize Go module and Python SDK package skeleton.
- Add `uv` lock and developer extras in `sdk/python/`.

Repo paths:
- `.tool-versions`
- `go.mod`, `go.sum`
- `sdk/python/pyproject.toml`

Acceptance criteria:
- `go test ./...` runs.
- `uv run --python 3.13 --extra dev pytest -q` runs under `sdk/python`.

### Story 0.3: Standard make targets (single operator surface)

Tasks:
- Add a `Makefile` mirroring Gait conventions for fmt/lint/test/build and acceptance lanes.
- Include fast and full pre-push targets.
- Include docs-site build/lint targets.

Required targets:
- `fmt`, `lint`, `lint-fast`, `test`, `test-fast`, `build`
- `sast-fast`, `codeql-local`
- `license-check`
- `test-e2e`, `test-acceptance`, `test-contracts`, `test-conformance`, `test-runtime-slo`, `test-hardening-acceptance`
- `test-v1-acceptance`, `test-adoption`, `test-uat-local`
- `docs-site-install`, `docs-site-build`, `docs-site-lint`

Acceptance criteria:
- `make fmt && make lint-fast && make test-fast && make sast-fast` succeeds on clean checkout.

### Story 0.4: Pre-commit and pre-push enforcement

Tasks:
- Add `.pre-commit-config.yaml` with whitespace, secret, Go, Python, and site-stack checks.
- Add `.githooks` and `make hooks` flow.
- Enforce hook path and repository hygiene in lint checks.
- Keep pre-push default lightweight (`lint-fast`, `test-fast`, `sast-fast`) for contributor ergonomics.

Acceptance criteria:
- `pre-commit run --all-files` passes.
- Pre-push hook runs fast lane by default.

---

## Epic 1: Contract-First Primitive Surface

Objective: define and lock Wrkr v1 schemas, types, and validation before runtime complexity.

### Story 1.1: Create normative schemas for Wrkr primitives

Tasks:
- Add JSON schemas for:
  - `jobspec`
  - `checkpoint`
  - `environment_fingerprint`
  - `lease` record
  - `jobpack manifest`
  - `job`
  - `events` record
  - `artifacts manifest`
  - `accept result`
  - `approval record`
  - `work item` payload (bridge)
  - `report` outputs (GitHub summary)
  - `serve` API contract (OpenAPI + error envelope mapping)
  - `status response`
- Include required fields: `schema_id`, `schema_version`, `created_at`, `producer_version`.

Repo paths:
- `schemas/v1/jobspec/`
- `schemas/v1/checkpoint/`
- `schemas/v1/bridge/`
- `schemas/v1/jobpack/`
- `schemas/v1/accept/`
- `schemas/v1/report/`
- `schemas/v1/serve/`
- `schemas/v1/status/`

Acceptance criteria:
- Schemas validate with Draft 2020-12 validators in CI.

### Story 1.2: Go type mapping and validators

Tasks:
- Implement matching Go structs under `core/schema/v1/*`.
- Add schema/JSONL validators under `core/schema/validate/`.
- Add valid/invalid fixtures and golden tests.

Repo paths:
- `core/schema/v1/`
- `core/schema/validate/`
- `core/schema/testdata/`

Acceptance criteria:
- Validators pass all valid fixtures and reject invalid fixtures deterministically.

### Story 1.3: Canonicalization and digest contract

Tasks:
- Implement `core/jcs` wrappers for RFC 8785 canonicalization.
- Implement deterministic digest helpers for all signed/verified artifacts.

Repo paths:
- `core/jcs/`
- `core/sign/`

Acceptance criteria:
- Same semantic JSON yields identical canonical bytes and digest across OS matrix.

### Story 1.4: Exit-code and error taxonomy lock

Tasks:
- Add ADR for error categories and exit-code mapping.
- Implement stable error envelope for `--json` output.
- Lock reason codes for lease conflicts and environment gating; document `wrkr serve` HTTP status mapping vs CLI exit codes.

Repo paths:
- `docs/adr/`
- `core/errors/`
- `cmd/wrkr/`

Acceptance criteria:
- Golden tests verify stable exit and error envelope behavior.

---

## Epic 2: Durable Store, Queue, and Recovery Engine

Objective: deliver crash-tolerant, append-only job state with deterministic recovery.

### Story 2.1: Append-only event store with snapshots

Tasks:
- Implement event log under `~/.wrkr/` with lock-safe appends.
- Implement snapshot checkpoints to speed status/recovery reads.
- Use atomic writes and lock contention strategy.

Repo paths:
- `core/store/`
- `core/fsx/`
- `docs/adr/`

Acceptance criteria:
- Crash during append does not corrupt previously committed records.

### Story 2.2: Queue and job lifecycle engine

Tasks:
- Implement local queue semantics (`queued`, `running`, `paused`, `blocked_*`, `completed`, `canceled`).
- Implement deterministic transitions and state machine checks.

Repo paths:
- `core/queue/`
- `core/status/`
- `core/runner/`

Acceptance criteria:
- Invalid transitions are rejected with stable reason codes.

### Story 2.3: Recovery and resumability contract

Tasks:
- Implement recovery from last committed checkpoint.
- Preserve counters and idempotency keys across resumes.
- Document replay and resume semantics clearly.

Repo paths:
- `core/runner/`
- `core/store/`
- `docs/contracts/`

Acceptance criteria:
- Forced process restart resumes from last durable checkpoint without state loss.

### Story 2.4: Session and contention reliability tests

Tasks:
- Add concurrency and contention tests for store append + lock behavior.
- Add soak script for long-running session durability.

Repo paths:
- `internal/integration/`
- `scripts/test_session_soak.sh`

Acceptance criteria:
- Stress tests show no malformed log lines and no lock starvation beyond defined thresholds.

### Story 2.5: Lease + heartbeat job claims (no double-execute)

Tasks:
- Define a `worker_id` and `lease_id` model persisted in job state.
- Implement `lease acquire` on job start/resume with deterministic conflict handling.
- Implement heartbeat writes and lease expiry semantics (TTL + interval, configurable with safe defaults).
- Ensure lease state is surfaced in `wrkr status --json`.
- Persist lease and heartbeat activity in the event ledger and include in `jobpack` export.

Repo paths:
- `core/lease/`
- `core/runner/`
- `core/store/`
- `core/status/`
- `cmd/wrkr/status.go`

Acceptance criteria:
- Two `wrkr` processes cannot both run the same job; the loser fails closed with `E_LEASE_CONFLICT`.
- After TTL expiry without heartbeat, a new runner can claim lease and resume deterministically.

---

## Epic 3: Runner, Checkpoint Protocol, Budgets, and Approvals

Objective: implement the supervision core (`plan`, `progress`, `decision-needed`, `blocked`, `completed`).

### Story 3.1: Checkpoint protocol implementation

Tasks:
- Implement checkpoint model and persistence.
- Enforce bounded summaries and artifact delta structure.
- Implement `decision-needed` required action payload.

Repo paths:
- `core/runner/`
- `core/schema/v1/checkpoint/`
- `cmd/wrkr/checkpoint.go`

Acceptance criteria:
- Checkpoints emit in deterministic order and schema-valid format.

### Story 3.2: Budget ledger and stop conditions

Tasks:
- Implement budget tracking for wall time, retries, step count, tool calls.
- Add optional adapter-provided token/cost fields.
- Emit deterministic stop checkpoint on exceed (`E_BUDGET_EXCEEDED`).

Repo paths:
- `core/budget/`
- `core/runner/`

Acceptance criteria:
- Budget exceed behavior is deterministic and test-covered.

### Story 3.3: Approval flow and records

Tasks:
- Implement approval token/recording for `decision-needed` checkpoints.
- Add CLI command for explicit approval reason capture.

Repo paths:
- `core/approve/`
- `cmd/wrkr/approve.go`
- `schemas/v1/checkpoint/`

Acceptance criteria:
- Decision-needed jobs block until valid approval is recorded.

### Story 3.4: Status and monitor outputs

Tasks:
- Implement `wrkr status`, checkpoint list/show with bounded text and stable `--json`.

Repo paths:
- `cmd/wrkr/status.go`
- `cmd/wrkr/checkpoint.go`
- `core/status/`

Acceptance criteria:
- Status outputs are deterministic and machine-parseable.

### Story 3.5: Environment fingerprint capture and resume gating

Tasks:
- Implement `environment_fingerprint` capture based on JobSpec rules and store it durably with the job.
- On `wrkr resume`, compare current environment fingerprint to the recorded fingerprint:
  - if incompatible, emit a `blocked` checkpoint with `reason_codes=[E_ENV_FINGERPRINT_MISMATCH]` and stop
  - require explicit override flag to continue, and record the override in the event ledger
- Surface environment mismatch state in `wrkr status` (bounded text and `--json`).

Repo paths:
- `core/envfp/`
- `core/runner/`
- `core/status/`
- `cmd/wrkr/resume.go`
- `cmd/wrkr/status.go`

Acceptance criteria:
- Resume deterministically blocks on mismatch with stable reason codes and exit codes.
- Override path is explicit and auditably recorded in job history and jobpack export.

---

## Epic 4: Jobpack Assembly, Verify, Inspect, and Diff

Objective: make `jobpack` the portable unit of review and verification.

### Story 4.1: Deterministic pack writer

Tasks:
- Implement deterministic zip writer (stable order, timestamps, metadata).
- Assemble required contents:
  - `manifest.json`
  - `job.json`
  - `events.jsonl`
  - `checkpoints.jsonl`
  - `artifacts_manifest.json`
  - `accept/accept_result.json` (optional)
  - `approvals.jsonl` (optional)

Repo paths:
- `core/pack/`
- `core/zipx/`

Acceptance criteria:
- Identical inputs produce identical zip bytes.

### Story 4.2: Verify command and hash/signature checks

Tasks:
- Implement `wrkr verify <job_id|path>`.
- Validate schema, manifest hashes, and signature envelope where configured.

Repo paths:
- `cmd/wrkr/verify.go`
- `core/pack/verify.go`

Acceptance criteria:
- Tampering is detected with stable exit code and reason.

### Story 4.3: Inspect and diff views

Tasks:
- Implement `wrkr job inspect <job_id|path>` for deterministic timeline output.
- Implement `wrkr job diff <jobpack_a> <jobpack_b>` for deterministic deltas.

Repo paths:
- `cmd/wrkr/job_inspect.go`
- `cmd/wrkr/job_diff.go`
- `core/pack/diff.go`

Acceptance criteria:
- Same inputs produce identical inspect/diff output.

### Story 4.4: Ticket footer and receipt contract

Tasks:
- Implement ticket footer generation and extraction helper.
- Add golden tests for footer shape and required fields.

Repo paths:
- `cmd/wrkr/receipt.go`
- `docs/contracts/ticket_footer_conformance.md`

Acceptance criteria:
- Footer contract is stable and CI-enforced.

### Story 4.5: Deterministic `./wrkr-out/` output layout (jobpacks, integrations, reports)

Tasks:
- Implement an output-path resolver that defaults to `./wrkr-out/` and can be overridden explicitly (`--out-dir`).
- Standardize and document v1 default subpaths:
  - `./wrkr-out/jobpacks/jobpack_<job_id>.zip`
  - `./wrkr-out/integrations/<lane>/...`
  - `./wrkr-out/reports/...`
- Update `wrkr demo`, `wrkr export`, and `wrkr accept run` to write to the standardized layout.
- Add golden tests for output path generation and filesystem writes (cross-platform).

Repo paths:
- `core/out/`
- `cmd/wrkr/demo.go`
- `cmd/wrkr/export.go`
- `cmd/wrkr/accept.go`
- `docs/contracts/`

Acceptance criteria:
- Default outputs land in the documented locations and are consistent across OS matrix.
- Output layout is treated as an additive contract within `v1.x` (no breaking moves).

---

## Epic 5: Acceptance Harness (Deterministic Review Signal)

Objective: deliver deterministic acceptance checks for local and CI use.

### Story 5.1: Acceptance config and command surface

Tasks:
- Implement `wrkr accept init` to create `accept.yaml`.
- Implement `wrkr accept run <job_id> [--ci] [--json] [--junit ...]`.

Repo paths:
- `cmd/wrkr/accept.go`
- `core/accept/`

Acceptance criteria:
- Accept commands run on clean repo with deterministic outputs.

### Story 5.2: Deterministic checks engine

Tasks:
- Implement checks for:
  - schema validity
  - required artifacts present
  - tests executed and pass/fail recorded
  - lint executed and pass/fail recorded
  - optional path/diff constraints

Repo paths:
- `core/accept/checks/`
- `schemas/v1/accept/`

Acceptance criteria:
- `accept_result.json` includes checks_run, checks_passed, failures, and reason codes.

### Story 5.3: CI-friendly reporting

Tasks:
- Add optional JUnit reporting.
- Keep JSON output as canonical source.

Repo paths:
- `core/accept/report/`
- `cmd/wrkr/accept.go`

Acceptance criteria:
- CI can fail deterministically based on stable exit code behavior.

### Story 5.4: GitHub-native summaries (jobpack -> deterministic markdown)

Tasks:
- Implement deterministic summary generator for GitHub surfaces:
  - final checkpoint summary
  - acceptance result pass/fail + top failures
  - artifact manifest deltas and key pointers
- Add CLI surface for summaries (either `wrkr report github <job_id|jobpack>` or a flag on `wrkr accept run --ci`).
- Write summary artifacts under `./wrkr-out/reports/` and support writing to `GITHUB_STEP_SUMMARY` when present.
- Add golden tests for markdown output and stable ordering.

Repo paths:
- `core/report/`
- `cmd/wrkr/report.go`
- `cmd/wrkr/accept.go`
- `docs/contracts/`

Acceptance criteria:
- The same jobpack produces identical markdown summary output across OS matrix.
- CI can publish the summary without requiring a hosted UI.

---

## Epic 6: CLI Surface, Wrap Mode, and Adapter Layer

Objective: make adoption low-friction and vendor-neutral from day one.

### Story 6.1: Full CLI command map

Tasks:
- Implement command tree:
  - `demo`, `init`, `submit`, `status`
  - `checkpoint list/show`
  - `pause`, `resume`, `cancel`
  - `approve`
  - `wrap`
  - `export`, `verify`
  - `accept init/run`
  - `report github`
  - `bridge work-item`
  - `serve`
  - `job inspect/diff`
  - `doctor`

Repo paths:
- `cmd/wrkr/`

Acceptance criteria:
- Command help and examples are complete and consistent.

### Story 6.2: Wrap mode (default adoption wedge)

Tasks:
- Implement `wrkr wrap -- <agent_command...>`.
- Capture bounded checkpoints and artifact refs by default.
- Emit jobpack and ticket footer.

Repo paths:
- `core/adapters/wrap/`
- `cmd/wrkr/wrap.go`

Acceptance criteria:
- Wrap mode works with at least one real CLI scenario and one fixture scenario.

### Story 6.3: Reference structured adapter

Tasks:
- Ship one structured adapter for coding-agent workflows.
- Adapter emits step events, tool-call counts, artifacts, and checkpoint context.
- Step events include explicit `executed` semantics so non-executed work is fail-closed and reviewable.

Repo paths:
- `core/adapters/reference/`
- `examples/integrations/reference/`

Acceptance criteria:
- Structured adapter path supports submit -> resume -> accept -> export flow.

### Story 6.4: Sidecar contract for non-Python stacks

Tasks:
- Implement canonical sidecar example that reads normalized request and invokes `wrkr` CLI.
- Keep sidecar as transport only with no embedded decision logic.
- Ensure sidecar artifacts and logs are written deterministically under `./wrkr-out/integrations/<lane>/...`.

Repo paths:
- `examples/sidecar/`
- `docs/integration_checklist.md`

Acceptance criteria:
- Sidecar flow is runnable offline using fixtures.

### Story 6.5: Python SDK thin wrapper

Tasks:
- Add `sdk/python/wrkr/` client and models for invoking CLI and parsing JSON.
- Keep SDK source-compatible and deterministic.

Repo paths:
- `sdk/python/wrkr/`
- `sdk/python/tests/`

Acceptance criteria:
- SDK can drive wrap/status/accept workflows without re-implementing Wrkr logic.

### Story 6.6: Checkpoint-to-work-item bridge (deterministic interrupts)

Tasks:
- Implement `wrkr bridge work-item <job_id> --checkpoint <id> [--dry-run]`.
- Convert `blocked` and `decision-needed` checkpoints into a stable payload with:
  - required action
  - reason codes
  - artifact pointers
  - next recommended commands (resume/approve/export/accept)
- Write outputs under `./wrkr-out/reports/` by default and keep stdout output bounded and stable.
- Add provider templates as purely presentational mappings (GitHub issue, Jira ticket) without embedding provider credentials in v1.

Repo paths:
- `core/bridge/`
- `schemas/v1/bridge/`
- `cmd/wrkr/bridge.go`
- `docs/contracts/`

Acceptance criteria:
- Identical inputs produce identical bridge payload bytes (golden tests).
- `--dry-run` never mutates state and clearly prints the next commands to run.

### Story 6.7: Local service mode (`wrkr serve`) with loopback-default hardening

Tasks:
- Implement `wrkr serve` exposing a minimal local HTTP API for:
  - submit/status
  - checkpoint list/show
  - approve
  - export/verify
  - accept/report summary
- Default bind is loopback only; non-loopback requires explicit flags including auth token and request-size limits.
- Ensure HTTP responses reuse existing schemas and the same error envelope semantics as the CLI (`--json`).
- Add hardening tests for auth enforcement, request limits, and path traversal attempts.

Repo paths:
- `core/serve/`
- `cmd/wrkr/serve.go`
- `docs/contracts/serve_api.md`
- `internal/integration/`

Acceptance criteria:
- `wrkr serve` is safe-by-default (no listener without explicit command; loopback-only by default).
- Non-loopback mode is guarded by mandatory auth and strict request limits.

---

## Epic 7: Documentation System and Docs Site (Mirror Standards)

Objective: replicate Gait-grade documentation quality, ownership, and discoverability.

### Story 7.1: Documentation map and ownership rules

Tasks:
- Create `docs/README.md` as authoritative docs map.
- Define ownership rules:
  - contracts are normative
  - README is onboarding only
  - runbooks are operational procedures
  - wiki is convenience, not authority

Repo paths:
- `docs/README.md`

Acceptance criteria:
- Every major topic has a single canonical home.

### Story 7.2: Core concept and architecture docs

Tasks:
- Add `docs/concepts/mental_model.md`, `docs/architecture.md`, `docs/flows.md`.
- Keep diagrams and flows aligned with command surface.

Acceptance criteria:
- Docs match implemented command behavior and artifact contracts.

### Story 7.3: Contract docs and conformance docs

Tasks:
- Add `docs/contracts/primitive_contract.md` for Wrkr primitives.
- Add contract docs for:
  - `./wrkr-out/` output layout (jobpacks, integrations, reports)
  - lease + heartbeat semantics
  - `environment_fingerprint` capture and resume gating
  - checkpoint protocol semantics (types, required fields, bounded summary rules)
  - jobpack contents and verify rules
  - acceptance result schema and exit code mapping
  - GitHub summary format (deterministic markdown)
  - work-item bridge payload contract
  - `wrkr serve` API surface (endpoints + OpenAPI file)
- Add conformance docs:
  - ticket footer + verify behavior
  - "Wrkr-compatible" lane definition and how conformance/parity are enforced in CI

Acceptance criteria:
- Contract docs are referenced from README and CI conformance scripts.

### Story 7.4: Operational runbooks and hardening docs

Tasks:
- Add runbooks:
  - integration checklist
  - blessed lane (worker boundary) integration kit
  - CI accept kit
  - GitHub Actions kit (verify/accept/report summary)
  - `wrkr serve` mode deployment + hardening
  - production defaults
  - runtime SLO
  - retention profiles
  - hardening release checklist

Repo paths:
- `docs/integration_checklist.md`
- `docs/ecosystem/blessed_lane.md`
- `docs/ecosystem/github_actions_kit.md`
- `docs/deployment/serve_mode.md`
- `docs/ci_regress_kit.md` (Wrkr naming may remain for compatibility)
- `docs/project_defaults.md`
- `docs/slo/runtime_slo.md`
- `docs/slo/retention_profiles.md`
- `docs/hardening/release_checklist.md`

Acceptance criteria:
- A new team can onboard without tribal knowledge.

### Story 7.5: Docs-site build and navigation parity

Tasks:
- Create static Next.js docs site (`docs-site/`) ingesting `docs/**`, `README.md`, `SECURITY.md`, `CONTRIBUTING.md`.
- Add navigation configuration and markdown rendering pipeline.
- Add docs workflow for GitHub Pages deploy.

Repo paths:
- `docs-site/`
- `docs-site/src/lib/navigation.ts`
- `.github/workflows/docs.yml`

Acceptance criteria:
- `docs-site` builds in CI and deploys from main.

### Story 7.6: Integration RFC templates and conformance kit docs

Tasks:
- Add an integration RFC template describing what a "lane" must specify:
  - adapter type (wrap/sidecar/structured)
  - deterministic `./wrkr-out/integrations/<lane>/...` outputs
  - checkpoint cadence and decision points
  - required artifacts and acceptance checks
  - security posture (what is captured, redaction rules)
- Add documentation for the "Wrkr-compatible" claim and the conformance/parity gates that enforce it.
- Add one worked example RFC for the blessed lane.

Repo paths:
- `docs/ecosystem/integration_rfc_template.md`
- `docs/contracts/wrkr_compatible.md`
- `docs/ecosystem/blessed_lane.md`

Acceptance criteria:
- New adapters/lanes can be proposed without ambiguity using the template.
- Conformance and parity scripts in CI reference the canonical docs.

---

## Epic 8: CI/CD Pipeline Parity (Fast, Mainline, Nightly, Release)

Objective: mirror Gait-grade CI discipline and throughput strategy.

### Story 8.1: Fast PR lane (`pr-fast.yml`)

Tasks:
- Add fast PR workflow with `lint-fast` and `test-fast` jobs.
- Keep runtime low and deterministic.

Repo paths:
- `.github/workflows/pr-fast.yml`

Acceptance criteria:
- PR fast checks are required and green for merge.

### Story 8.2: Mainline CI lane (`ci.yml`)

Tasks:
- Add broad CI jobs:
  - `lint` (linux + mac)
  - `test` (linux + mac + windows)
  - `e2e`
  - contracts, acceptance, install smoke, release smoke
  - docs-site and ui-local if applicable
- Add path filters for adoption-critical changes.

Repo paths:
- `.github/workflows/ci.yml`

Acceptance criteria:
- Mainline CI catches regression across contracts, acceptance, and integration paths.

### Story 8.3: Nightly lanes

Tasks:
- Add nightly workflows:
  - adoption-nightly
  - hardening-nightly
  - perf-nightly
  - windows-lint-nightly

Repo paths:
- `.github/workflows/adoption-nightly.yml`
- `.github/workflows/hardening-nightly.yml`
- `.github/workflows/perf-nightly.yml`
- `.github/workflows/windows-lint-nightly.yml`

Acceptance criteria:
- Nightly failures are artifact-rich and triageable.

### Story 8.4: Contract and conformance gates

Tasks:
- Add dedicated contract workflow checks for schema stability and consumer compatibility.
- Add ticket footer/verify conformance script.
- Add "Wrkr-compatible" conformance suite that validates end-to-end:
  - demo -> export -> verify -> accept -> report (GitHub summary)
  - deterministic `./wrkr-out/` outputs (jobpacks, reports, integrations fixtures)
  - stable exit codes and reason codes
- Add adapter parity harness asserting wrap vs sidecar vs structured adapter behavior matches for core contracts.
- Add service-mode hardening checks (`wrkr serve` loopback-default, non-loopback guardrails).

Repo paths:
- `scripts/test_contracts.sh`
- `scripts/test_ent_consumer_contract.sh`
- `scripts/test_ticket_footer_conformance.sh`
- `scripts/test_wrkr_compatible_conformance.sh`
- `scripts/test_github_summary_golden.sh`
- `scripts/test_serve_hardening.sh`
- `.github/workflows/ticket-footer-conformance.yml`
- `.github/workflows/wrkr-compatible-conformance.yml`

Acceptance criteria:
- Contract drifts fail CI deterministically.
- Conformance/parity failures produce artifact-rich logs and are triageable without a hosted UI.

### Story 8.5: Security scanning and code analysis

Tasks:
- Add a two-tier security scan model:
  - `make sast-fast` for local/pre-push checks (`gosec`, `govulncheck`, secret scan)
  - CI CodeQL for deep semantic analysis on PR, main, and nightly schedule
- Add `make codeql-local` (wrapper script) for optional local deep scan parity with CI.
- Upload SARIF artifacts in CI and document deterministic triage flow (new finding, baseline/suppression, owner action).
- Run `gosec` and `govulncheck` in lint or dedicated jobs.
- Add a lightweight license compliance check in CI (`make license-check`) suitable for OSS dependency hygiene.

Repo paths:
- `.github/workflows/codeql.yml`
- `Makefile`
- `scripts/run_codeql_local.sh`
- `docs/security/codeql_triage.md`
- `docs/security/license_policy.md`

Acceptance criteria:
- `sast-fast` runs locally with no cloud dependency and is pre-push friendly.
- CodeQL findings and triage artifacts are visible in CI and reproducible locally via `make codeql-local`.
- Critical scanner findings block release.

### Story 8.6: Workflow governance and required-check policy (single-admin friendly)

Tasks:
- Define required check policy for PR merge and document it in one place:
  - `pr-fast`
  - `ci`
  - contract/conformance checks
  - CodeQL
- Add a branch protection bootstrap script using `gh` CLI that a single admin can run idempotently (no two-admin assumptions).
- Enforce least-privilege `permissions:` in all workflows and restrict write scopes to jobs that need them.
- Pin third-party GitHub Actions by full commit SHA; add a lightweight update cadence doc and automation hook.
- Add workflow concurrency controls (`cancel-in-progress`) for PR jobs to reduce queue waste.
- Define cache strategy for Go/Python/docs-site jobs (stable keys + restore keys) and document cache busting rules.
- Add merge queue guidance as optional: enabled only when contributor volume justifies it; not required for single-maintainer operation.

Repo paths:
- `.github/workflows/*.yml`
- `.github/dependabot.yml`
- `scripts/bootstrap_branch_protection.sh`
- `docs/ci_required_checks.md`
- `docs/ci_permissions_model.md`
- `docs/ci_cache_strategy.md`

Acceptance criteria:
- Required checks are explicit and enforced without manual interpretation.
- A single admin can bootstrap and maintain branch protection from CLI.
- Workflow permissions and pinned actions are auditable and CI-reviewed.

---

## Epic 9: Hardening, Reliability, and Operability

Objective: enforce prime-time reliability and safe defaults as product requirements.

### Story 9.1: Production readiness doctor profile

Tasks:
- Implement `wrkr doctor --production-readiness` with checks for:
  - strict config profile
  - key source and signing mode
  - store/lock health
  - retention settings
  - unsafe defaults in production contexts

Repo paths:
- `cmd/wrkr/doctor.go`
- `core/doctor/`

Acceptance criteria:
- Doctor returns non-zero on critical readiness failures with actionable remediation.

### Story 9.2: Retention and lifecycle controls

Tasks:
- Add retention/rotation flags and prune reporting for job artifacts and logs.
- Add dry-run mode and deterministic reports.

Repo paths:
- `core/store/retention.go`
- `cmd/wrkr/store.go`

Acceptance criteria:
- Long-running deployments avoid unbounded artifact growth.

### Story 9.3: Chaos and stress suites

Tasks:
- Add chaos scripts for:
  - concurrent append corruption
  - session lock contention
  - oversized payload and malformed input handling
  - lease conflict, heartbeat expiry, and safe re-claim behavior
  - environment fingerprint mismatch gating and override audit trail
  - `wrkr serve` abuse cases (auth missing, non-loopback guardrails, request-size limits, path traversal)
  - trace/artifact uniqueness
- Add soak test for long-running sessions.

Repo paths:
- `scripts/test_chaos_*.sh`
- `scripts/test_session_soak.sh`
- `internal/integration/`

Acceptance criteria:
- Chaos suites are deterministic and release-blocking for hardening-critical paths.

### Story 9.4: Runtime SLO budgets and perf checks

Tasks:
- Define runtime SLO budgets for key commands.
- Add budget check scripts and benchmark gates.

Repo paths:
- `perf/runtime_slo_budgets.json`
- `perf/resource_budgets.json`
- `scripts/check_command_budgets.py`
- `scripts/check_resource_budgets.py`

Acceptance criteria:
- Budget regressions are detected in CI/nightly with actionable reports.

---

## Epic 10: Release Engineering and Supply Chain Integrity

Objective: ensure releases are reproducible, attestable, and operationally safe.

### Story 10.1: Goreleaser and multi-platform artifacts

Tasks:
- Add `.goreleaser.yaml` for linux/darwin/windows, amd64/arm64.
- Emit checksums and archives deterministically.

Repo paths:
- `.goreleaser.yaml`

Acceptance criteria:
- Tag `v*` produces expected artifacts.

### Story 10.2: Release workflow with integrity artifacts

Tasks:
- Add `release.yml` with:
  - acceptance gate before release
  - GoReleaser publish
  - SBOM generation
  - vulnerability scan artifact
  - cosign signing + provenance
  - release asset integrity verification

Repo paths:
- `.github/workflows/release.yml`

Acceptance criteria:
- Release workflow fails on missing or invalid integrity artifacts.

### Story 10.3: Homebrew tap publication (optional but mirrored)

Tasks:
- Add Homebrew formula render script from release checksums.
- Add publish workflow step with guarded token behavior.

Repo paths:
- `scripts/render_homebrew_formula.sh`
- `scripts/publish_homebrew_tap.sh`

Acceptance criteria:
- Formula asset is generated and publish path is testable.

### Story 10.4: Release runbooks and checklists

Tasks:
- Add release checklist documenting mandatory gates and sign-offs.

Repo paths:
- `docs/hardening/release_checklist.md`
- `docs/launch/github_release_template.md`

Acceptance criteria:
- Release decision process is explicit and auditable.

---

## Epic 11: Adoption Kit, UAT, and Distribution Readiness

Objective: make onboarding and operationalization reproducible in one lane.

### Story 11.1: Blessed lane integration kit

Tasks:
- Publish one canonical integration lane for v1:
  - coding-agent wrapper (wrap + reference structured adapter)
  - worker boundary wrapper/sidecar templates with deterministic outputs under `./wrkr-out/integrations/<lane>/...`
  - GitHub Actions kit for verify/accept/report summary (GitHub-native summaries via `GITHUB_STEP_SUMMARY`)
  - conformance script and CI template that makes "Wrkr-compatible" a testable claim
- Keep other adapters as parity references.

Repo paths:
- `docs/integration_checklist.md`
- `docs/ecosystem/blessed_lane.md`
- `docs/ecosystem/github_actions_kit.md`
- `examples/integrations/`
- `examples/ci/`
- `.github/workflows/adoption-regress-template.yml`

Acceptance criteria:
- New team completes first real job flow in under 15 minutes with docs only.

### Story 11.2: Adoption smoke and parity suites

Tasks:
- Add deterministic smoke suite validating:
  - demo output
  - verify output
  - submit/status/checkpoint flow
  - accept output
  - report output (GitHub summary)
  - `wrkr-out` output layout stability
  - bridge output (work-item payload)
  - serve mode smoke and hardening guardrails
  - wrap mode block/fail-safe behavior

Repo paths:
- `scripts/test_adoption_smoke.sh`
- `scripts/test_adapter_parity.sh`

Acceptance criteria:
- Smoke suite fails with explicit, stage-specific diagnostics.

### Story 11.3: UAT functional plan and automation

Tasks:
- Add `docs/uat_functional_plan.md` with command matrix.
- Add `scripts/test_uat_local.sh` and CI/nightly hooks.

Repo paths:
- `docs/uat_functional_plan.md`
- `scripts/test_uat_local.sh`

Acceptance criteria:
- UAT can be run end-to-end from clean checkout and outputs pass/fail summary.

### Story 11.4: Test cadence policy

Tasks:
- Add `docs/test_cadence.md` with fast/mainline/nightly expectations and enforcement policy.

Acceptance criteria:
- Contributors can choose the right lane quickly; release blockers are clear.

---

## Validation Plan (Local)

Run in order:

1. `make fmt`
2. `make lint-fast`
3. `make sast-fast`
4. `make test-fast`
5. `make lint`
6. `make license-check`
7. `make test`
8. `make test-e2e`
9. `make test-contracts`
10. `make test-acceptance`
11. `make test-conformance`
12. `make test-v1-acceptance`
13. `make test-runtime-slo`
14. `make test-hardening-acceptance`
15. `make test-adoption`
16. `make docs-site-build`
17. `make test-uat-local`

Optional deep local pass (before opening a high-risk PR):

18. `make codeql-local`

If any command fails, fix and rerun from failing step.

---

## Execution Order (Strict)

1. Land Epic 0 scaffold and developer guardrails.
2. Lock schema/contracts (Epic 1) before runtime business logic.
3. Build durable store/queue/recovery (Epic 2).
4. Implement checkpoint/budget/approval core (Epic 3).
5. Implement jobpack export/verify/inspect/diff (Epic 4).
6. Implement acceptance harness (Epic 5).
7. Complete CLI/wrap/adapter surfaces (Epic 6).
8. Land docs system and docs-site (Epic 7).
9. Land CI/CD parity lanes and conformance gates (Epic 8).
10. Land hardening and SLO enforcement (Epic 9).
11. Complete release engineering and supply chain hardening (Epic 10).
12. Complete adoption/UAT distribution assets (Epic 11).
13. Cut v1 only after all release-blocking exit criteria are green.

---

## Non-Goals (v1)

- Hosted dashboard or centralized control plane dependency.
- Fleet-wide enterprise RBAC/SSO as a requirement for OSS correctness.
- Model hosting or agent planning framework development.
- Non-deterministic default acceptance logic.

---

## Definition of Done (Applies to Every Story)

- Code is formatted and linted.
- Tests are added/updated and pass in local and CI lanes.
- Any new artifact schema has:
  - JSON schema under `schemas/v1/`
  - matching Go type under `core/schema/v1/`
  - validator and valid/invalid fixtures
- `--json` outputs and exit codes are stable and covered by tests.
- Commands remain offline-first unless explicitly documented otherwise.
- Docs are updated in canonical location and linked from `docs/README.md`.
- Conformance/parity suites and any golden outputs are updated for any contract change.
- No product logic is duplicated into SDK/adapters/skills.
- Security-sensitive changes include hardening docs and release checklist updates.
