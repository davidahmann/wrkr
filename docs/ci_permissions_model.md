# CI Permissions Model

Wrkr workflows default to least privilege.

## Default Baseline

- `permissions: contents: read`

## Elevated Permissions by Exception

- `docs.yml` deploy job:
  - `pages: write`
  - `id-token: write`
- `codeql.yml`:
  - `security-events: write`
  - `actions: read`

All other workflows remain read-only unless a specific write scope is required.
