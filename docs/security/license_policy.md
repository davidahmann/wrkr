# OSS License Policy (Lightweight v1)

Wrkr v1 enforces a lightweight dependency hygiene policy for OSS operation.

## Policy

- Dependencies must resolve from public module registries.
- Local/internal module paths are not allowed in release-bound builds.
- A deterministic dependency inventory is generated in CI.

## Command

```bash
make license-check
```

Outputs:

- `wrkr-out/reports/license_inventory_go.txt`

## Escalation

If a dependency has uncertain licensing terms, open an issue and block release until reviewed.
