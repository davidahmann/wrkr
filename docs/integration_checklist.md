# Integration Checklist (v1)

Use this checklist when integrating Wrkr via wrap mode, reference adapter, or sidecar.

## 1. Contract Fit (Required)

- Define one lane name and keep outputs under `./wrkr-out/integrations/<lane>/...`.
- Emit only canonical checkpoint types:
  - `plan`
  - `progress`
  - `decision-needed`
  - `blocked`
  - `completed`
- Ensure every `decision-needed` checkpoint has explicit `required_action`.
- Keep summaries bounded, stable, and deterministic.
- Preserve stable reason codes and exit codes in failure paths.

## 2. Runtime Controls (Required)

- Define budgets (`max_wall_time_seconds`, retries, tool calls, step count).
- Require environment fingerprint rules for resumability safety.
- Ensure blocked jobs can be resumed only through explicit approve/resume flow.

## 3. Evidence and Acceptance (Required)

- Export `jobpack_<job_id>.zip` and verify with `wrkr verify`.
- Run `wrkr accept run <job_id>` before final approval.
- Produce GitHub summary output via `wrkr report github <job_id>` for CI visibility.

## 4. Sidecar Rules (If Used)

- Sidecar is transport-only: no policy or planning logic.
- Sidecar outputs are deterministic and lane-scoped:
  - `./wrkr-out/integrations/<lane>/request.json`
  - `./wrkr-out/integrations/<lane>/result.json`
  - `./wrkr-out/integrations/<lane>/sidecar.log`
- Dry-run path must work offline with fixture input.

## 5. Required Verification Commands

- `make test-adoption`
- `make test-conformance`
- `make test-uat-local`

If all three pass, the lane is ready to claim Wrkr-compatible behavior.
