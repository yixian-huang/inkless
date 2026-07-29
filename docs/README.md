# Documentation index

Inkless documentation lives in three places:

| Location | Purpose |
|----------|---------|
| **This tree (`docs/`)** | Deep specs, architecture, deploy guides, ADRs |
| **`docs-site/`** | VitePress “getting started” and extension guides |
| **Root markdown** | README, CONTRIBUTING, SECURITY, OPS summary, CHANGELOG |

## Start here

| If you want to… | Read |
|-----------------|------|
| Run locally in 5 minutes | [../README.md](../README.md) · [../docs-site/guide/getting-started.md](../docs-site/guide/getting-started.md) |
| Contribute code | [../CONTRIBUTING.md](../CONTRIBUTING.md) |
| Understand layering | [architecture.md](architecture.md) · [developer-guide.md](developer-guide.md) |
| Deploy to a server | [deployment.md](deployment.md) · [docker-setup.md](docker-setup.md) |
| Theme packages / host boundary | [theme-contract.md](theme-contract.md) · [adr/0002-theme-host-boundary.md](adr/0002-theme-host-boundary.md) |
| Official theme store (Phase A) | [design-official-extension-store-phase-a.md](design-official-extension-store-phase-a.md) |
| Report a vulnerability | [../SECURITY.md](../SECURITY.md) |

## Public guides (recommended)

- [api-spec.md](api-spec.md) — REST contract overview  
- [api-versioning.md](api-versioning.md)  
- [data-model.md](data-model.md)  
- [deployment.md](deployment.md)  
- [docker-setup.md](docker-setup.md)  
- [developer-guide.md](developer-guide.md)  
- [theme-contract.md](theme-contract.md)  
- [testing-strategy.md](testing-strategy.md)  
- [adr/0001-single-instance-single-site.md](adr/0001-single-instance-single-site.md)  
- [adr/0002-theme-host-boundary.md](adr/0002-theme-host-boundary.md) — Host vs Theme ownership  
- [design-official-extension-store-phase-a.md](design-official-extension-store-phase-a.md) — Official theme one-click install  

## Product / research (may lag code)

- [product-roadmap.md](product-roadmap.md) — capability map + backlog (historical sections remain)  
- [business-requirements.md](business-requirements.md)  
- [article-ai-meta-seo.md](article-ai-meta-seo.md)  
- [agent-access.md](agent-access.md) — local agents via API Key (no direct DB)  
- [agent-fleet.schema.json](agent-fleet.schema.json) — multi-site fleet registry schema  
- [examples/agent-fleet.example.json](examples/agent-fleet.example.json) — sample multi-site fleet  
- [agent-mcp.md](agent-mcp.md) — MCP server (`inkless mcp serve`, 2026-07-28)  
- [design-inkless-mcp.md](design-inkless-mcp.md) — MCP design  
- [picgo.md](picgo.md) — PicGo / media API Key 

## Internal (maintainers)

Operator runbooks and incident notes with **no committed secrets**:

→ **[internal/README.md](internal/README.md)**

These are optional for forks. End users should not need them.

## Superpowers plans / specs

Historical implementation plans under [`superpowers/`](superpowers/) are retained
for maintainers. They are **not** the public product docs entrypoint.

## VitePress site

```bash
cd docs-site
pnpm install
pnpm dev      # local preview
pnpm build    # static site
```

Publishing to GitHub Pages or `docs.inkless.run` is a separate ops step; see
[`.github/REPO_SETUP.md`](../.github/REPO_SETUP.md).
