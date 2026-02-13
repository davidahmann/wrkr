# UAT Functional Plan (v1)

This plan defines local UAT for Wrkr OSS before release or rollout.

## Scope

- Core command flow: demo, submit, status, checkpoint, approve, resume, export, verify, accept, report.
- Integration surfaces: bridge work-item, wrap mode, serve hardening checks.
- Deterministic output layout under `./wrkr-out/`.

## Local UAT Command Matrix

| Area | Command | Expected |
| --- | --- | --- |
| Adoption smoke | `./scripts/test_adoption_smoke.sh` | Pass with stage diagnostics |
| Adapter parity | `./scripts/test_adapter_parity.sh` | Pass and deterministic sidecar dry-run output |
| UAT orchestrator | `./scripts/test_uat_local.sh` | Pass + emits summary files |

## Outputs

- `./wrkr-out/reports/uat_summary.json`
- `./wrkr-out/reports/uat_summary.md`
- `./wrkr-out/reports/test_adoption_smoke.log`
- `./wrkr-out/reports/test_adapter_parity.log`

## CI/Nightly Hooks

- `make test-uat-local` is wired into `perf-nightly.yml`.
- `adoption-nightly.yml` runs adoption and conformance regressions.

## Pass Criteria

- `status=pass` in `uat_summary.json`
- All matrix steps pass with no contract regressions.
