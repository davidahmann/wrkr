# Architecture

Wrkr v1 is a buyer-hosted, single-binary CLI with local durable store and optional local HTTP service.

## Modules

- `cmd/wrkr`: command surface.
- `core/store`: append-only event log + snapshots.
- `core/runner`: lifecycle, checkpoint, resume, approval, budget enforcement.
- `core/adapters`: wrap and structured adapter implementations.
- `core/pack`: export/verify/inspect/diff and ticket footer.
- `core/accept`: deterministic acceptance harness.
- `core/bridge`: checkpoint interrupt payloads.
- `core/serve`: loopback-default API surface.
- `core/schema`: versioned contract types and validators.

## State and Durability

- Local default root: `~/.wrkr`.
- Per-job event log: append-only JSONL.
- Snapshots accelerate recovery and restart.
- Jobpack export is deterministic from persisted state.

## Security Posture

- No network listener by default.
- `wrkr serve` is explicit and loopback-only by default.
- Non-loopback requires explicit auth and request-size limits.
