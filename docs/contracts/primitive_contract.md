# Primitive Contract

Wrkr v1 primitives are stable contract surfaces.

## Dispatch Primitive

- Input: `JobSpec` (`wrkr.jobspec`).
- Output: deterministic job lifecycle state and checkpoints.

## Checkpoint Primitive

- Types: `plan`, `progress`, `decision-needed`, `blocked`, `completed`.
- Fields: summary, status, budget state, artifacts delta, optional required action, reason codes.

## Jobpack Primitive

- Portable artifact bundle containing job metadata, checkpoints, events, artifacts manifest, optional approvals and acceptance output.
- Verified with `wrkr verify`.

## Acceptance Primitive

- Deterministic checks and stable exit code mapping.
- CI-friendly outputs (JSON and optional JUnit).
