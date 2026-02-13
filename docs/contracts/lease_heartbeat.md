# Lease and Heartbeat Contract

Wrkr runner supports lease ownership semantics for safe worker coordination.

## Lease Fields

- `worker_id`
- `lease_id`
- `expires_at`

## Behavior

- Acquire sets ownership for active execution.
- Heartbeat extends expiry for current owner.
- Conflicting lease acquisition returns deterministic conflict error.
