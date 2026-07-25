# Repository Guidelines

Short agent entrypoint. Deeper stack/architecture: [`Claude.md`](Claude.md). Ops summary: [`OPS.md`](OPS.md); operator notes: [`docs/internal/`](docs/internal/).

## Stack & layout

- **frontend/** — Vite + React SPA (`@` → `src`; do not edit `out/` or generated `auto-imports.d.ts`)
- **backend/** — Go/Gin API
- **docs/**, **scripts/**, **ops/** — specs, harness, deploy helpers

## Commands

```bash
pnpm install
pnpm dev                 # frontend :3000 (proxies uploads → :8088)
pnpm lint && pnpm type-check   # default verification
pnpm test                # Vitest (frontend)
cd backend && go test -v -race ./...
```

## Coding style

- TS/React functional components; 2 spaces; double quotes; semicolons; Tailwind utilities
- Follow existing files + ESLint; hooks/router/`useTranslation` are often auto-imported
- Go: `gofmt`; repository interface + `_impl.go` pattern

## Production site isolation (mandatory)

**Domain ≠ process.** Two public brands need two processes (port, data dir, JWT). Details: [`docs/internal/ops-lessons-site-isolation.md`](docs/internal/ops-lessons-site-isolation.md).

| Site role | Unit (example) | Port | Data |
|------|------|------|------|
| Personal blog | `inkless` | `8088` | `/opt/inkless/data/` |
| Product ops | `inkless-ops` | `8089` | `/opt/inkless-ops/data/` |

1. Before any prod write: confirm **unit + port + `DB_DSN` + reverse-proxy upstream** for the domain you mean (resolve hosts via `npc`, not hard-coded IPs in git).
2. Never cross-write independent instances (separate JWT/env/DB trees).
3. Backup before DB mutation; afterward verify **each** site via `/public/bootstrap` (theme + identity).

## Default delivery

After a coherent feature/fix (verification passes), **by default**:

1. **Commit** related changes (no secrets / unrelated dirty files).
2. **Push** to the tracking branch (usually `main`).
3. **Deploy only the intended instance** — one deploy does **not** cover both sites.

```bash
# Personal (yx.ink) — NoPanel artifact env
npc deploy impress hk-artifact --ref <branch-or-sha> --wait
```

- Product site: separate process (e.g. `inkless-ops` on `:8089` / `/opt/inkless-ops`). Do **not** “fix product” by editing another instance’s DB. See `OPS.md` + [`docs/internal/`](docs/internal/); code may share artifact symlinks, runtime state must not.
- Skip auto-deploy when the user says so, change is docs-only, or deploy readiness is blocked — then say why and stop.

## Pointers

| Need | Where |
|------|--------|
| Architecture, backend, long-agent | `Claude.md` |
| Deploy summary / dual-process rules | `OPS.md`, `docs/internal/` |
| Site isolation incident lesson | `docs/internal/ops-lessons-site-isolation.md` |
| Article AI meta / SEO (design + eval) | `docs/article-ai-meta-seo.md` |
| AI meta golden samples / scoring | `docs/article-ai-meta-golden-samples.md` |
| Frontend flags | `BASE_PATH`, `IS_PREVIEW`, … in `frontend/vite.config.ts` |
