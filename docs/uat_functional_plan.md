# Local UAT + Functional Plan

This runbook defines how to validate Wrkr end-to-end locally across supported install paths.

## Goal

Prove that an operator can:

- install Wrkr via each v1 install path
- run core dispatch/checkpoint/accept/jobpack flows
- verify deterministic output contracts
- produce UAT summary artifacts for release readiness

## Install Paths In Scope

1. Source build (`go build -o ./wrkr ./cmd/wrkr`)
2. GitHub release installer (`scripts/install.sh`)
3. Homebrew tap (`davidahmann/tap/wrkr`)

Windows stays validated in release artifact matrix; this local UAT script focuses on Linux/macOS hosts.
Release-installer path requires at least one published GitHub release tag.

## UAT Orchestrator

Entry point:

```bash
bash scripts/test_uat_local.sh
```

Options:

```bash
bash scripts/test_uat_local.sh --release-version vX.Y.Z
bash scripts/test_uat_local.sh --skip-release-installer
bash scripts/test_uat_local.sh --skip-brew
```

Nightly CI runners currently use skip flags for brew and release-installer paths; full three-path coverage is the default local release-readiness run.

## Required Scripts

- `scripts/test_uat_local.sh` (orchestrator)
- `scripts/test_adoption_smoke.sh` (core command flow)
- `scripts/test_adapter_parity.sh` (adapter/sidecar parity)
- `scripts/test_install.sh` (installer smoke)
- `scripts/install.sh` (release installer)

## Outputs

- `wrkr-out/uat_local/summary.md`
- `wrkr-out/uat_local/summary.json`
- `wrkr-out/uat_local/logs/*.log`

## Pass Criteria

- Source install path passes adoption smoke.
- Release installer path passes install + adoption smoke (unless explicitly skipped).
- Homebrew path passes tap/reinstall/test + adoption smoke (unless explicitly skipped).
- Adapter parity suite passes.
- Final summary status is `pass`.

## Failure Handling

1. Open failing log in `wrkr-out/uat_local/logs/`.
2. Fix root cause in code/scripts/docs (no test weakening).
3. Re-run `bash scripts/test_uat_local.sh`.
4. Merge only when all required paths are green.
