# GAPS v1: Wrkr (OSS)

Date: 2026-02-13
Source references: `product/PRD.md`, `product/PLAN_v1.md`
Baseline branch: `master` (audited)
Scope: OSS v1 hard gaps only (correctness, coherence, leverage)

This document enumerates the remaining work required to make Wrkr v1 behavior match the product and plan claims with low reputational risk for public promotion.

---

## 1. Gap Severity Model

- `P0 Release-Blocking`: breaks core PRD/PLAN contract for durable long-running work.
- `P1 Promotion-Blocking`: technically works but creates expectation or trust risk for public launch messaging.
- `P2 Post-Launch`: useful and important, but not required for initial public MVP credibility.

---

## 2. Executive Gap Summary

Current Wrkr is strong on deterministic artifacts, verify, acceptance, and CI posture.
The biggest remaining gap is lifecycle correctness for true long-running dispatch semantics.

Top blockers:
- `P0`: resume does not continue structured adapter execution after approval.
- `P0`: JobSpec budgets are not automatically enforced in execution flow.
- `P0`: lease/heartbeat primitives are not wired into submit/resume execution lifecycle.
- `P0`: JobSpec environment fingerprint rules are not honored at init (default rules always used).
- `P1`: install/onboarding path is not explicit enough for PLG self-serve activation.
- `P1`: perf coverage is narrow (small command set and demo-sized artifacts only) versus required confidence for long-running OSS workloads.

---

## 3. Product-Contract Gaps (PRD/PLAN Alignment)

### 3.1 Gap G-01 (P0): Resume Continuation Semantics

Problem:
- `wrkr resume` moves state to `running` but does not continue remaining structured steps from last checkpoint.
- Observed behavior: checkpoint count unchanged after approval/resume in multi-step reference jobs.

Contract impact:
- Violates PRD Journey B and PLAN Story 6.3 acceptance expectation: submit -> resume -> accept -> export real continuation.

Required work:
- Introduce resumable execution cursor semantics for structured adapter jobs.
- Persist `next_step_index` (or equivalent) in durable state and event log.
- On `resume`, continue execution from first unexecuted step after decision checkpoint.
- Emit deterministic progress/completed checkpoints for resumed work.
- Preserve idempotency and fail-closed behavior for already-executed steps.

Repo paths:
- `core/adapters/reference/`
- `core/runner/`
- `core/store/`
- `core/schema/v1/`
- `cmd/wrkr/resume.go`

Acceptance criteria:
- After `decision-needed` approval and `resume`, remaining steps execute deterministically.
- Checkpoints and event ledger prove post-resume continuation.
- Repeated resume calls are idempotent.

---

### 3.2 Gap G-02 (P0): Budget Enforcement Wiring

Problem:
- Budget evaluator exists, but submit/adapter runtime does not automatically enforce JobSpec budgets.
- Budget checks currently rely on explicit `wrkr budget check` command path.

Contract impact:
- Violates PRD FR-8 budget stop conditions as a default runtime guardrail.

Required work:
- Wire JobSpec budgets into active execution loop for structured adapter and wrap mode where applicable.
- Increment counters (`step_count`, `tool_call_count`, `retry_count`) during execution and persist each update.
- Evaluate budgets before and after each step boundary; fail fast with deterministic blocked checkpoint.
- Ensure stop reasons and status transitions are consistent (`blocked_budget`, `E_BUDGET_EXCEEDED`).

Repo paths:
- `core/dispatch/`
- `core/adapters/reference/`
- `core/adapters/wrap/`
- `core/runner/`
- `core/budget/`
- `cmd/wrkr/submit.go`

Acceptance criteria:
- Jobs that exceed declared budgets stop automatically with deterministic reason code and checkpoint.
- No budget bypass in default submit path.

---

### 3.3 Gap G-03 (P0): Lease + Heartbeat Integration in Real Execution

Problem:
- Lease primitives are implemented and tested in isolation but not used by submit/resume runtime execution path.

Contract impact:
- PLAN Story 2.5 and v1 exit criteria require no double-execute semantics in actual lifecycle, not only primitive tests.

