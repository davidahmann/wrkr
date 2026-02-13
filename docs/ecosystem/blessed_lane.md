# Blessed Lane Integration Kit

This is the worked example lane for Wrkr-compatible integration.

## Lane Shape

- Adapter type: structured (`reference`) or wrap.
- Deterministic outputs under `./wrkr-out/integrations/blessed/`.
- Explicit decision checkpoint for human approval.

## Required Commands

1. `wrkr submit jobspec.yaml`
2. `wrkr checkpoint list <job_id>`
3. `wrkr approve <job_id> --checkpoint <id> --reason ...`
4. `wrkr resume <job_id>`
5. `wrkr accept run <job_id>`
6. `wrkr export <job_id>` and `wrkr verify <job_id>`

## Evidence Pack

- Jobpack zip
- Acceptance result
- GitHub summary report (optional CI)
