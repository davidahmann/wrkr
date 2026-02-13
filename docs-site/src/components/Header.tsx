'use client'

import Link from 'next/link'
import { useState } from 'react'
import { navigation } from '../lib/navigation'

export default function Header() {
  const [open, setOpen] = useState(false)

  return (
    <header className="mobile-header">
      <div className="mobile-header-row">
        <Link href="/" className="brand-link">
          <div className="brand-mark">
            <span>W</span>
          </div>
          <span className="brand-text">Wrkr</span>
        </Link>

        <button
          onClick={() => setOpen(!open)}
          className="menu-toggle"
          aria-label="Toggle navigation"
        >
          {open ? 'Close' : 'Menu'}
        </button>
      </div>

      {open && (
        <nav className="mobile-nav">
          {navigation.map((section) => (
            <div key={section.title} className="mobile-nav-section">
              <h3>{section.title}</h3>
              <div className="mobile-nav-links">
                {section.children?.map((item) => (
                  <Link
                    key={item.href}
                    href={item.href}
                    onClick={() => setOpen(false)}
                    className="mobile-nav-link"
                  >
                    {item.title}
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </nav>
      )}
    </header>
  )
}
