# Serve Mode Deployment and Hardening

## Default Safe Mode

- Bind to loopback only (`127.0.0.1:9488` default).
- No auth required for loopback-only local use.

## Non-Loopback Hardening

Must provide all:

- `--allow-non-loopback`
- `--auth-token <token>`
- `--max-body-bytes <n>`

## Operational Checks

- Verify endpoint access requires auth in non-loopback mode.
- Verify path traversal attempts are rejected.
- Verify oversized body requests are rejected.
