# Production Readiness Doctor

Use `wrkr doctor --production-readiness` before non-trivial production rollout.

You can evaluate serve posture from actual runtime inputs (recommended) with:

- `--serve-listen`
- `--serve-allow-non-loopback`
- `--serve-auth-token`
- `--serve-max-body-bytes`

If those flags are omitted, doctor falls back to `WRKR_SERVE_*` environment variables.

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
  - non-loopback serve listeners (including wildcard binds such as `:9488`) require:
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

# evaluate actual serve launch parameters
wrkr doctor --production-readiness \
  --serve-listen :9488 \
  --serve-allow-non-loopback \
  --serve-auth-token "$WRKR_TOKEN" \
  --serve-max-body-bytes 1048576 \
  --json
```

Exit behavior:

- `0`: all critical checks passed
- `1`: at least one critical check failed
