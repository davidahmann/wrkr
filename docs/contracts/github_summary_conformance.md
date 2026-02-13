# GitHub Summary Conformance (v1)

`wrkr report github` and `wrkr accept run --ci` produce deterministic GitHub summary artifacts.

## Contract

- JSON summary is written to `./wrkr-out/reports/github_summary_<job_id>.json`.
- Markdown summary is written to `./wrkr-out/reports/github_summary_<job_id>.md`.
- If `GITHUB_STEP_SUMMARY` is set, the markdown is appended to that path.
- JSON summary conforms to `schemas/v1/report/github_summary.schema.json`.
- Markdown contains these sections in stable order:
  - `Final Checkpoint`
  - `Top Failures`
  - `Artifact Pointers`

## Determinism

For identical `jobpack` input:

- JSON content is canonicalized and byte-stable.
- Markdown content is byte-stable.
- Artifact pointers are sorted lexicographically and capped to 5 items.
- Top failures are emitted in deterministic acceptance-result order and capped to 5 items.

Conformance automation:
- `scripts/test_github_summary_golden.sh`
- `.github/workflows/wrkr-compatible-conformance.yml`
