# AGENTS

Wrkr repository guidance for coding agents.

## Scope

- Preserve deterministic behavior and stable contracts.
- Prefer additive changes to schemas and CLI interfaces.
- Keep OSS defaults safe and offline-first.

## Local validation

Run before pushing:

```bash
make fmt
make lint-fast
make test-fast
make sast-fast
```
