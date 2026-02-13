export interface NavItem {
  title: string
  href: string
  children?: NavItem[]
}

export const navigation: NavItem[] = [
  {
    title: 'Start Here',
    href: '/docs',
    children: [
      { title: 'Install', href: '/docs/install' },
      { title: 'Mental Model', href: '/docs/concepts/mental_model' },
      { title: 'Architecture', href: '/docs/architecture' },
      { title: 'Flows', href: '/docs/flows' },
      { title: 'Docs Map', href: '/docs/readme' },
    ],
  },
  {
    title: 'Operate',
    href: '/docs/integration_checklist',
    children: [
      { title: 'Integration Checklist', href: '/docs/integration_checklist' },
      { title: 'Blessed Lane', href: '/docs/ecosystem/blessed_lane' },
      { title: 'GitHub Actions Kit', href: '/docs/ecosystem/github_actions_kit' },
      { title: 'Project Defaults', href: '/docs/project_defaults' },
      { title: 'CI Regress Kit', href: '/docs/ci_regress_kit' },
      { title: 'UAT Plan', href: '/docs/uat_functional_plan' },
    ],
  },
  {
    title: 'Production',
    href: '/docs/hardening/production_readiness',
    children: [
      { title: 'Production Readiness', href: '/docs/hardening/production_readiness' },
      { title: 'Runtime SLO', href: '/docs/slo/runtime_slo' },
      { title: 'Retention Profiles', href: '/docs/slo/retention_profiles' },
      { title: 'Serve Deployment', href: '/docs/deployment/serve_mode' },
      { title: 'Release Checklist', href: '/docs/hardening/release_checklist' },
    ],
  },
  {
    title: 'Contracts',
    href: '/docs/contracts/primitive_contract',
    children: [
      { title: 'Primitive Contract', href: '/docs/contracts/primitive_contract' },
      { title: 'Checkpoint Protocol', href: '/docs/contracts/checkpoint_protocol' },
      { title: 'Acceptance Contract', href: '/docs/contracts/acceptance_contract' },
      { title: 'Jobpack Verify', href: '/docs/contracts/jobpack_verify' },
      { title: 'Output Layout', href: '/docs/contracts/output_layout' },
      { title: 'Failure Taxonomy', href: '/docs/contracts/failure_taxonomy' },
      { title: 'Serve API', href: '/docs/contracts/serve_api' },
      { title: 'Wrkr-Compatible', href: '/docs/contracts/wrkr_compatible' },
    ],
  },
  {
    title: 'Ecosystem',
    href: '/docs/ecosystem/integration_rfc_template',
    children: [
      { title: 'Integration RFC Template', href: '/docs/ecosystem/integration_rfc_template' },
      { title: 'Homebrew', href: '/docs/homebrew' },
      { title: 'CodeQL Triage', href: '/docs/security/codeql_triage' },
      { title: 'License Policy', href: '/docs/security/license_policy' },
      { title: 'Contributing Guide', href: '/docs/contributing' },
      { title: 'Security Policy', href: '/docs/security' },
    ],
  },
]
