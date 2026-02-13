# PRD v1.0: Wrkr (OSS)

Working name: Wrkr
CLI: `wrkr`
Version: 1.0 (OSS v1)
Date: February 13, 2026
Owner: Product + Platform
Status: Execution-ready PRD

---

## 1. One-paragraph summary

Wrkr turns agents into durable workers. You submit a multi-hour job; Wrkr runs it with durable state, structured checkpoints, budgets, and resumability, and produces a portable `jobpack_<job_id>.zip` with acceptance results so humans can approve outcomes quickly without babysitting a chat.

Wrkr is a dispatch + supervision substrate for long-running agent jobs, not an agent framework and not a tool-boundary policy engine.

Shareable artifact: `job_id` + verifiable `jobpack_<job_id>.zip` with a one-line ticket footer that teams paste into PRs/incidents.

Wrkr's wedge is a durable job contract plus a portable evidence bundle. If `jobpack` becomes the default unit of review, Wrkr becomes infrastructure, not a feature.

---

## 2. Problem and hair-on-fire triggers

### Problem statement

Long-running agent work fails for systems reasons:
- Runs are not durable: crashes or interruptions lose state; resumes are unsafe.
- Supervision does not scale: humans must read a stream to detect drift.
- Costs/time explode: no budgets or deterministic stop conditions.
- Outputs are hard to accept: missing artifacts, unclear diffs, unknown tests, inconsistent evidence.

### Hair-on-fire triggers (concrete incidents)

Incident A: Half-finished refactor
- A coding agent runs 3 hours, hits a dependency error, exits. There is no safe resume; the human cannot reconstruct what changed and what remains.

Incident B: Silent waste
- The job loops (bad plan, unstable tool) for hours. No budget ceiling exists; costs spike.

Incident C: Unreviewable completion
- The agent "finishes" but there is no artifact manifest, no acceptance signal, no clear diff summary. Review costs exceed build costs.

### Why current alternatives fail

Chat-first agent UIs:
- Human attention is the bottleneck; review is unstructured and noisy.

Agent frameworks:
- Great for orchestration logic, but typically do not ship production-grade durability primitives and acceptance/evidence contracts.

Generic workflow orchestrators:
- Durable, but not agent-native: they do not standardize checkpoints, human decision interrupts, or jobpack/acceptance artifacts.

---

## 3. Product thesis and positioning

### Positioning

Wrkr is:
- "Dispatch and supervision for long-running agent jobs."

Wrkr is not:
- an agent planning/orchestration framework
- a hosted dashboard requirement
- a tool-boundary policy engine

### Product thesis

The wedge is artifact-first supervision:
- A neutral dispatch contract (JobSpec in, checkpoints/artifacts/jobpack out).
- Noise-killer checkpoints (bounded summaries, deltas, explicit decisions).
- Deterministic budgets and stop reasons.
- Deterministic acceptance harness as the default review signal.

Wrkr wins when review becomes fast and safe:
- A manager/lead can approve a PR-sized change by reading the final checkpoint + acceptance summary, not a full transcript.

### Defensibility (why this stays independent)

Wrkr assumes the real world is multi-agent and multi-surface.

Even if agent products and frameworks ship native budgets and checkpointing, teams still need:
- A vendor-neutral, portable artifact (`jobpack`) to review and attest in GitHub and CI.
- Consistent stop reasons, acceptance signals, and evidence formats across tools.
- A unit of review that outlives any single runtime UI.

Wrkr's defensibility is not that it has budgets or checkpoints; it's that it standardizes the job evidence contract and ships the conformance kit + distribution surfaces that make it ubiquitous.

### What Wrkr standardizes (the contract surface)

Wrkr treats these as the public API surface (versioned, stable, additive within `v1.x`):
- JobSpec schema.
- Checkpoint schema and semantics.
- Jobpack contents plus hash/verify rules.
- AcceptResult schema plus exit code contract.
- Ticket footer format.
- Failure taxonomy (reason codes) and status model.

