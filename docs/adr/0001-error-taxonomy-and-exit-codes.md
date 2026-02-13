# ADR 0001: Error Taxonomy and Exit Codes

Date: 2026-02-13
Status: Accepted

## Context

Wrkr requires stable machine-readable failure behavior across CLI, CI, and API layers.

## Decision

Wrkr uses a stable error envelope with this shape:

- `schema_id`: `wrkr.error_envelope`
- `schema_version`: `v1`
- `created_at`
- `producer_version`
- `code`
- `message`
- `exit_code`
- `details` (optional)

Exit-code contract:

- `0`: success
- `1`: generic failure
- `2`: verification failed
- `4`: approval required
- `5`: acceptance failed
- `6`: invalid input/schema
- `8`: unsafe operation attempted without explicit flag

Primary reason-code taxonomy for v1:

- `E_BUDGET_EXCEEDED`
- `E_ADAPTER_FAIL`
- `E_CHECKPOINT_APPROVAL_REQUIRED`
- `E_ACCEPT_MISSING_ARTIFACT`
- `E_ACCEPT_TEST_FAIL`
- `E_VERIFY_HASH_MISMATCH`
- `E_STORE_CORRUPT`
- `E_ENV_FINGERPRINT_MISMATCH`
- `E_LEASE_CONFLICT`
- `E_INVALID_INPUT_SCHEMA`
- `E_UNSAFE_OPERATION`

## Consequences

- Consumers can rely on deterministic exit handling in scripts and CI.
- Any future changes must be additive within `v1.x`.
