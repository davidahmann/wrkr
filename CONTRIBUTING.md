# Contributing

## Development flow

1. Create a branch from `master`.
2. Run local checks:

```bash
make fmt
make lint-fast
make test-fast
make sast-fast
```

3. Run full checks as needed:

```bash
make lint
make test
make test-conformance
```

## Hooks

Install hooks once per clone:

```bash
make hooks
```

## Commit style

- Keep commits scoped to one logical change.
- Include test updates with behavior changes.