Wrkr also ships a conformance suite that validates contract stability and end-to-end determinism (demo -> verify -> accept -> export).

### Forcing function (why teams adopt)

Wrkr is designed to ride GitHub review economics:
- Large agent changes are expensive to review unless there is a bounded summary and deterministic acceptance signal.
- Jobpacks become the pasteable unit of evidence in PRs and incidents.
- CI can verify and accept or fail fast without a hosted dashboard.

### Competitive landscape (and differentiation)

Wrkr is not trying to out-orchestrate orchestrators or out-chat chat UIs. It sits below them.

Adjacent categories and how Wrkr differs:
- Durable execution engines (Temporal/Inngest/Restate): provide durability primitives; Wrkr provides agent-native checkpoints, acceptance artifacts, and a portable evidence contract.
- Agent frameworks (LangGraph/CrewAI/AutoGen-like): provide planning and workflow composition; Wrkr provides durable dispatch and review artifacts across frameworks.
- Vertically integrated agent products (Devin/Codex/Claude Code/Cursor tasks): can ship their own budgets/checkpoints; Wrkr provides a neutral jobpack and CI acceptance contract that survives vendor diversity.
- Tracing and eval platforms: excellent for observability; Wrkr produces deterministic acceptance outputs and verifiable jobpacks as the unit of review.
- CI tooling: can run checks; Wrkr defines what "done" means for long-running agent jobs and produces the artifacts and exit codes CI needs.

---

## 4. Personas and JTBD

### Persona 1: Agent developer (shipping long-running work)

JTBD:
- When my jobs take hours, I need durable execution and structured checkpoints so I can resume safely and review outcomes quickly.

Adoption blockers:
- Requires rewriting my agent stack.
- Requires a hosted control plane.

### Persona 2: Platform engineer (runtime reliability owner)

JTBD:
- When teams dispatch agents to do real work, I need standard lifecycle controls (budgets, retries, resumability) and predictable failure modes.

Adoption blockers:
- Heavy infra requirements.
- Non-deterministic outputs that break automation.

### Persona 3: Tech lead / manager (review bottleneck)

JTBD:
- When agents produce big changes, I need high-signal checkpoints and acceptance results so I can approve quickly.

Adoption blockers:
- Too many notifications.
- Outputs are not verifiable or complete.

### Persona 4: DevEx (rollout owner)

JTBD:
- When multiple teams use different agents, I need a consistent job interface that integrates with GitHub/CI and stays vendor-neutral.

Adoption blockers:
- Vendor lock-in.
- No standard artifact contract.

v1 priority:
- Agent developer + platform engineer durability loop.
- Tech lead review loop via acceptance + jobpack.
- DevEx distribution via CLI + GitHub Actions.

---

## 5. Goals, non-goals, and scope

### Goals (v1.0)

- Durable long-running jobs:
  - crash-safe state store
  - pause/resume/cancel
  - deterministic status and stop reasons
  - lease + heartbeat semantics for safe job claims and crash recovery
- Structured supervision:
  - typed checkpoints with bounded summaries
  - explicit "decision-needed" interrupts with approvals
- Budget enforcement:
  - time, steps/tool-calls (adapter-reported), retries
  - deterministic budget-exceeded behavior
- Evidence artifact:
  - deterministic jobpack export
  - offline verify
  - stable ticket footer
  - deterministic output paths under `./wrkr-out/` (jobpacks, integration artifacts, reports)
- Acceptance harness:
  - deterministic checks first (schemas, artifacts present, tests/lint executed, diff/path rules)
  - CI-friendly outputs (JSON required, JUnit optional)
- Distribution-first:
  - CLI is the product surface
  - GitHub Actions kit for verify/accept/report and GitHub-native summaries
  - "wrap anything" adoption path
  - blessed integration lane kits (worker boundary wrapper + CI template)
  - conformance + parity suites for "Wrkr-compatible" integrations

### Explicit non-goals (v1.0)

- Hosted dashboards or multi-tenant control plane.
- Managing enterprise secrets, permissions, or policy governance.
- Building models or hosting inference.
- Becoming an agent framework.

