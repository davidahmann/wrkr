# Wrkr

Wrkr is a durable dispatch and supervision substrate for long-running agent jobs.

## Current status

This repository includes implemented foundations through Epic 11:
- durable local runner with checkpoints, approvals, budgets, export/verify, and acceptance harness
- release engineering pipeline (multi-platform artifacts, checksums, SBOM, vulnerability report, provenance, signatures)
- adoption kit and blessed integration lane (`examples/integrations`, sidecar + wrap + bridge flow)
- UAT automation and adoption parity suites (`make test-adoption`, `make test-uat-local`)
- deterministic GitHub summary/report artifacts and Wrkr-compatible conformance workflows
- local developer workflows (`make`, hooks, pre-commit), CI fast lane, mainline, and nightly deep lanes

## Quickstart

```bash
make fmt
make lint-fast
make test-fast
make sast-fast
make coverage
wrkr help
wrkr --json --explain submit
```

## Product docs

- PRD: `product/PRD.md`
- Plan: `product/PLAN_v1.md`
- Docs map: `docs/README.md`
