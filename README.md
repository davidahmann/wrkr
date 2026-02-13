# Wrkr

Wrkr is a durable dispatch and supervision substrate for long-running agent jobs.

## Current status

This repository includes implemented foundations through Epic 9:
- durable local runner, checkpoints, approvals, budgets, export/verify
- acceptance harness (`wrkr accept`) and GitHub summary report (`wrkr report github`)
- CLI command map for demo/init/submit/pause/resume/cancel/wrap/bridge/serve/doctor/store
- adapter layer (wrap + reference), sidecar example, and Python SDK wrappers
- hardening and SLO checks (`make test-hardening-acceptance`, `make test-runtime-slo`)
- local developer workflows (`make`, hooks, pre-commit) and CI fast lane

## Quickstart

```bash
make fmt
make lint-fast
make test-fast
make sast-fast
wrkr help
```

## Product docs

- PRD: `product/PRD.md`
- Plan: `product/PLAN_v1.md`
- Docs map: `docs/README.md`
