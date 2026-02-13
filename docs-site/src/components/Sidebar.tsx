'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { navigation, type NavItem } from '../lib/navigation'

function NavLink({ item }: { item: NavItem }) {
  const pathname = usePathname()
  const isActive = pathname === item.href || pathname === `${item.href}/`

  return (
    <Link
      href={item.href}
      className={`sidebar-link ${isActive ? 'active' : ''}`}
    >
      {item.title}
    </Link>
  )
}

function Section({ item }: { item: NavItem }) {
  const pathname = usePathname()
  const active = item.children?.some((child) => pathname === child.href || pathname === `${child.href}/`)

  return (
    <div className="sidebar-section">
      <h3 className={active ? 'active' : ''}>{item.title}</h3>
      <div className="sidebar-links">
        {item.children?.map((child) => <NavLink key={child.href} item={child} />)}
      </div>
    </div>
  )
}

export default function Sidebar() {
  return (
    <aside className="sidebar-wrap">
      <div className="sidebar-sticky">
        <Link href="/" className="sidebar-brand">
          <div className="brand-mark">
            <span>W</span>
          </div>
          <span className="brand-text">Wrkr</span>
        </Link>

        <nav>
          {navigation.map((section) => (
            <Section key={section.title} item={section} />
          ))}
        </nav>

        <div className="sidebar-footer">
          <Link href="/llms">LLM Context</Link>
          <a href="https://github.com/davidahmann/wrkr" target="_blank" rel="noopener noreferrer">
            GitHub
          </a>
        </div>
      </div>
    </aside>
  )
}
