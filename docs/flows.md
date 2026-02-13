# Wrkr Flow Diagrams

This document is the canonical runtime flow reference for Wrkr OSS v1.

## 1) First-Win Flow (Demo -> Verify)

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant CLI as wrkr CLI
    participant Core as Dispatch Core
    participant FS as Local Filesystem

    Dev->>CLI: wrkr demo --json
    CLI->>Core: run deterministic demo job
    Core->>FS: write job state + jobpack
    Core-->>CLI: job_id + jobpack + footer
    CLI-->>Dev: first-win output
    Dev->>CLI: wrkr verify <job_id> --json
    CLI->>Core: verify manifest + hashes + schemas
    Core-->>CLI: deterministic verify result
    CLI-->>Dev: verify success/failure
```

Value: produces a verifiable artifact in under one minute with no external dependencies.

## 2) Structured Dispatch With Decision Checkpoint

```mermaid
sequenceDiagram
    participant Op as Operator
    participant CLI as wrkr CLI
    participant Dispatch as core/dispatch
    participant Runner as core/runner
    participant Store as core/store

    Op->>CLI: wrkr submit jobspec.yaml --job-id <id>
    CLI->>Dispatch: submit JobSpec
    Dispatch->>Runner: init + run structured adapter
    Runner->>Store: append events/checkpoints
    Runner-->>Dispatch: status=blocked_decision
    Dispatch-->>CLI: submit result
    CLI-->>Op: decision checkpoint visible
    Op->>CLI: wrkr approve <id> --checkpoint <cp>
    Op->>CLI: wrkr resume <id>
    CLI->>Dispatch: resume from durable cursor
    Dispatch->>Runner: continue remaining steps
    Runner->>Store: append completion checkpoints
    Runner-->>Dispatch: status=completed
    Dispatch-->>CLI: resume result
```

Rule: resume continues from persisted `next_step_index` and does not replay completed steps.

## 3) Budget Stop Condition

```mermaid
sequenceDiagram
    participant Adapter as Structured Adapter
    participant Runner as core/runner
    participant Budget as core/budget
    participant Store as core/store

    Adapter->>Runner: report step/call progress
    Runner->>Budget: evaluate limits vs usage
    alt budget exceeded
        Budget-->>Runner: exceeded + violations
        Runner->>Store: emit blocked checkpoint (E_BUDGET_EXCEEDED)
        Runner->>Store: set status=blocked_budget
    else budget available
        Budget-->>Runner: continue
    end
```

## 4) Wrap Adoption Flow

```mermaid
sequenceDiagram
    participant Eng as Engineer
    participant CLI as wrkr wrap
    participant Wrap as core/adapters/wrap
    participant Runner as core/runner
    participant Pack as core/pack

    Eng->>CLI: wrkr wrap -- <agent command>
    CLI->>Wrap: execute wrapped command
    Wrap->>Runner: emit plan/progress/completed or blocked
    Eng->>CLI: wrkr export <job_id>
    CLI->>Pack: assemble deterministic jobpack
    Eng->>CLI: wrkr verify <job_id|path>
```

Wrap gives zero-integration adoption and still lands on the same jobpack/verify contract.

## 5) Acceptance + CI Gate

```mermaid
sequenceDiagram
    participant CI as CI Workflow
    participant CLI as wrkr accept/report
    participant Accept as core/accept
    participant Pack as core/pack
    participant FS as wrkr-out/reports

    CI->>CLI: wrkr accept run <job_id> --config <accept.yaml> --ci
    CLI->>Accept: run deterministic checks
    Accept-->>CLI: accept_result + stable exit code
    CI->>CLI: wrkr report github <job_id>
    CLI->>Pack: read/export jobpack
    CLI->>FS: write github_summary_<job_id>.json/.md
    CLI-->>CI: machine-readable gate output
```

## 6) Local Serve API Transport

```mermaid
sequenceDiagram
    participant Client as Local Automation
    participant API as wrkr serve
    participant Dispatch as core/dispatch
    participant Runner as core/runner

    Client->>API: POST /v1/jobs:submit
    API->>Dispatch: submit jobspec path
    Dispatch-->>API: job status result
    Client->>API: GET /v1/jobs/<id>:status
    API->>Runner: recover status
    Runner-->>API: status response
```

Hardening rule: non-loopback serve requires explicit allow + auth token + body limit.
