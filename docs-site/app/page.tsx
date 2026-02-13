import Link from 'next/link'
import { navigation } from '../src/lib/navigation'

export default function HomePage() {
  return (
    <main className="layout">
      <header className="header">
        <h1>Wrkr Docs</h1>
        <p className="meta">Dispatch and supervision for long-running agent jobs</p>
      </header>
      <section className="content">
        <p>
          This docs site renders repository markdown from <code>docs/</code> and core root docs.
        </p>
        {navigation.map((section) => (
          <div key={section.title}>
            <h2>{section.title}</h2>
            <ul>
              {section.items.map((item) => (
                <li key={item.href}>
                  <Link href={item.href}>{item.label}</Link>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </section>
    </main>
  )
}