Required work:
- Acquire lease at execution start (`submit` and `resume`).
- Heartbeat lease during long-running step execution.
- Release/expire behavior must allow deterministic takeover after TTL.
- Reject conflicting active claims with `E_LEASE_CONFLICT` in active runtime flow.

Repo paths:
- `core/runner/`
- `core/dispatch/`
- `core/adapters/reference/`
- `cmd/wrkr/submit.go`
- `cmd/wrkr/resume.go`

Acceptance criteria:
- Two workers contending for same running job cannot both progress steps.
- Post-expiry takeover resumes correctly from durable cursor.

---

### 3.4 Gap G-04 (P0): Environment Fingerprint Rule Fidelity

Problem:
- Init captures default fingerprint rules, not JobSpec-declared rule set.

Contract impact:
- PRD/PLAN state that JobSpec governs fingerprint capture and resume gating policy.

Required work:
- Capture environment fingerprint using JobSpec rules at job initialization.
- Persist rules + hash + values as authoritative baseline.
- Keep override path behavior as-is (audited explicit override event).

Repo paths:
- `core/dispatch/submit.go`
- `core/runner/`
- `core/envfp/`
- `schemas/v1/jobspec/`

Acceptance criteria:
- Rule changes in JobSpec alter captured fingerprint deterministically.
- Resume mismatch decisions are based on the captured JobSpec rule set.

---

### 3.5 Gap G-05 (P1): Error Taxonomy Drift vs PRD Wording

Problem:
- PRD lists some reason codes that are not implemented in runtime taxonomy (`E_ACCEPT_LINT_FAIL`, `E_INVALID_INPUT` naming mismatch with `E_INVALID_INPUT_SCHEMA`).

Contract impact:
- Messaging and API contract ambiguity for integrators.

Required work:
- Normalize PRD + docs + schemas + runtime code to one authoritative reason-code set.
- Keep additive compatibility policy; avoid breaking existing code consumers.

Repo paths:
- `product/PRD.md`
- `docs/contracts/`
- `core/errors/errors.go`
- `schemas/v1/`

Acceptance criteria:
- One canonical reason-code table with no undocumented variants.
- Contract tests assert presence and exit-code mapping consistency.

---

## 4. OSS PLG Gaps (Bessemer 10 Principles Applied)

Reference: Bessemer Venture Partners, "The 10 laws of PLG" and its practical guidance.
- Source: https://www.bvp.com/atlas/the-10-laws-of-plg

### 4.1 Current Principle Fit (Directional)

1) End-user value first
- Status: Partial.
- Wrkr demo value is strong; first real-job continuation confidence is not yet strong enough.

2) Frictionless first experience
- Status: Partial.
- `wrkr demo` is good, but install path is not explicit in root onboarding.

3) Time-to-value in minutes
- Status: Partial.
- For advanced users yes; for net-new OSS users onboarding still depends on local build assumptions.

4) Product as primary growth engine
- Status: Moderate-strong.
- Wrap mode + jobpack + conformance are good product-led artifacts.

5) Build for habitual/retained usage
- Status: Partial.
- CI hooks exist; recurring operational loops need clearer default templates for teams.

6) Expansion from individual to team
- Status: Partial.
- Artifacts are shareable, but default team review workflow docs can be tighter.

7) Data-informed product iteration
- Status: Partial (OSS constrained).
- Local reports exist, but no explicit OSS telemetry-lite opt-in framework for activation/failure funnels.

8) Align monetization with delivered value
- Status: N/A for OSS core, but directional docs exist in PRD.

9) Cross-functional PLG operating model
- Status: Moderate.
- Repo quality is strong; product messaging must match actual lifecycle behavior.

10) Trust and reliability as growth multipliers
- Status: Partial.
- Deterministic artifacts are strong; long-running lifecycle correctness gaps weaken trust narrative.

### 4.2 Required OSS-Scoped PLG Work

#### Gap P-01 (P1): Install-to-first-win path is under-specified

Required work:
- Add explicit install matrix in root onboarding:
  - from source (`make build` + `./bin/wrkr`)
  - release binary download path
  - Homebrew path once release tap is live
