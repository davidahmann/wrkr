# Wrkr Docs Site

Static Next.js docs site that ingests:

- `docs/**`
- `README.md`
- `CONTRIBUTING.md`
- `SECURITY.md`

## Local

```bash
npm ci
npm run lint
npm run build
```

## GitHub Pages base path

When exporting for repository-scoped GitHub Pages, set:

```bash
DOCS_SITE_BASE_PATH=/<repo-name> npm run build
```

The docs workflow sets this automatically to `/${{ github.event.repository.name }}`.
