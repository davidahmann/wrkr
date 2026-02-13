# Failure Taxonomy Contract

This document is normative for Wrkr v1 reason codes and exit code mapping.

## Reason Codes (Canonical v1)

- `E_BUDGET_EXCEEDED`
- `E_ADAPTER_FAIL`
- `E_CHECKPOINT_APPROVAL_REQUIRED`
- `E_ACCEPT_MISSING_ARTIFACT`
- `E_ACCEPT_TEST_FAIL`
- `E_VERIFY_HASH_MISMATCH`
- `E_STORE_CORRUPT`
- `E_ENV_FINGERPRINT_MISMATCH`
- `E_LEASE_CONFLICT`
- `E_INVALID_STATE_TRANSITION`
- `E_INVALID_INPUT_SCHEMA`
- `E_UNSAFE_OPERATION`

## Exit Codes

- `0` success
- `1` generic failure
- `2` verification failed (`E_VERIFY_HASH_MISMATCH`)
- `4` approval required (`E_CHECKPOINT_APPROVAL_REQUIRED`)
- `5` acceptance failed (`E_ACCEPT_MISSING_ARTIFACT`, `E_ACCEPT_TEST_FAIL`)
- `6` invalid input/schema (`E_INVALID_INPUT_SCHEMA`)
- `8` unsafe operation attempted without explicit flag (`E_UNSAFE_OPERATION`)

## Compatibility Policy

- The v1 set is additive-only in `v1.x`.
- Existing codes must keep semantics and exit-code mapping.
- Removing or redefining a code requires a major version.
