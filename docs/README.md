# Wrkr Documentation Map

This file is the authoritative docs index for Wrkr OSS v1.

## Start Here

1. `README.md` for product overview and first win
2. `docs/install.md` for installation paths
3. `docs/concepts/mental_model.md` for runtime primitives
4. `docs/architecture.md` for component boundaries
5. `docs/flows.md` for operational sequences
6. `docs/contracts/primitive_contract.md` for normative behavior

## Core Product Docs

- Architecture: `docs/architecture.md`
- Runtime flows: `docs/flows.md`
- Install paths: `docs/install.md`
- Homebrew path: `docs/homebrew.md`
- Project defaults: `docs/project_defaults.md`
- Integration checklist: `docs/integration_checklist.md`

## Contracts And Compatibility (Normative)

- Primitive contract: `docs/contracts/primitive_contract.md`
- Output layout: `docs/contracts/output_layout.md`
- Checkpoint protocol: `docs/contracts/checkpoint_protocol.md`
- Lease/heartbeat: `docs/contracts/lease_heartbeat.md`
- Environment fingerprint: `docs/contracts/environment_fingerprint.md`
- Jobpack verify: `docs/contracts/jobpack_verify.md`
- Acceptance harness: `docs/contracts/acceptance_contract.md`
- Failure taxonomy: `docs/contracts/failure_taxonomy.md`
- Ticket footer conformance: `docs/contracts/ticket_footer_conformance.md`
- GitHub summary conformance: `docs/contracts/github_summary_conformance.md`
- Work-item bridge contract: `docs/contracts/work_item_bridge_contract.md`
- Serve API contract: `docs/contracts/serve_api.md`
- Wrkr-compatible claim: `docs/contracts/wrkr_compatible.md`

## Operations And Hardening

- CI regress kit: `docs/ci_regress_kit.md`
- CI required checks: `docs/ci_required_checks.md`
- CI permissions model: `docs/ci_permissions_model.md`
- CI cache strategy: `docs/ci_cache_strategy.md`
- Runtime SLOs: `docs/slo/runtime_slo.md`
- Retention profiles: `docs/slo/retention_profiles.md`
- Production readiness doctor: `docs/hardening/production_readiness.md`
- Release checklist: `docs/hardening/release_checklist.md`
- CodeQL triage: `docs/security/codeql_triage.md`
- License policy: `docs/security/license_policy.md`
- UAT plan: `docs/uat_functional_plan.md`
- Test cadence: `docs/test_cadence.md`

## Ecosystem And Launch

- Blessed lane runbook: `docs/ecosystem/blessed_lane.md`
- GitHub Actions kit: `docs/ecosystem/github_actions_kit.md`
- Integration RFC template: `docs/ecosystem/integration_rfc_template.md`
- Serve deployment: `docs/deployment/serve_mode.md`
- Release template: `docs/launch/github_release_template.md`

## Ownership Rules

- `docs/contracts/*` are normative. If another doc conflicts, contracts win.
- `README.md` is onboarding and positioning, not the full runbook set.
- Operational procedures belong in runbooks (`docs/*_kit.md`, `docs/hardening/*`, `docs/security/*`).
- `docs/wiki/*` is convenience material; `docs/*` remains authoritative.

## Tooling References

- Docs site source: `docs-site/`
- Docs workflow: `.github/workflows/docs.yml`
- Hero demo recorder: `scripts/record_wrkr_hero_demo.sh`