### Philosophy

- Artifacts and schemas are the API surface.
- Determinism and offline verification are default.
- Integration must start with near-zero friction.

---

## 6. Core primitives (Wrkr contracts)

Wrkr standardizes four durable primitives and one adoption surface.

### 6.1 JobSpec (input contract)

Purpose: describe a job in a structured, portable way.

JobSpec requirements:
- `name`, `objective`
- `inputs` (workspace/repo path, references)
- `expected_artifacts` (paths/patterns)
- `adapter` selection and config
- `budgets` (wall time, max retries, max step/tool-call count; optional cost/tokens)
- `checkpoint_policy` (minimum frequency, mandatory decision points)
- `acceptance` reference (optional)
- `environment_fingerprint` capture rules

Principle: JobSpec is immutable once submitted; any change is a new job.

Resume safety:
- On resume, Wrkr evaluates environment compatibility using the captured `environment_fingerprint`.
- If the fingerprint is incompatible, Wrkr must emit a `blocked` checkpoint with `reason_codes=[E_ENV_FINGERPRINT_MISMATCH]` and require explicit operator action (override or new job submission).

### 6.2 Checkpoint (supervision contract)

Purpose: provide a bounded, reviewable state transition.

Checkpoint types (v1.0):
- `plan`
- `progress`
- `decision-needed` (blocks until approval)
- `blocked`
- `completed`

Checkpoint fields (minimum):
- `checkpoint_id`, `type`, `created_at`
- bounded `summary`
- `status` (job status at checkpoint)
- `budget_state` (counters + remaining)
- `artifacts_delta` (added/changed/removed pointers)
- `required_action` (for decision-needed)
- `reason_codes` (stable taxonomy)

Explicit constraint:
- Wrkr does not store or surface raw chain-of-thought as a product surface.

### 6.3 Budget ledger (stop-condition contract)

Budgets enforced:
- `max_wall_time`
- `max_step_count`
- `max_tool_calls` (adapter-reported)
- `max_retries`
- optional: `max_estimated_cost`, `max_tokens` (adapter-provided)

Behavior:
- On exceed: emit checkpoint `type=blocked`, `reason_codes=[E_BUDGET_EXCEEDED]`, set status `blocked_budget`.

### 6.4 Acceptance result (review contract)

Purpose: produce a deterministic pass/fail (plus details) suitable for CI.

Acceptance checks (deterministic-first):
- schema validity (JobSpec/checkpoints/jobpack)
- required artifacts present
- tests executed and recorded (pass/fail)
- lint executed and recorded (pass/fail)
- diff/path constraints (optional)

Outputs:
- `accept_result.json` (required)
- optional JUnit

### 6.5 Jobpack (portable evidence artifact)

Purpose: a verifiable, portable bundle for review, replay of evidence, and acceptance.

Jobpack is a deterministic zip export.

Wrkr standardizes deterministic output paths (v1.0 default) so docs, CI, and bug reports can rely on stable locations:
- `./wrkr-out/jobpacks/jobpack_<job_id>.zip` (default export target)
- `./wrkr-out/integrations/<lane>/...` (integration kit outputs)
- `./wrkr-out/reports/...` (acceptance and summary artifacts)

Required contents (v1.0):
- `manifest.json` (schema versions, tool versions, hashes)
- `job.json` (JobSpec + derived metadata)
- `events.jsonl` (append-only event ledger projection)
- `checkpoints.jsonl` (ordered)
- `artifacts_manifest.json` (hashes + capture mode + pointers)
- `accept/accept_result.json` (if acceptance run)
- `approvals.jsonl` (if approvals occurred)
- `verify.json` (optional cached verify output)

Ticket footer (stable):
- `WRKR job_id=<job_id> manifest=sha256:<hash> verify="wrkr verify <job_id>"`

Integrity verification:
- `wrkr verify <job_id|path>` validates hashes and schemas.

