# Acceptance Harness Contract

## Commands

- `wrkr accept init`
- `wrkr accept run <job_id>`

## Deterministic Checks

- Required artifact presence.
- Test command execution.
- Lint command execution.
- Path policy checks.

## Exit Codes

- `0`: acceptance passed
- `5`: acceptance failed (`E_ACCEPT_*`)
- `6`: invalid acceptance input/schema
