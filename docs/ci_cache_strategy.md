# CI Cache Strategy

Wrkr uses stable, dependency-file keyed caches through setup actions.

## Go

- `actions/setup-go` manages module/build caches keyed by Go version + `go.sum`.
- Bust cache by changing `go.sum` or Go toolchain version.

## Python

- `actions/setup-python` cache is keyed by Python version and lock/dependency files where configured.
- Wrkr jobs install `uv` and resolve from `sdk/python/uv.lock`.

## Docs Site (Node)

- `actions/setup-node` uses npm cache with `docs-site/package-lock.json`.
- Bust cache by changing the lock file or Node version.

## Operational Guidance

- Avoid manual cache invalidation unless a cache corruption issue is confirmed.
- Prefer lockfile updates over ad-hoc cache key changes.
