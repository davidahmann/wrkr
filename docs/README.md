# Wrkr Docs Map

This is the authoritative documentation map for Wrkr.

## Ownership Rules

- Contracts in `docs/contracts/` are normative.
- `README.md` is onboarding, not the full spec.
- Runbooks describe operational procedures.
- `docs/wiki/` is convenience material, never the source of truth.

## Start Here

- Product PRD: `product/PRD.md`
- Delivery plan: `product/PLAN_v1.md`
- Core concepts: `docs/concepts/mental_model.md`
- Architecture: `docs/architecture.md`
- Runtime flows: `docs/flows.md`

## Normative Contracts

- Primitive contract: `docs/contracts/primitive_contract.md`
- Output layout: `docs/contracts/output_layout.md`
- Checkpoint protocol: `docs/contracts/checkpoint_protocol.md`
- Lease/heartbeat: `docs/contracts/lease_heartbeat.md`
- Environment fingerprint: `docs/contracts/environment_fingerprint.md`
- Jobpack + verify: `docs/contracts/jobpack_verify.md`
- Acceptance harness contract: `docs/contracts/acceptance_contract.md`
- Failure taxonomy contract: `docs/contracts/failure_taxonomy.md`
- Ticket footer conformance: `docs/contracts/ticket_footer_conformance.md`
- GitHub summary conformance: `docs/contracts/github_summary_conformance.md`
- Work-item bridge contract: `docs/contracts/work_item_bridge_contract.md`
- Serve API contract: `docs/contracts/serve_api.md`
- Wrkr-compatible claim: `docs/contracts/wrkr_compatible.md`

## Operational Runbooks

- Integration checklist: `docs/integration_checklist.md`
- Blessed lane kit: `docs/ecosystem/blessed_lane.md`
- Integration RFC template: `docs/ecosystem/integration_rfc_template.md`
- GitHub Actions kit: `docs/ecosystem/github_actions_kit.md`
- Serve mode deployment: `docs/deployment/serve_mode.md`
- CI regress kit: `docs/ci_regress_kit.md`
- CI required checks: `docs/ci_required_checks.md`
- CI permissions model: `docs/ci_permissions_model.md`
- CI cache strategy: `docs/ci_cache_strategy.md`
- Project defaults: `docs/project_defaults.md`
- Runtime SLO: `docs/slo/runtime_slo.md`
- Retention profiles: `docs/slo/retention_profiles.md`
- Coverage thresholds: `perf/coverage_thresholds.json`
- Production readiness doctor: `docs/hardening/production_readiness.md`
- CodeQL triage: `docs/security/codeql_triage.md`
- OSS license policy: `docs/security/license_policy.md`
- Release hardening checklist: `docs/hardening/release_checklist.md`
- UAT functional plan: `docs/uat_functional_plan.md`
- Test cadence policy: `docs/test_cadence.md`
- GitHub release template: `docs/launch/github_release_template.md`
