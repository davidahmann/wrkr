import Link from 'next/link'
import type { Metadata } from 'next'
import { canonicalUrl } from '../../src/lib/site'

export const metadata: Metadata = {
  title: 'Wrkr Documentation',
  description: 'Start-to-production documentation ladder for Wrkr OSS.',
  alternates: { canonical: canonicalUrl('/docs/') },
}

const tracks = [
  {
    title: 'Track 1: First Win (5 Minutes)',
    steps: [
      { label: 'Install', href: '/docs/install' },
      { label: 'Mental Model', href: '/docs/concepts/mental_model' },
      { label: 'Flows', href: '/docs/flows' },
    ],
  },
  {
    title: 'Track 2: Integration (30-120 Minutes)',
    steps: [
      { label: 'Integration Checklist', href: '/docs/integration_checklist' },
      { label: 'Blessed Lane', href: '/docs/ecosystem/blessed_lane' },
      { label: 'GitHub Actions Kit', href: '/docs/ecosystem/github_actions_kit' },
      { label: 'Project Defaults', href: '/docs/project_defaults' },
      { label: 'CI Regress Kit', href: '/docs/ci_regress_kit' },
    ],
  },
  {
    title: 'Track 3: Production Posture',
    steps: [
      { label: 'Production Readiness', href: '/docs/hardening/production_readiness' },
      { label: 'Primitive Contract', href: '/docs/contracts/primitive_contract' },
      { label: 'Runtime SLO', href: '/docs/slo/runtime_slo' },
      { label: 'Release Checklist', href: '/docs/hardening/release_checklist' },
    ],
  },
  {
    title: 'Track 4: Compliance and Evidence',
    steps: [
      { label: 'Checkpoint Protocol', href: '/docs/contracts/checkpoint_protocol' },
      { label: 'Acceptance Contract', href: '/docs/contracts/acceptance_contract' },
      { label: 'Jobpack Verify', href: '/docs/contracts/jobpack_verify' },
      { label: 'Failure Taxonomy', href: '/docs/contracts/failure_taxonomy' },
    ],
  },
]

export default function DocsHomePage() {
  return (
    <div className="docs-home not-prose">
      <h1>Documentation</h1>
      <p>
        Use this ladder to go from first runnable demo to deterministic long-running dispatch operations and
        production supervision.
      </p>

      <div className="track-grid">
        {tracks.map((track) => (
          <section key={track.title} className="track-card">
            <h2>{track.title}</h2>
            <div className="track-steps">
              {track.steps.map((step) => (
                <Link key={step.href} href={step.href} className="track-step">
                  {step.label}
                </Link>
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  )
}
