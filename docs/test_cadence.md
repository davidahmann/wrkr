# Test Cadence Policy

Wrkr uses three lanes to keep feedback fast while preserving release safety.

## Fast Lane (every PR)

Run:

- `make lint-fast`
- `make test-fast`
- `make sast-fast`

Purpose: quick correctness and security signal.

## Mainline Lane (merge to master)

Run in CI:

- `make test-e2e`
- `make test-contracts`
- `make test-acceptance`
- `make test-conformance`
- docs-site lint/build

Purpose: enforce contracts and integration compatibility.

## Nightly Deep Lane

Run via nightly workflows:

- `make test-adoption`
- `make test-uat-local`
- `make test-runtime-slo`
- `make test-hardening-acceptance`

Purpose: detect drift/regressions in adoption readiness, runtime budgets, and hardening guardrails.

## Release-Blocking Gates

Before release, all must be green:

- `make test-v1-acceptance`
- `make test-adoption`
- `make test-uat-local`
- `make test-hardening-acceptance`