- Add 60-second happy-path commands as copy/paste block.
- Add troubleshooting block for missing `wrkr` in PATH.

Repo paths:
- `README.md`
- `docs/launch/github_release_template.md`
- `docs/ecosystem/blessed_lane.md`

Acceptance criteria:
- Fresh user from clean machine reaches `wrkr demo` success without inference.

#### Gap P-02 (P1): Product messaging exceeds implemented lifecycle behavior

Required work:
- Temporarily tighten OSS messaging to "deterministic supervision + evidence-first durable state" until G-01..G-04 close.
- Add explicit "current execution semantics" note in docs to avoid overpromising.

Repo paths:
- `README.md`
- `docs/concepts/mental_model.md`
- `docs/flows.md`

Acceptance criteria:
- External copy does not promise behavior not yet wired end-to-end.

#### Gap P-03 (P2): OSS product analytics loop is missing

Required work (privacy-respecting, opt-in):
- Define local-only activation report command that summarizes:
  - time to first demo
  - time to first submit/export/verify/accept
  - failure code distribution from local runs
- Keep fully offline by default; no remote collection required.

Repo paths:
- `core/report/`
- `cmd/wrkr/report.go`
- `docs/`

Acceptance criteria:
- Maintainer can track PLG funnel quality from local and CI artifacts without hosted telemetry.

---

## 5. Performance and Reliability Coverage Gaps

## 5.1 What Exists Today

Current perf/SLO surface:
- Runtime command budget checks for 4 commands:
  - `version`, `doctor`, `demo`, `verify`
- Resource budgets:
  - binary size, demo jobpack size, demo store size, demo reports size
- Nightly + hardening suites run adoption/UAT/chaos/soak checks.

Implemented files:
- `perf/runtime_slo_budgets.json`
- `perf/resource_budgets.json`
- `scripts/check_command_budgets.py`
- `scripts/check_resource_budgets.py`
- `scripts/test_session_soak.sh`
- `scripts/test_chaos_*.sh`

## 5.2 Required Perf Coverage for an App Like Wrkr (OSS v1)

Wrkr needs perf confidence on the surfaces that define long-running job operability:
- submit latency under realistic JobSpec sizes.
- checkpoint emission throughput and store append latency under sustained load.
- recovery time from large event logs/snapshots.
- export/verify latency and memory for larger jobpacks (not only demo-sized packs).
- serve API request latency and concurrency under bounded load.
- contention behavior under concurrent writer scenarios with lease + CAS.
- soak stability over longer durations and larger step counts.

## 5.3 Perf Gaps to Close

### Gap PF-01 (P1): Runtime command perf coverage too narrow

Problem:
- No performance budgets for `submit`, `status`, `checkpoint list/show`, `approve`, `resume`, `accept run`, `export`, `report github`, `bridge work-item`.

Required work:
- Expand `runtime_slo_budgets.json` to include core lifecycle commands with realistic fixtures.
- Track p50/p95 over N runs, not single-run elapsed only.

Acceptance criteria:
- Runtime SLO report includes lifecycle-critical commands and percentile summary.

### Gap PF-02 (P1): Scale profile missing (large jobs/jobpacks)

Problem:
- Perf checks are demo-scale only.

Required work:
- Add fixture generators for:
  - high checkpoint counts
  - large events logs
  - larger artifacts manifest/jobpack metadata sizes
- Add export/verify/recover benchmarks over those fixtures.

Acceptance criteria:
- Defined max-supported OSS baseline profile with measurable pass/fail thresholds.

### Gap PF-03 (P1): Serve-mode performance is hardening-tested, not load-tested

Problem:
- Serve has security conformance but no throughput/latency targets.

Required work:
- Add lightweight serve load script (single-admin friendly) with bounded concurrent requests.
- Measure p95 latency and error rate for status/checkpoint endpoints.

Acceptance criteria:
- Serve perf report exists and has explicit budgets for local-loopback usage.

### Gap PF-04 (P2): Resource budgets omit memory/CPU peaks

Problem:
- Current resource checks are file-size focused only.

Required work:
- Add optional lightweight profiling capture for major commands in CI nightly:
  - peak RSS
  - CPU time
