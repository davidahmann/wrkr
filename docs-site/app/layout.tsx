import type { Metadata } from 'next'
import './globals.css'
import Sidebar from '../src/components/Sidebar'
import Header from '../src/components/Header'
import { SITE_BASE_PATH, SITE_ORIGIN } from '../src/lib/site'

export const metadata: Metadata = {
  metadataBase: new URL(`${SITE_ORIGIN}${SITE_BASE_PATH}`),
  title: 'Wrkr | Dispatch and Supervision for Long-Running Agent Jobs',
  description:
    'Wrkr makes multi-hour agent jobs operable with durable dispatch, checkpoints, approvals, budgets, acceptance checks, and verifiable jobpacks.',
  keywords:
    'agent dispatch, long running agents, job checkpointing, jobpack, acceptance harness, ai agent supervision, durable execution, developer tooling',
  openGraph: {
    title: 'Wrkr | Durable Dispatch for Long-Running Agents',
    description:
      'Submit once, supervise by checkpoints, and verify offline. Wrkr provides dispatch, checkpoint, accept, and jobpack primitives.',
    url: 'https://davidahmann.github.io/wrkr',
    siteName: 'Wrkr',
    type: 'website',
    images: [
      {
        url: '/og.svg',
        width: 1200,
        height: 630,
        alt: 'Wrkr',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Wrkr | Durable Dispatch for Long-Running Agents',
    description:
      'Dispatch long-running agent jobs with checkpoint supervision and verifiable jobpacks.',
    images: ['/og.svg'],
  },
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="site-body">
        <Header />
        <div className="site-shell">
          <Sidebar />
          <main className="site-main">
            <article>{children}</article>
          </main>
        </div>
      </body>
    </html>
  )
}
