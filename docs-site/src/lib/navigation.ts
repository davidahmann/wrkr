export type NavSection = {
  title: string
  items: Array<{ label: string; href: string }>
}

export const navigation: NavSection[] = [
  {
    title: 'Getting Started',
    items: [
      { label: 'Docs Map', href: '/docs/readme' },
      { label: 'Root README', href: '/docs/repo/readme' },
      { label: 'Contributing', href: '/docs/repo/contributing' },
      { label: 'Security', href: '/docs/repo/security' },
    ],
  },
  {
    title: 'Concepts',
    items: [
      { label: 'Mental Model', href: '/docs/concepts/mental_model' },
      { label: 'Architecture', href: '/docs/architecture' },
      { label: 'Flows', href: '/docs/flows' },
    ],
  },
  {
    title: 'Contracts',
    items: [
      { label: 'Primitive Contract', href: '/docs/contracts/primitive_contract' },
      { label: 'Checkpoint Protocol', href: '/docs/contracts/checkpoint_protocol' },
      { label: 'Jobpack Verify', href: '/docs/contracts/jobpack_verify' },
      { label: 'Serve API', href: '/docs/contracts/serve_api' },
      { label: 'Wrkr-Compatible', href: '/docs/contracts/wrkr_compatible' },
    ],
  },
  {
    title: 'Runbooks',
    items: [
      { label: 'Integration Checklist', href: '/docs/integration_checklist' },
      { label: 'Blessed Lane', href: '/docs/ecosystem/blessed_lane' },
      { label: 'GitHub Actions Kit', href: '/docs/ecosystem/github_actions_kit' },
      { label: 'Serve Mode', href: '/docs/deployment/serve_mode' },
      { label: 'Release Checklist', href: '/docs/hardening/release_checklist' },
    ],
  },
]
