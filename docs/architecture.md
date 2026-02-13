# Wrkr Architecture

This document is the canonical architecture view for Wrkr OSS v1.

## Component Architecture

```mermaid
flowchart LR
    operator["Developer / Platform Engineer / CI"] --> cli["wrkr CLI (cmd/wrkr)"]

    subgraph core["Go Core (core/*)"]
        dispatch["Dispatch runtime (core/dispatch)"]
        store["Durable store (core/store)"]
        runner["Lifecycle engine (core/runner)"]
        adapters["Adapters (core/adapters)"]
        budget["Budget controls (core/budget)"]
        approve["Approval records (core/approve)"]
        pack["Jobpack export + verify (core/pack)"]
        accept["Acceptance harness (core/accept)"]
        bridge["Checkpoint bridge (core/bridge)"]
        serve["Local API surface (core/serve)"]
        schema["Schema + validation (core/schema)"]
        sign["Hash/sign utilities (core/sign, core/jcs, core/zipx)"]
    end

    cli --> dispatch
    dispatch --> runner
    runner --> store
    dispatch --> adapters
    adapters --> runner
    runner --> budget
    runner --> approve
    runner --> bridge
    dispatch --> pack
    dispatch --> accept
    cli --> serve
    dispatch --> schema
    pack --> schema
    accept --> schema
    pack --> sign

    subgraph artifacts["Artifact Surface (durable contract)"]
        jobpack["jobpack_<job_id>.zip"]
        checkpoints["checkpoints.jsonl"]
        events["events.jsonl"]
        acceptOut["accept_result.json + junit"]
        githubSummary["github_summary_<job_id>.json/.md"]
    end

    pack --> jobpack
    runner --> checkpoints
    store --> events
    accept --> acceptOut
    bridge --> githubSummary
```

## Runtime Boundaries

- Authoritative boundary: Go core owns status transitions, checkpoint semantics, budget/approval gates, export/verify integrity, and exit codes.
- Adoption boundary: wrappers/SDK integrations are transport layers and should not replace core state or contract logic.
- Durable contract boundary: schemas + persisted artifacts are the long-lived API, not in-memory structs.

## State and Persistence

- Default store root: `~/.wrkr/`
- Per-job state:
  - append-only event log (`events.jsonl`)
  - periodic snapshot (`snapshot.json`)
  - runtime execution cursor (`runtime_config.json`)
- Deterministic artifact root: `./wrkr-out/`
  - `jobpacks/`
  - `reports/`
  - `integrations/`

## Failure Posture

- Lease + heartbeat prevents concurrent double execution on a single job claim.
- Environment mismatch on resume blocks by default (`E_ENV_FINGERPRINT_MISMATCH`) unless explicitly overridden.
- Budget violations emit deterministic blocked checkpoints (`E_BUDGET_EXCEEDED`).
- Verify fails closed on hash mismatch, missing files, or undeclared entries.
- Serve defaults to loopback; non-loopback requires explicit hardening inputs.
