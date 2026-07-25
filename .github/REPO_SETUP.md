# GitHub repository listing (maintainers)

These fields are **not** fully controlled by git; set them once in the GitHub UI
or with `gh`. Suggested values for [yixian-huang/inkless](https://github.com/yixian-huang/inkless):

## About

| Field | Suggested value |
|-------|-----------------|
| Description | Self-hosted bilingual (zh/en) CMS — Go API + React, themes, plugins, SQLite/Postgres |
| Homepage | `https://inkless.run` |
| Topics | `cms`, `golang`, `react`, `self-hosted`, `sqlite`, `postgresql`, `headless-cms`, `vite`, `bilingual` |

```bash
gh repo edit yixian-huang/inkless \
  --description "Self-hosted bilingual (zh/en) CMS — Go API + React, themes, plugins, SQLite/Postgres" \
  --homepage "https://inkless.run" \
  --add-topic cms --add-topic golang --add-topic react \
  --add-topic self-hosted --add-topic sqlite --add-topic postgresql \
  --add-topic headless-cms --add-topic vite --add-topic bilingual
```

## Features to enable

- **Issues** — on (bug / feature / plugin templates already present)  
- **Discussions** — optional; useful for Q&A without issue noise  
- **Security advisories** — on (see `SECURITY.md`)  
- **Branch protection** on `main` — require Quality Gate checks  

## Releases

```bash
# After merging release-ready work:
git tag -a v0.1.0-alpha.2 -m "Inkless v0.1.0-alpha.2"
git push origin v0.1.0-alpha.2
# release.yml builds and uploads artifacts to the GitHub Release
```

Update [CHANGELOG.md](../CHANGELOG.md) before tagging.

## Docs site hosting (optional)

Build VitePress (`cd docs-site && pnpm build`) and publish `docs-site/.vitepress/dist`
to GitHub Pages or your CDN. Point README “Documentation” at the public URL once live.

## Deploy workflow vs release workflow

| Workflow | Audience | Purpose |
|----------|----------|---------|
| `quality-gate.yml` | Everyone | PR / push CI |
| `release.yml` | Community | Tag → downloadable artifacts |
| `deploy.yml` | Maintainers only | Optional CD to *your* hosts (needs secrets); leave `AUTO_DEPLOY_ENABLED` unset for pure upstream |
