import fs from 'node:fs'
import path from 'node:path'

export type DocPage = {
  slug: string[]
  id: string
  title: string
  sourcePath: string
  body: string
}

const repoRoot = path.resolve(process.cwd(), '..')

const rootDocs = ['README.md', 'CONTRIBUTING.md', 'SECURITY.md']

function walkMarkdown(dir: string, relativeBase: string): string[] {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  const out: string[] = []
  for (const entry of entries) {
    if (entry.name.startsWith('.')) {
      continue
    }
    const full = path.join(dir, entry.name)
    const rel = path.join(relativeBase, entry.name)
    if (entry.isDirectory()) {
      out.push(...walkMarkdown(full, rel))
      continue
    }
    if (entry.isFile() && entry.name.endsWith('.md')) {
      out.push(rel)
    }
  }
  return out
}

function titleFromBody(sourcePath: string, body: string): string {
  const line = body.split('\n').find((v) => v.startsWith('# '))
  if (line) {
    return line.slice(2).trim()
  }
  const base = path.basename(sourcePath, '.md')
  return base.replace(/_/g, ' ')
}

function slugFromSourcePath(sourcePath: string): string[] {
  const normalized = sourcePath.replace(/\\/g, '/')
  if (normalized.startsWith('docs/')) {
    return normalized
      .slice('docs/'.length, -3)
      .split('/')
      .map((v) => v.toLowerCase())
  }
  return ['repo', normalized.slice(0, -3).toLowerCase()]
}

export function loadAllPages(): DocPage[] {
  const files = new Set<string>()
  const docsDir = path.join(repoRoot, 'docs')
  for (const rel of walkMarkdown(docsDir, 'docs')) {
    files.add(rel)
  }
  for (const rootFile of rootDocs) {
    const abs = path.join(repoRoot, rootFile)
    if (fs.existsSync(abs)) {
      files.add(rootFile)
    }
  }

  const pages: DocPage[] = []
  for (const sourcePath of files) {
    const abs = path.join(repoRoot, sourcePath)
    const body = fs.readFileSync(abs, 'utf8')
    const slug = slugFromSourcePath(sourcePath)
    pages.push({
      slug,
      id: slug.join('/'),
      title: titleFromBody(sourcePath, body),
      sourcePath,
      body,
    })
  }

  pages.sort((a, b) => a.id.localeCompare(b.id))
  return pages
}

export function loadPageBySlug(slug: string[]): DocPage | null {
  const id = slug.map((v) => v.toLowerCase()).join('/')
  for (const page of loadAllPages()) {
    if (page.id === id) {
      return page
    }
  }
  return null
}
