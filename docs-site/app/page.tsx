import Link from 'next/link'
import type { Metadata } from 'next'
import { canonicalUrl } from '../src/lib/site'

export const metadata: Metadata = {
  title: 'Wrkr | Dispatch and Supervision for Long-Running Agent Jobs',
  description:
    'Wrkr is the offline-first execution substrate for long-running agent jobs: dispatch, checkpoint, accept, and jobpack.',
  alternates: {
    canonical: canonicalUrl('/'),
  },
}

const QUICKSTART = `git clone https://github.com/davidahmann/wrkr.git\ncd wrkr\nmake build\n./bin/wrkr demo --json\n./bin/wrkr verify <job_id> --json`

const features = [
  {
    title: 'Dispatch: Durable Job Lifecycle',
    description: 'Submit, pause, resume, and cancel jobs with deterministic state transitions and restart-safe persistence.',
    href: '/docs/contracts/primitive_contract',
  },
  {
    title: 'Checkpoint: Structured Supervision',
    description: 'Use typed checkpoints to review meaningful deltas and unblock jobs with explicit approvals.',
    href: '/docs/contracts/checkpoint_protocol',
  },
  {
    title: 'Accept: Deterministic Readiness Signal',
    description: 'Run deterministic acceptance checks locally and in CI with stable machine-readable output.',
    href: '/docs/contracts/acceptance_contract',
  },
  {
    title: 'Jobpack: Portable Evidence Bundle',
    description: 'Export verifiable jobpacks with manifests, checkpoints, and artifacts for PRs and incidents.',
    href: '/docs/contracts/jobpack_verify',
  },
  {
    title: 'Wrap: Zero-Integration Adoption',
    description: 'Wrap existing agent commands and still land on the same checkpoint and jobpack contract.',
    href: '/docs/ecosystem/blessed_lane',
  },
  {
    title: 'Doctor: Production Readiness',
    description: 'Detect risky runtime posture early with explicit diagnostics and hardening recommendations.',
    href: '/docs/hardening/production_readiness',
  },
]

const faqs = [
  {
    question: 'What problem does Wrkr solve?',
    answer:
      'Wrkr makes multi-hour agent work durable and reviewable by default with resumable state, budget controls, and acceptance signals.',
  },
  {
    question: 'Does Wrkr require a hosted control plane?',
    answer:
      'No. Wrkr OSS v1 is CLI-first and offline-first for core workflows including demo, export, verify, and most supervision operations.',
  },
  {
    question: 'What is the shareable artifact?',
    answer:
      'The shareable unit is a jobpack zip plus job_id, which can be attached to PRs, tickets, and incident records for deterministic review.',
  },
  {
    question: 'How quickly can I integrate Wrkr?',
    answer:
      'Most teams start with demo in under a minute, then wire a real jobspec path and acceptance checks in one CI lane within hours.',
  },
]

const softwareApplicationJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'Wrkr',
  applicationCategory: 'DeveloperApplication',
  operatingSystem: 'Linux, macOS, Windows',
  description:
    'Offline-first CLI for durable dispatch and supervision of long-running agent jobs with checkpointing, acceptance harnesses, and jobpack verification.',
  url: 'https://davidahmann.github.io/wrkr/',
  softwareHelp: 'https://davidahmann.github.io/wrkr/docs/',
  codeRepository: 'https://github.com/davidahmann/wrkr',
  offers: {
    '@type': 'Offer',
    price: '0',
    priceCurrency: 'USD',
  },
}

const faqJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'FAQPage',
  mainEntity: faqs.map((entry) => ({
    '@type': 'Question',
    name: entry.question,
    acceptedAnswer: {
      '@type': 'Answer',
      text: entry.answer,
    },
  })),
}

export default function HomePage() {
  return (
    <div className="home not-prose">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareApplicationJsonLd) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }}
      />

      <section className="hero">
        <h1>
          Dispatch and Supervise
          <span> Long-Running Agent Jobs</span>
        </h1>
        <p>
          Wrkr turns agent runs into durable work units: checkpointed execution, budget and approval gates,
          deterministic acceptance, and verifiable jobpack evidence.
        </p>
        <div className="hero-cta">
          <Link href="/docs/install" className="btn btn-primary">
            Start Here
          </Link>
          <Link href="/docs/integration_checklist" className="btn btn-secondary">
            Integration Checklist
          </Link>
        </div>
      </section>

      <section className="quickstart">
        <pre><code>{QUICKSTART}</code></pre>
      </section>

      <section className="feature-grid">
        {features.map((feature) => (
          <Link key={feature.title} href={feature.href} className="feature-card">
            <h3>{feature.title}</h3>
            <p>{feature.description}</p>
          </Link>
        ))}
      </section>

      <section className="compare">
        <h2>Why Teams Adopt Wrkr</h2>
        <table>
          <thead>
            <tr>
              <th></th>
              <th>Without Wrkr</th>
              <th>With Wrkr</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>Long-running reliability</td>
              <td>state loss after interruption</td>
              <td>durable resume from checkpointed cursor</td>
            </tr>
            <tr>
              <td>Human supervision</td>
              <td>scrolling chat transcripts</td>
              <td>typed checkpoints with decision gates</td>
            </tr>
            <tr>
              <td>Acceptance and merge safety</td>
              <td>manual review drift</td>
              <td>deterministic acceptance results and reason codes</td>
            </tr>
            <tr>
              <td>Evidence and auditability</td>
              <td>incomplete reconstruction</td>
              <td>offline-verifiable jobpack artifacts</td>
            </tr>
          </tbody>
        </table>
      </section>

      <section className="faq">
        <h2>Frequently Asked Questions</h2>
        <div className="faq-grid">
          {faqs.map((entry) => (
            <div key={entry.question} className="faq-card">
              <h3>{entry.question}</h3>
              <p>{entry.answer}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="final-cta">
        <h2>Start with one command and supervise with confidence.</h2>
        <p>Run demo, verify jobpack, then dispatch your first real jobspec.</p>
        <Link href="/docs/install" className="btn btn-primary">
          Open Install Guide
        </Link>
        <p className="llm-link">
          For assistant and crawler resources, use{' '}
          <Link href="/llms">LLM Context</Link>.
        </p>
      </section>
    </div>
  )
}
