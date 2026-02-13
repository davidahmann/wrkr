import Link from 'next/link'
import { notFound } from 'next/navigation'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { loadAllPages, loadPageBySlug } from '../../../src/lib/content'
import { navigation } from '../../../src/lib/navigation'

export function generateStaticParams() {
  return loadAllPages().map((page) => ({ slug: page.slug }))
}

export default async function DocPage({ params }: { params: Promise<{ slug: string[] }> }) {
  const { slug } = await params
  const page = loadPageBySlug(slug)
  if (!page) {
    notFound()
  }

  return (
    <main className="layout">
      <header className="header">
        <h1>Wrkr Docs</h1>
        <p className="meta">Source: {page.sourcePath}</p>
      </header>
      <div className="shell">
        <aside className="nav">
          {navigation.map((section) => (
            <div key={section.title}>
              <h3>{section.title}</h3>
              <ul>
                {section.items.map((item) => (
                  <li key={item.href}>
                    <Link href={item.href}>{item.label}</Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </aside>
        <article className="content">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{page.body}</ReactMarkdown>
        </article>
      </div>
    </main>
  )
}