Event ledger semantics (v1.0):
- `events.jsonl` is an append-only sequence of step-attempt records.
- Each record must support a stable `executed` (or equivalent) field so wrapper/sidecar/adapter integrations can be fail-closed consistently when a step is not allowed to run.

### 6.6 Wrap mode (adoption surface)

Purpose: near-zero integration to get durability + jobpack + acceptance without writing an adapter.

`wrkr wrap -- <agent_command...>`:
- runs an arbitrary agent CLI command
- captures produced artifacts (reference-only by default)
- emits checkpoints and jobpack
- allows teams to adopt Wrkr before building deeper adapters

---

## 7. User journeys (v1.0)

### Journey A: First 60 seconds (offline demo)

Commands:
- `wrkr demo`
- `wrkr verify <job_id|path>`

Success criteria:
- completes offline in <60 seconds
- emits 3 checkpoints (plan/progress/completed)
- produces jobpack + ticket footer

### Journey B: First real job (repo refactor)

Commands:
- `wrkr init jobspec.yaml`
- `wrkr submit jobspec.yaml`
- `wrkr status <job_id>`
- `wrkr checkpoint list <job_id>`
- `wrkr checkpoint show <job_id> <checkpoint_id>`
- `wrkr approve <job_id> --checkpoint <checkpoint_id> --reason "..."`
- `wrkr resume <job_id>`
- `wrkr export <job_id>`
- `wrkr accept run <job_id> --ci`

Success criteria:
- crash/restart resumes without losing committed progress
- pauses only at decision-needed checkpoints
- acceptance yields deterministic pass/fail

### Journey C: CI enforcement (GitHub Actions)

Flow:
- verify jobpack
- run acceptance
- attach summary + artifacts

Success criteria:
- CI fails fast on schema/artifact/test failures
- stable exit codes; JSON outputs suitable for workflows

### Journey D: Worker boundary integration (Gas Town-like workloads)

Entry point:
- A worker/orchestrator executes many actions and needs durable job dispatch, resumability, and portable jobpacks for review.

Two integration options (v1):
- CLI wrapper/sidecar: worker emits JobSpec and calls `wrkr submit/status/checkpoint/approve/export` via subprocess.
- Optional local service: worker calls a loopback `wrkr serve` endpoint for submit/status/approve/export without embedding Wrkr logic.

Success criteria:
- Integrations write deterministic artifacts under `./wrkr-out/integrations/<lane>/...`.
- Non-executed steps are explicit (`executed=false` semantics), enabling fail-closed behavior.
- Jobpacks remain verifiable offline and reviewable in GitHub/CI.

---

## 8. Functional requirements (FR)

FR-1: Single-binary CLI
- Go static binary for macOS/Linux/Windows
- deterministic outputs by default

FR-2: Offline demo
- `wrkr demo` works offline, <60 seconds
- produces jobpack + footer line

FR-3: JobSpec init
- `wrkr init` generates a minimal `jobspec.yaml` with comments and safe defaults

FR-4: Submit + queue
- `wrkr submit` returns `job_id`
- job is queued and runnable by local worker

FR-5: Durable execution
- append-only event log persisted after each step
- restart recovery without losing committed work
- idempotency keys per step

FR-6: Checkpointing
- emits typed checkpoints (plan/progress/decision-needed/blocked/completed)
- each includes bounded summary + artifact delta + budget state

FR-7: Pause/resume/cancel
- `wrkr pause|resume|cancel <job_id>`

FR-8: Budgets + stop reasons
- enforce wall time, retries, step/tool-call count
- deterministic status transitions and reason codes

FR-9: Approval flow
- decision-needed blocks forward progress until approval is recorded
- `wrkr approve <job_id> --checkpoint <id> --reason <text>`

FR-10: Jobpack export + verify
- deterministic export and offline verify

FR-11: Status + monitoring
- `wrkr status <job_id>` yields bounded summary
- `--json` stable machine output

FR-12: Acceptance harness
- `wrkr accept init` creates `accept.yaml`
- `wrkr accept run <job_id>` executes deterministic checks

