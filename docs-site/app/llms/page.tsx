import type { Metadata } from 'next'
import Link from 'next/link'
import { canonicalUrl } from '../../src/lib/site'

export const metadata: Metadata = {
  title: 'LLM Context | Wrkr',
  description: 'Machine-readable and human-readable context for assistants and evaluators about Wrkr OSS.',
  alternates: { canonical: canonicalUrl('/llms/') },
}

const resources = [
  { label: 'llms.txt', href: '/llms.txt' },
  { label: 'LLM Quickstart', href: '/llm/quickstart.md' },
  { label: 'LLM Product Overview', href: '/llm/product.md' },
  { label: 'LLM Security and Safety', href: '/llm/security.md' },
  { label: 'LLM FAQ', href: '/llm/faq.md' },
  { label: 'LLM Contracts', href: '/llm/contracts.md' },
  { label: 'Crawler Policy (robots.txt)', href: '/robots.txt' },
  { label: 'AI Sitemap', href: '/ai-sitemap.xml' },
]

export default function LlmsPage() {
  return (
    <div className="llms-page not-prose">
      <h1>LLM Context</h1>
      <p>
        These resources are optimized for AI assistants, search agents, and evaluators to discover Wrkr capabilities,
        contracts, and safe usage boundaries.
      </p>
      <div className="llms-resources">
        {resources.map((resource) => (
          <Link key={resource.href} href={resource.href} className="llms-resource-link">
            {resource.label}
          </Link>
        ))}
      </div>
      <div className="llms-backlink">
        <Link href="/docs">Back to docs</Link>
      </div>
    </div>
  )
}
