# Jobpack and Verify Contract

## Required Jobpack Files

- `manifest.json`
- `job.json`
- `events.jsonl`
- `checkpoints.jsonl`
- `artifacts_manifest.json`
- Optional: `accept/accept_result.json`, `approvals.jsonl`

## Verification Rules

- Manifest schema must validate.
- `manifest_sha256` must match canonical manifest hash.
- Every declared file hash must match.
- Undeclared archive entries fail verification.
- Schema validation for known artifact files is enforced.