- Keep thresholds conservative and stable across CI variance.

Acceptance criteria:
- Resource report includes memory/CPU envelope for key commands.

### Gap PF-05 (P2): Coverage threshold governance mismatch

Problem:
- PLAN states >=85% Go+Python; enforced Go threshold is 63.0.

Required work:
- Reconcile policy and config:
  - either ratchet thresholds upward with staged plan
  - or update PLAN to current realistic baseline + ratchet schedule

Acceptance criteria:
- No contradiction between `product/PLAN_v1.md` and `perf/coverage_thresholds.json`.

---

## 6. Security, Release, and Trust Gaps

### 6.1 Gap S-01 (P1): Root security policy versioning mismatch

Problem:
- `SECURITY.md` still states pre-v1 best-effort posture.

Required work:
- Update security policy to v1 OSS posture and SLAs appropriate for single-admin OSS.

Acceptance criteria:
- Security policy language matches release maturity and CI/security gates.

### 6.2 Gap S-02 (P1): OSS licensing artifact missing

Problem:
- Root `LICENSE` file is missing.

Required work:
- Add explicit OSS license file aligned with project intent.

Acceptance criteria:
- Release artifacts and repository include unambiguous license declaration.

---

## 7. Cross-Platform and DX Gaps

### 7.1 Gap DX-01 (P1): Shell portability risks in runtime paths

Problem:
- Structured adapter and acceptance command checks use `sh -lc`, which is Unix-centric.

Required work:
- Define cross-platform command execution strategy:
  - either platform-specific shell selection
  - or explicit "Unix-only for command-executing adapter paths" in OSS v1 docs
- Strengthen Windows validation for relevant command surfaces.

Acceptance criteria:
- Platform support statement is explicit and matches tested behavior.

### 7.2 Gap DX-02 (P2): CLI ergonomics convention gap

Problem:
- `wrkr --help` is not supported; only `wrkr help`.

Required work:
- Add `--help` and `-h` global compatibility behavior.

Acceptance criteria:
- Conventional CLI help invocation works across command surfaces.

---

## 8. Workstreams and Sequencing (Strict)

## Workstream A (P0): Lifecycle Correctness

Includes:
- G-01 Resume continuation
- G-02 Budget enforcement wiring
- G-03 Lease integration
- G-04 Env fingerprint fidelity

Exit gate:
- Submit -> decision-needed -> approve -> resume -> complete remaining steps -> export -> verify -> accept passes deterministically in one automated scenario.

## Workstream B (P1): Promotion Safety

Includes:
- G-05 taxonomy normalization
- P-01 install path clarity
- P-02 messaging correctness
- S-01 security policy update
- S-02 license artifact
- PF-01 perf command coverage expansion
- PF-02 scale profile perf tests
- PF-03 serve perf envelope
- DX-01 platform support clarity

Exit gate:
- Public OSS MVP messaging matches measured behavior.

## Workstream C (P2): Post-Launch Hardening

Includes:
- P-03 OSS activation analytics artifacts
- PF-04 memory/CPU budget checks
- PF-05 coverage governance ratchet schedule
- DX-02 CLI ergonomics polish

Exit gate:
- Better operator insight and lower maintenance cost without widening product scope.

---

## 9. Validation Matrix for Gap Closure

For each closed gap, require:
- unit tests for affected module behavior.
- integration test in `internal/integration` for end-to-end correctness.
- conformance update where contract semantics changed.
- docs update under `docs/contracts` or runbooks.
- CI lane assignment (fast/mainline/nightly) with deterministic artifacts.

Release candidate gate after all `P0` + `P1`:
- `make test-v1-acceptance`
- `make test-adoption`
- `make test-uat-local`
- `make test-hardening-acceptance`
- expanded perf checks (runtime + scale + serve)
- `make codeql-local`
- docs-site lint/build

---

## 10. Out of Scope for This Gap Plan (v1 OSS)

- hosted dashboards
- multi-tenant fleet scheduler
- enterprise RBAC/SSO
- central telemetry backend
- non-deterministic rubric evaluators as default acceptance path
