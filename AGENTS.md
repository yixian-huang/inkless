# Repository Guidelines

Short agent entrypoint. Deeper stack/architecture: [`Claude.md`](Claude.md). Ops topology: [`OPS.md`](OPS.md).

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

**yx.ink ≠ inkless.run.** Domain ≠ process. Details: [`docs/ops-lessons-yx-ink-vs-inkless-run.md`](docs/ops-lessons-yx-ink-vs-inkless-run.md).

| Site | Role | Unit | Port | Data |
|------|------|------|------|------|
| **yx.ink** | Personal blog | `inkless` | `8088` | `/opt/inkless/data/` |
| **inkless.run** | Product ops | `inkless-ops` | `8089` | `/opt/inkless-ops/data/` |

1. Before any prod write: confirm **unit + port + `DB_DSN` + Caddy upstream** for the domain you mean.
2. Never cross-write: personal = `/opt/inkless`, product = `/opt/inkless-ops` (separate JWT/env/DB).
3. Backup before DB mutation; afterward verify **both** sites via `/public/bootstrap` (theme + identity).

## Default delivery

After a coherent feature/fix (verification passes), **by default**:

1. **Commit** related changes (no secrets / unrelated dirty files).
2. **Push** to the tracking branch (usually `main`).
3. **Deploy only the intended instance** — one deploy does **not** cover both sites.

```bash
# Personal (yx.ink) — NoPanel artifact env
npc deploy impress hk-artifact --ref <branch-or-sha> --wait
```

- Product (**inkless.run**): separate process `inkless-ops` on `:8089` / `/opt/inkless-ops`. Do **not** “fix product” by editing the personal DB. See `OPS.md` + ops lesson; code may share artifact symlinks, runtime state must not.
- Skip auto-deploy when the user says so, change is docs-only, or deploy readiness is blocked — then say why and stop.

## Pointers

| Need | Where |
|------|--------|
| Architecture, backend, long-agent | `Claude.md` |
| Deploy targets, dual-process helpers | `OPS.md` |
| yx.ink vs inkless.run incident rules | `docs/ops-lessons-yx-ink-vs-inkless-run.md` |
| Article AI meta / SEO (design + eval) | `docs/article-ai-meta-seo.md` |
| AI meta golden samples / scoring | `docs/article-ai-meta-golden-samples.md` |
| Frontend flags | `BASE_PATH`, `IS_PREVIEW`, … in `frontend/vite.config.ts` |