FR-13: Adapter interface
- adapter reports steps, tool-call count, produced artifacts
- v1 ships:
  - wrap adapter (unstructured)
  - one reference structured adapter (for a coding-agent workflow)

FR-14: UX contracts
- `--json` for major commands
- stable exit codes
- `--explain` prints short command intent

FR-15: Deterministic integration artifact paths
- Standardize `./wrkr-out/` output layout (jobpacks, integrations, reports) and keep it stable across releases.

FR-16: Adapter parity and adoption smoke suites
- Ship deterministic scripts and CI lanes that assert parity across `wrap`, sidecar, and the reference adapter.
- Parity must include: budgets and stop reasons, checkpoint types, jobpack export/verify, acceptance outputs, and exit codes.

FR-17: Checkpoint-to-work-item bridge
- Provide a deterministic bridge that converts `blocked` or `decision-needed` checkpoints into work item payloads (GitHub Issue/Jira/Beads-like).
- Always support `--dry-run` and include next commands and artifact pointers.

FR-18: Optional local service mode
- `wrkr serve` provides a minimal local HTTP surface for submit/status/checkpoint/approve/export.
- Service defaults to loopback bind; non-loopback requires explicit auth and request-size limits.

FR-19: GitHub-native job summaries
- Generate deterministic PR/check summaries from jobpacks (final checkpoint summary, acceptance result, artifact manifest deltas).

FR-20: Integration RFC templates and conformance kit
- Publish a "Wrkr-compatible" conformance doc + script that validates end-to-end expectations for blessed lanes and prevents contract drift.

---

## 9. Non-functional requirements (NFR)

NFR-1: Determinism
- jobpack export and verify are deterministic
- schema IDs and versions are stable

NFR-2: Offline-first
- demo, verify, export, inspect, and acceptance reading work offline

NFR-3: Reliability
- crash-safe store; no corruption on crash
- explicit recovery steps for store issues

NFR-4: Performance
- low overhead per step/checkpoint
- export/verify completes in reasonable time for typical laptop workloads

NFR-5: Security posture
- signed releases, checksums, SBOM
- no network listener by default; any service mode is explicit and must be loopback-default with strict non-loopback hardening (auth + request limits)

NFR-6: Portability
- single binary
- adapter model keeps vendor neutrality

NFR-7: Privacy
- artifact capture default is reference-only
- raw artifact capture requires explicit opt-in
- redaction metadata supported

---

## 10. Status model, exit codes, and failure taxonomy

### 10.1 Job statuses (stable)
- `queued`
- `running`
- `paused`
- `blocked_decision`
- `blocked_budget`
- `blocked_error`
- `completed`
- `canceled`

### 10.2 Exit codes (contract)
- `0` success
- `2` verification failed
- `4` approval required
- `5` acceptance failed
- `6` invalid input/schema
- `8` unsafe operation attempted without explicit flag
- `1` generic failure

### 10.3 Reason codes (minimum v1.0 set)
- `E_BUDGET_EXCEEDED`
- `E_ADAPTER_FAIL`
- `E_CHECKPOINT_APPROVAL_REQUIRED`
- `E_ACCEPT_MISSING_ARTIFACT`
- `E_ACCEPT_TEST_FAIL`
- `E_VERIFY_HASH_MISMATCH`
- `E_STORE_CORRUPT`
- `E_ENV_FINGERPRINT_MISMATCH`
- `E_LEASE_CONFLICT`
- `E_INVALID_STATE_TRANSITION`
- `E_INVALID_INPUT_SCHEMA`
- `E_UNSAFE_OPERATION`

---

## 11. Architecture (v1.0)

### 11.1 High-level

Buyer-hosted by default. Single-binary CLI plus pluggable adapters.

Default local store:
- filesystem under `~/.wrkr/`
- append-only event log + snapshots
- deterministic export from store to jobpack zip

Default working artifacts:
- `./wrkr-out/` for demo outputs, integration artifacts, and jobpack exports

### 11.2 Go module layout (target)

