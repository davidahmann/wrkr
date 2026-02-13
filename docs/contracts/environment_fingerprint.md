# Environment Fingerprint Contract

Wrkr captures environment fingerprints on job initialization and validates on resume.

## Capture

- Fingerprint hash and selected rule/value map are recorded in state.
- Default rules are applied when not explicitly set.

## Resume Gating

- If fingerprint mismatches and no override flag is provided, resume is blocked with `E_ENV_FINGERPRINT_MISMATCH`.
- Override path records explicit override event metadata.
