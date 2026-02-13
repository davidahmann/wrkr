# Integration Checklist (v1)

Use this checklist when integrating Wrkr via wrap, structured adapter, or sidecar.

## Required

- Define the lane name used in `./wrkr-out/integrations/<lane>/...`.
- Ensure checkpoint types use Wrkr canonical values:
  - `plan`
  - `progress`
  - `decision-needed`
  - `blocked`
  - `completed`
- Ensure every `decision-needed` checkpoint has explicit required action.
- Keep summaries bounded and deterministic.
- Export `jobpack_<job_id>.zip` and verify with `wrkr verify`.
- Run `wrkr accept run <job_id>` before final approval.

## Sidecar Rules

- Sidecar is transport-only: no hidden planning or policy logic.
- Sidecar writes deterministic outputs under:
  - `./wrkr-out/integrations/<lane>/request.json`
  - `./wrkr-out/integrations/<lane>/result.json`
  - `./wrkr-out/integrations/<lane>/sidecar.log`
- Sidecar must support offline fixture runs.

## Minimal Acceptance Gate

- Required artifacts present.
- Tests executed.
- Lint executed.
- Jobpack verify passes.