- `cmd/wrkr`: CLI entrypoint
- `core/schema`: versioned schemas for JobSpec/checkpoint/jobpack/accept
- `core/store`: durable local state store (log + snapshot + locks)
- `core/runner`: job execution engine, checkpoint emission, recovery
- `core/budget`: budget enforcement
- `core/approve`: approval tokens and recording
- `core/pack`: jobpack assembly and verification
- `core/accept`: acceptance harness
- `core/adapters`: adapter interface + reference adapters
- `core/doctor`: install + environment diagnostics
- `core/fsx`, `core/zipx`, `core/jcs`: deterministic IO, zip, canonicalization utilities

### 11.3 Runtime boundaries

Authoritative core:
- schemas, canonicalization, hashing, verification, durable store, exit codes

Adoption surfaces:
- wrap mode, SDKs, sidecars, GH Actions are transport layers that call CLI and parse `--json`
- optional `wrkr serve` is a local transport surface for systems that prefer HTTP over subprocess integration

---

## 12. Distribution surfaces (OSS)

### 12.1 CLI is the product

Wrkr must be fully useful without any hosted service.

### 12.2 GitHub Actions kit

Provide a default workflow that:
- verifies jobpacks
- runs acceptance
- uploads artifacts
- prints a high-signal job summary and GitHub-native check/PR outputs

### 12.3 Skills (thin wrappers)

Ship optional “skills” for popular agent shells (Codex/Claude/Cursor) that:
- call `wrkr` commands with `--json`
- never implement product logic outside the CLI

### 12.4 Integration kits and conformance

Publish a single blessed lane plus parity/conformance assets:
- Worker boundary wrapper/sidecar templates with deterministic artifact paths (`./wrkr-out/integrations/<lane>/...`).
- RFC templates for proposing new integrations without fragmenting behavior.
- A conformance script that proves compatibility end-to-end (demo -> verify -> accept -> export) and is CI-enforced.

### 12.5 Optional local service

Provide `wrkr serve` for environments that cannot shell out cleanly or want a stable loopback integration surface.

Constraints:
- loopback bind by default
- explicit auth required for non-loopback
- max request bytes and retention/rotation controls
- deterministic response payloads and stable status mapping

---

## 13. Acceptance criteria (v1.0)

First win:
- `wrkr demo` completes offline in <60 seconds and produces a verifiable jobpack

Durability:
- job resumes after forced restart without losing committed progress
- append-only store survives crash without corruption

Approvals:
- decision-needed checkpoint blocks until approval is recorded

Budgets:
- budget ceilings enforce deterministically with stable status and reason codes

Jobpack:
- export is verifiable offline; tampering is detected

Acceptance harness:
- deterministic checks run locally and in CI with stable exit codes and JSON outputs

UX:
- major commands support `--json` and bounded human summaries

---

## 14. Success metrics (v1.0)

Activation:
- time from install to `wrkr demo` success <= 5 minutes
- time from install to first real job acceptance <= 15 minutes (blessed lane)

Reliability:
- induced crash/resume succeeds >95% for reference adapters

Supervision efficiency:
- median review time for PR-sized job output <= 5 minutes
- decision-needed interrupts are rare and high-signal (<3 per job)

Cost control:
- budget ceilings prevent unbounded runs

---

## 15. Risks and mitigations

Risk: scope gravity into "agent platform"
- Mitigation: stay contract-first (jobpack, checkpoints, acceptance). No planning framework.

Risk: feature absorption by incumbents
- Mitigation: win the portable evidence contract: jobpack spec + conformance suite + GitHub/CI distribution + cross-vendor wrap/adapters.

Risk: standards plays are hard to win
- Mitigation: ship forcing functions (wrap, CI kit, ticket footer) and enforce conformance gates so "compatible" has a real meaning.

Risk: category/budget line risk
- Mitigation: wedge into existing budgets and workflows (CI gates, code review time, incident response) rather than selling a new dashboard category.

Risk: adoption friction (JobSpec/adapters)
- Mitigation: `wrkr wrap` is the default wedge; ship templates and one blessed lane.

