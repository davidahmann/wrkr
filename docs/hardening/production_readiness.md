# Production Readiness Doctor

Use `wrkr doctor --production-readiness` before non-trivial production rollout.

The production profile fails closed on critical checks:

- Strict profile: `WRKR_PROFILE=strict`
- Signing mode: `WRKR_SIGNING_MODE=ed25519`
- Signing key source:
  - `WRKR_SIGNING_KEY_SOURCE=file|env|kms`
  - if `file`, `WRKR_SIGNING_KEY_PATH` must exist
- Store lock health: append-lock acquire/release under `~/.wrkr/jobs`
- Retention settings:
  - `WRKR_RETENTION_DAYS` (positive integer)
  - `WRKR_OUTPUT_RETENTION_DAYS` (positive integer)
- Unsafe defaults disabled:
  - `WRKR_ALLOW_UNSAFE` must not be enabled
  - non-loopback serve listeners require:
    - `WRKR_SERVE_AUTH_TOKEN`
    - `WRKR_SERVE_MAX_BODY_BYTES`

Example:

```bash
WRKR_PROFILE=strict \
WRKR_SIGNING_MODE=ed25519 \
WRKR_SIGNING_KEY_SOURCE=env \
WRKR_RETENTION_DAYS=14 \
WRKR_OUTPUT_RETENTION_DAYS=14 \
wrkr doctor --production-readiness --json
```

Exit behavior:

- `0`: all critical checks passed
- `1`: at least one critical check failed
