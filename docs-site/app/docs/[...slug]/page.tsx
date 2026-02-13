import { notFound } from 'next/navigation'
import type { Metadata } from 'next'
import MarkdownRenderer from '../../../src/components/MarkdownRenderer'
import { getAllDocSlugs, getDocContent } from '../../../src/lib/docs'
import { markdownToHtml } from '../../../src/lib/markdown'
import { canonicalUrl } from '../../../src/lib/site'

interface PageProps {
  params: Promise<{ slug: string[] }>
}

export function generateStaticParams() {
  return getAllDocSlugs().map((slug) => ({
    slug: slug.split('/'),
  }))
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const resolvedParams = await params
  const slugPath = resolvedParams.slug.join('/').toLowerCase()
  const doc = getDocContent(slugPath)

  if (!doc) {
    return {
      title: 'Not Found | Wrkr Docs',
      description: 'Requested documentation page was not found.',
    }
  }

  const canonical = canonicalUrl(`/docs/${slugPath}/`)

  return {
    title: `${doc.title} | Wrkr Docs`,
    description: doc.description || `Wrkr documentation page: ${doc.title}`,
    alternates: { canonical },
    openGraph: {
      title: `${doc.title} | Wrkr Docs`,
      description: doc.description || `Wrkr documentation page: ${doc.title}`,
      url: canonical,
      type: 'article',
    },
  }
}

export default async function DocPage({ params }: PageProps) {
  const resolvedParams = await params
  const slugPath = resolvedParams.slug.join('/').toLowerCase()
  const doc = getDocContent(slugPath)
  if (!doc) {
    notFound()
  }

  const html = markdownToHtml(doc.content, slugPath)

  return (
    <div>
      <h1 className="doc-title">{doc.title}</h1>
      <MarkdownRenderer html={html} />
    </div>
  )
}
