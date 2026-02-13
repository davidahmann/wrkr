# Runtime SLO

## Service Objectives (OSS Baseline)

- Recovery correctness after restart: deterministic and lossless.
- Jobpack verify correctness: deterministic and tamper-detecting.
- Checkpoint protocol validity: schema-stable across v1.

## Operational Targets

- Fast lane CI remains reproducible and green on release branch.
- Deterministic acceptance checks run locally and in CI.

## Budget Contracts

- Runtime command budgets: `perf/runtime_slo_budgets.json`
- Resource budgets: `perf/resource_budgets.json`

## Local Validation

- `make test-runtime-slo`

This runs:

- `scripts/check_command_budgets.py`
- `scripts/check_resource_budgets.py`

Both scripts emit deterministic reports under `./wrkr-out/reports/`.
