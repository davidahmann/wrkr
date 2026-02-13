# Retention Profiles

## Local Dev

- Keep recent job state and jobpacks for active debugging.
- Prune stale outputs periodically.

## CI

- Retain jobpack and reports for build auditing windows.
- Keep acceptance and summary artifacts with CI retention policy.

## Security

- Prefer reference-only artifact capture unless explicitly required.

## CLI Retention Controls

Wrkr retention is deterministic and scriptable via `wrkr store prune`.

Examples:

```bash
# Preview only
wrkr store prune \
  --dry-run \
  --job-max-age 168h \
  --jobpack-max-age 168h \
  --report-max-age 168h \
  --integration-max-age 168h \
  --max-jobpacks 200 \
  --max-reports 500 \
  --json

# Apply deletions
wrkr store prune \
  --job-max-age 168h \
  --jobpack-max-age 168h \
  --report-max-age 168h \
  --integration-max-age 168h \
  --max-jobpacks 200 \
  --max-reports 500
```

The prune report includes:

- checked objects count
- matched objects count
- removed objects count
- freed bytes
- per-entry reason (`age` or `count`)
