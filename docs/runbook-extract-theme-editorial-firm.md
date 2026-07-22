# Runbook：将 `editorial-firm` 拆为独立 GitHub 仓库

**Audience:** host maintainers  
**Theme id:** `editorial-firm` (immutable)  
**Package:** `@inkless/theme-editorial-firm`  
**Target repo (proposed):** `yixian-huang/inkless-theme-editorial-firm`  
**Contract:** v1 (`@inkless/theme-host`)  
**Reference extraction:** `blog-first` → [`inkless-theme-blog-first`](https://github.com/yixian-huang/inkless-theme-blog-first)  
**Related:** [`docs/theme-contract.md`](theme-contract.md), [`docs/superpowers/specs/2026-07-22-editorial-firm-theme-design.md`](superpowers/specs/2026-07-22-editorial-firm-theme-design.md)

---

## 0. When to run this runbook

**Do extract** when all of the following are true:

1. PR with monorepo package is merged and deployed somewhere you can smoke-test.
2. Admin can **activate** Editorial Firm; public `/` renders seed (or empty state) without white screen.
3. Section library + chrome have had at least one real content pass (or you accept freezing API at current `ef-*` set).
4. Host `THEME_CONTRACT_VERSION` is still `"1"` (or you have bumped the theme contractVersion in lockstep).

**Do not extract yet** if:

- Seed apply / UnifiedPage path is still broken on activate.
- You are mid-flight changing host facade exports consumed by the theme.
- You want to keep same-PR iteration on host + theme for another sprint.

**Status (2026-07-22):** External repo **live** —  
[`yixian-huang/inkless-theme-editorial-firm`](https://github.com/yixian-huang/inkless-theme-editorial-firm)  
Host pin example: `github:yixian-huang/inkless-theme-editorial-firm#e92a3be923ebeeb58927559b23fc98b81b5d9388`  
Monorepo `packages/theme-editorial-firm` **removed**; further visual work lands in the theme repo.

**Historical default stance (pre-cut):** monorepo `workspace:*` until UMD ready, then this runbook.

---

## 1. What stays in the monorepo forever

These **never** move to the theme repo (host ownership):

| Asset | Path / system |
|-------|----------------|
| Theme id constant | `frontend/src/plugins/builtinThemes.ts` → `EDITORIAL_FIRM` |
| Built-in registration entry | `frontend/src/plugins/themes/editorial-firm/index.ts` (thin re-export) |
| ThemeManager register | `ThemeManagerContext.tsx` |
| Backend page meta | `backend/internal/builtinthemes/pages.json` + `constants.go` |
| InstalledTheme seed | `backend/internal/seed/seed.go` |
| Activate-time UnifiedPage seed apply | `ApplyEditorialFirmPageSeeds` + embedded JSON (see §5 dual seed) |
| Host section schema merge | `frontend/src/theme/sectionSchemas.ts` (or future dynamic merge) |
| Tailwind content scan | `frontend/tailwind.config.ts` (points at package path under `node_modules`) |
| Contract inventory / UMD smoke host scripts | `scripts/theme-umd-smoke.mjs`, CI |

---

## 2. What moves to the external repo

Everything under today’s monorepo package, plus standalone build tooling:

```
inkless-theme-editorial-firm/
  package.json
  inkless.theme.json
  README.md
  LICENSE                 # match blog-first / org policy
  tsconfig.json
  vite.config.ts          # UMD + ESM (copy from blog-first, rename globals)
  types/
    theme-host-shim.d.ts  # ambient @inkless/theme-host for standalone tsc
  src/
    index.ts
    register.ts
    tokens.ts
    chrome/
    sections/
    seed/pageConfigs.ts   # TS SSOT for default sections (operators still edit CMS)
  dist/                   # build output (gitignored or release artifact)
    theme.umd.js
    theme.es.js
```

**Not copied as product source:** monorepo `src/__typecheck__/theme-host.ts` (replaced by `types/theme-host-shim.d.ts`).

---

## 3. Prerequisites (monorepo, before cut)

### 3.1 Fix known nits (recommended before first external pin)

| Nit | Why |
|-----|-----|
| `EfContactSplit` API base | Use same base as host `http` / honor `VITE_API_BASE_URL` so split-origin dev works after extraction |
| Dual seed drift | Add a small test or script that compares `pageConfigs.ts` export to `editorial_firm_seeds.json` (host) — or generate JSON from TS in CI before extract |
| README “does not replace classic” | Keep on both repos |

### 3.2 Add UMD build in monorepo package — **done (pre-extract)**

Landed under `packages/theme-editorial-firm/`:

- `vite.config.ts` — entry `src/register.ts`, UMD name `InklessThemeEditorialFirm`, externals/globals same as blog-first
- `package.json` scripts: `build` / `type-check` / `test`; `inkless.umd` / `inkless.esm`
- `inkless.theme.json` entry includes `umd` / `esm` dist paths
- Outputs: `dist/theme.umd.js`, `dist/theme.es.js`

```bash
pnpm --filter @inkless/theme-editorial-firm build
```

### 3.3 Smoke UMD (host) — **done (pre-extract)**

`scripts/theme-umd-smoke.mjs` accepts theme id (default `blog-first`):

```bash
pnpm theme:umd:smoke:editorial-firm
# or: node scripts/theme-umd-smoke.mjs editorial-firm
```

Asserts register, `manifest.id === "editorial-firm"`, contractVersion, non-empty pages.

### 3.4 Standalone typecheck — **done (pre-extract)**

- `types/theme-host-shim.d.ts` — extract-ready ambient host surface
- Monorepo still uses `src/__typecheck__/theme-host.ts` via tsconfig paths for a thin package graph
- On extract day: drop monorepo paths; `include: ["src", "types"]` like blog-first

Also landed with pre-extract:

- Contact form resolves `VITE_API_BASE_URL` / `window.__INKLESS_API_BASE__`
- Vitest `src/seed/hostSeedParity.test.ts` fingerprints TS seeds vs host `editorial_firm_seeds.json`

---

## 4. Extraction procedure (cut day)

### Phase A — Create external repo

1. Create empty GitHub repo `inkless-theme-editorial-firm` (public/private per org).
2. From monorepo (clean tree on the commit you want to freeze):

```bash
# Example: subtree-ish copy without git history of the whole monorepo
mkdir -p /tmp/inkless-theme-editorial-firm
rsync -a --exclude node_modules --exclude dist \
  packages/theme-editorial-firm/ /tmp/inkless-theme-editorial-firm/

cd /tmp/inkless-theme-editorial-firm
# Add vite.config.ts + types/theme-host-shim.d.ts if not already present
# Set package.json repository.url to the new GitHub URL
git init
git add .
git commit -m "chore: initial import of @inkless/theme-editorial-firm from inkless monorepo"
git branch -M main
git remote add origin git@github.com:yixian-huang/inkless-theme-editorial-firm.git
git push -u origin main
```

3. Tag a release-ready commit:

```bash
git tag v0.1.0
git push origin v0.1.0
# Record full SHA for host pin:
git rev-parse HEAD
```

4. CI on theme repo (minimal):

- `pnpm install`
- `pnpm type-check`
- `pnpm test`
- `pnpm build` (produces `dist/`)

Optional: attach `dist/theme.umd.js` as release asset for pure-URL install.

### Phase B — Point host monorepo at GitHub pin

In `frontend/package.json`:

```json
"@inkless/theme-editorial-firm": "github:yixian-huang/inkless-theme-editorial-firm#<FULL_SHA>"
```

Remove or stop using:

```json
"@inkless/theme-editorial-firm": "workspace:*"
```

Then:

```bash
pnpm install
pnpm -C frontend type-check
pnpm -C frontend test:run
pnpm build:theme-editorial-firm   # should type-check/build via node_modules package
```

Update root scripts if needed:

```json
"build:theme-editorial-firm": "cd frontend/node_modules/@inkless/theme-editorial-firm && npm install --ignore-scripts --no-fund --no-audit && npm run build"
```

(align with blog-first / product-first style).

### Phase C — Remove monorepo package source

Only after host pin works:

1. Delete `packages/theme-editorial-firm/` from monorepo (or leave a one-line README pointing to the external repo for one release cycle — optional soft delete).
2. Ensure `pnpm-workspace.yaml` still has `packages/*` but empty dir is gone.
3. Update Tailwind content globs to scan `node_modules/@inkless/theme-editorial-firm/src/**` (already typical for blog-first).
4. Keep thin host entry:

```ts
// frontend/src/plugins/themes/editorial-firm/index.ts
import { createEditorialFirmTheme } from "@inkless/theme-editorial-firm";
export const editorialFirmTheme = createEditorialFirmTheme();
// re-exports as needed for tests
```

5. Commit host PR:  
   `chore(theme): consume editorial-firm from GitHub pin`

### Phase D — Dual install path (optional same PR)

| Path | Use |
|------|-----|
| **Built-in** | `registerBuiltIn(editorialFirmTheme)` from pinned package |
| **Remote UMD** | Admin `externalUrl` → host loads `theme.umd.js` → `__INKLESS_THEME_REGISTER__` |

Remote path requires host already supporting external themes (same as blog-first). Verify once with a static file server URL of `dist/theme.umd.js`.

---

## 5. Dual seed problem (TS vs backend JSON)

Today:

| Source | Location | Consumer |
|--------|----------|----------|
| TS configs | package `src/seed/pageConfigs.ts` | docs, tests, future generators |
| JSON embed | monorepo `backend/internal/builtinthemes/editorial_firm_seeds.json` | Go activate path |

**After extract:**

- **Backend JSON stays in host monorepo** (Go embed cannot depend on a JS package at runtime).
- **TS seed stays in theme repo** as authoring SSOT for “what the theme wants.”
- **Sync rule (pick one and document in both READMEs):**

**Option A (recommended for v1):** Host owns apply JSON; theme repo has `seed/pageConfigs.ts` for documentation + theme tests only. When changing defaults:

1. Edit theme repo seed (PR on theme).
2. Copy/export into host `editorial_firm_seeds.json` (host PR with pin bump).
3. Never auto-overwrite non-empty published pages (existing behavior).

**Option B:** Script in theme repo `pnpm run export-seeds` → writes JSON; host CI fails if JSON differs from committed embed when pin updates.

Do **not** leave two silent sources without a checklist step in the release process.

---

## 6. Host release checklist (every theme bump)

When publishing a new editorial-firm version to production host:

1. Theme repo: green CI, tag `vX.Y.Z`, note SHA.
2. Host: bump pin `#SHA` in `frontend/package.json`.
3. `pnpm install` + lint + type-check + frontend tests + contractAlignment for editorial-firm.
4. If seeds changed: update `editorial_firm_seeds.json` + Go tests for EditorialFirm.
5. Deploy host; **do not** expect existing sites with content to pick up new seeds (by design).
6. Smoke: activate on empty site OR new env → four pages + sections render.
7. Smoke: corporate-classic still activates and renders (isolation).

---

## 7. Versioning policy

| Change | Bump | Host action |
|--------|------|-------------|
| Visual / section CSS only | patch | pin bump optional but recommended |
| New optional section type `ef-*` | minor | pin bump; merge schemas in host if builder needs them |
| Breaking prop rename / remove section | major | pin bump only after host admin/content migration note |
| Needs new host facade export | **host contract first** | bump theme-host inventory + contract if breaking; then theme |

Theme `contractVersion` must remain in `THEME_CONTRACT_SUPPORTED` on host.

---

## 8. Rollback

| Failure | Action |
|---------|--------|
| Bad theme pin breaks build | Revert `package.json` pin to previous SHA; redeploy |
| Runtime white screen after pin | Same; keep previous `dist` if using CDN UMD |
| Need emergency monorepo again | Re-vendor: copy tag tree back into `packages/theme-editorial-firm` + `workspace:*` for one hotfix (document temporary) |

Theme id never changes during rollback.

---

## 9. Definition of done (extraction complete)

- [ ] External repo exists with green CI (type-check, test, build).
- [ ] Host depends on `github:...#sha` (not `workspace:*`).
- [ ] Monorepo no longer requires `packages/theme-editorial-firm` source to build.
- [ ] UMD smoke passes for editorial-firm (or shared smoke matrix includes it).
- [ ] Activate + public home smoke on a clean DB.
- [ ] Corporate Classic regression smoke.
- [ ] Both READMEs + this runbook link each other; theme-contract mentions external repo when live.
- [ ] Seed sync rule chosen (A or B) and written in theme README + host runbook.

---

## 10. Suggested calendar

| Day | Action |
|-----|--------|
| 0 | Merge monorepo theme PR; deploy |
| 0–7 | Soak: activate, edit sections, contact form, bilingual |
| 7–10 | Land UMD + shim + contact baseURL nit in monorepo package |
| 10 | Extract repo (Phase A) |
| 10–11 | Host pin PR (Phase B–C) |
| 11+ | Theme-only PRs for visual work; host pins on release |

---

## 11. Explicit non-goals of extraction

- Do **not** extract `corporate-classic` in the same cut (different maturity / hardcode pages).
- Do **not** move backend `pages.json` or Go seed apply into the theme repo.
- Do **not** rename theme id `editorial-firm`.
- Do **not** change `DEFAULT_FALLBACK_THEME_ID` as part of extraction.

---

## 12. Quick command cheat sheet

```bash
# Monorepo (today)
pnpm --filter @inkless/theme-editorial-firm type-check
pnpm --filter @inkless/theme-editorial-firm test

# After UMD exists
pnpm --filter @inkless/theme-editorial-firm build

# Host consume pin
# frontend/package.json → "github:yixian-huang/inkless-theme-editorial-firm#<sha>"
pnpm install
pnpm -C frontend type-check && pnpm -C frontend test:run

# Backend seed tests (host)
cd backend && go test ./internal/service/ -count=1 -run EditorialFirm
```
