# Recovery and Resume Contract (v1)

Wrkr resume behavior is deterministic and based on durable state in the local store.

## Durable state components

- `events.jsonl`: append-only event log ordered by `seq`.
- `snapshot.json`: periodic materialized state with `last_seq`.

## Replay rules

1. Load `snapshot.json` if present; otherwise start from default job state.
2. Load `events.jsonl` and apply events with `seq > snapshot.last_seq`.
3. Ignore a trailing partial event line with no newline terminator.
4. Unknown event types are treated as store corruption and fail closed.

## Resume guarantees

- Counters (`retry_count`, `step_count`, `tool_call_count`) survive restarts.
- Idempotency keys survive restarts.
- Lease records survive restarts.
- Invalid queue transitions fail with `E_INVALID_STATE_TRANSITION`.
- Lease conflicts fail with `E_LEASE_CONFLICT`.

## Crash tolerance

An interrupted append may leave a partial final line, but previously committed events remain readable and valid.
