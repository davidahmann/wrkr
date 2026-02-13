# ADR 0002: Append-Only Event Log with Snapshot Recovery

Date: 2026-02-13
Status: Accepted

## Context

Wrkr needs crash-tolerant local durability for long-running jobs without introducing a heavy external database dependency.

## Decision

Use a filesystem store at `~/.wrkr/jobs/<job_id>/` with:

- `events.jsonl` as an append-only event ledger.
- `snapshot.json` as latest materialized state.
- lock file (`append.lock`) using `O_EXCL` for single-writer appends.

Append semantics:

- events are written as one JSON object per line and `fsync`ed.
- trailing partial line is ignored on read to preserve committed history.

Recovery semantics:

- restore from snapshot, replay newer events by sequence.
- unknown event type is treated as corruption (`E_STORE_CORRUPT`).

## Consequences

- durable and simple local runtime behavior.
- deterministic replay for resume.
- no background daemon required for OSS local mode.
