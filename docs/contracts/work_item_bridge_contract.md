# Work-Item Bridge Contract

`wrkr bridge work-item` converts interrupt checkpoints into deterministic payloads.

## Input

- `blocked` or `decision-needed` checkpoint.

## Output Fields

- `required_action`
- `reason_codes`
- `artifact_pointers`
- `next_commands`

## Behavior

- `--dry-run` prints deterministic payload and never mutates job state.
- Optional templates (GitHub/Jira) are presentation-only and credential-free.
