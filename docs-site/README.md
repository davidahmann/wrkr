# Wrkr Docs Site

Static Next.js docs site with Gait-aligned structure for UX, SEO, and AI discovery.

## Content Inputs

- `docs/**`
- `README.md`
- `CONTRIBUTING.md`
- `SECURITY.md`

## Core UX Structure

- landing page (`/`) with product narrative, feature cards, FAQ, JSON-LD
- docs ladder page (`/docs`) and doc detail pages (`/docs/<slug>`)
- responsive shell with desktop sidebar + mobile header
- client-rendered Mermaid diagrams and syntax highlighted code blocks

## SEO and AI Discovery

- OpenGraph/Twitter metadata via `app/layout.tsx`
- canonical URL helpers in `src/lib/site.ts`
- crawler and assistant assets in `public/`:
  - `robots.txt`
  - `sitemap.xml`
  - `ai-sitemap.xml`
  - `llms.txt`
  - `llm/*.md`
  - `og.svg`

## Local Development

```bash
npm ci
npm run lint
npm run build
```

Docs workflow sets `DOCS_SITE_BASE_PATH=/<repo-name>` for GitHub Pages builds.