Risk: resume is brittle for real agents
- Mitigation: strict checkpoint context contract; reference adapters prove reliability; idempotency keys per step.

Risk: acceptance harness becomes subjective
- Mitigation: deterministic checks first; rubric hooks are opt-in and later.

Risk: service-mode misuse or exposure
- Mitigation: loopback-default, auth for non-loopback, request-size limits, retention controls, and a strict documented contract for response semantics.

---

## 16. Open questions

- Minimum viable checkpoint context schema that enables reliable resumes across agent types.
- Artifact capture modes that balance privacy with verifiability.
- Best GitHub UX: PR comment summaries vs check runs vs both.
- Canonical definition of “tool call count” across vendors/frameworks.
- Whether to support workspace snapshot/patch capture in v1.0 or in v1.x ladder.
- Should `wrkr serve` map non-executable states to non-2xx status codes by default, or preserve a compat 200-only mode for easier client integration?
- What are the default lease TTL and heartbeat intervals that avoid double execution while keeping UX smooth on developer machines?

---

## 17. OSS ladder (post-v1.0 compounding plan)

This section is the "Gait-level" compounding ladder translated to Wrkr primitives. v1.0 is shippable without completing every rung; each rung must preserve core contracts.

### Ladder principles
- Preserve contracts: schemas + exit codes are append-only within v1.x.
- Stay artifact-first: do not introduce dashboard dependencies.
- One-lane adoption: keep one blessed lane perfect before expanding.
- Hardening is product: durability, contention, retention, and diagnostics are release-blocking.

### v1.1: Contract lock + conformance kit
- Publish normative contract docs for:
  - JobSpec, Checkpoint, Jobpack, AcceptResult
- Add compatibility guards and golden fixtures.
- Add `wrkr job inspect` timeline and `wrkr job diff`.
- Add a conformance script that validates:
  - schema stability
  - ticket footer format
  - verify determinism

### v1.2: Integration friction to near-zero
- Ship one canonical wrapper pattern and one sidecar pattern.
- Ship one CI kit and deprecate duplicates.
- Add adapter parity harness (wrap vs reference adapter behavior).

### v1.3: Long-running session checkpoint chains
- Append-only session journals with incremental sealed checkpoint jobpacks.
- `wrkr verify job-chain` for linked checkpoints.
- Resume contracts and crash recovery become soak-tested.

### v1.4: Prime-time hardening
- Concurrency-safe append utilities.
- Lock contention strategy + budgets.
- Artifact retention/rotation.
- Production-readiness `wrkr doctor --production-readiness`.
- Chaos/soak gates become release-blocking.

### v1.5: Acceptance and review UX lift
- Acceptance policy simulation (“what would fail if we changed checks?”).
- Deterministic minimization: reduce jobpack to minimal failing predicate.
- GitHub Actions workflow hardened as reusable template; job summary surfaces key deltas.

### v1.6: Adoption proof packaging
- "15-minute onboarding" proof scripts and scorecards.
- Adoption proof bundle generation (jobpack + verify + accept + CI outputs).
- Blessed lane selection governance for any new official adapter lane.

### v2.0+ (deferred, optional)
- Fleet scheduling, org workflows, central artifact indexing, RBAC/SSO.
- These remain out of OSS correctness path; OSS artifacts remain the integration contract.

---

## 18. Commercial capture (directional, not required for OSS v1 correctness)

Wrkr can be a complete OSS tool and still support a clear enterprise layer:
- Indexing and search across jobpacks, checkpoints, and acceptance results.
- Org retention policies, encryption, and KMS/HSM integration for evidence.
- RBAC for approvals and job lifecycle controls at org scale.
- Organization-wide acceptance profiles (required checks and artifact rules).
- Connectors (GitHub/GitLab/Jira/Slack/SIEM) that operate on jobpacks as the unit.
- Fleet scheduling and execution for distributed runners.

Principle: any enterprise layer must consume the same OSS jobpack contracts; it must not be required to create or verify jobpacks.
