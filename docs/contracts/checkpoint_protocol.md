# Checkpoint Protocol Contract

## Allowed Types

- `plan`
- `progress`
- `decision-needed`
- `blocked`
- `completed`

## Required Fields

- `checkpoint_id`
- `job_id`
- `type`
- `summary`
- `status`
- `budget_state`
- `artifacts_delta`
- `reason_codes`

## Summary Rules

- Summaries must be bounded and review-oriented.
- Decision checkpoints must include actionable required action context.
