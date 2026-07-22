# Design: Editorial Firm Theme

**Date:** 2026-07-22  
**Status:** Approved for planning (design review complete)  
**Related:** `docs/theme-contract.md`, `packages/theme-corporate-classic`, `frontend/src/theme/DynamicPage.tsx`

## 1. Problem

`corporate-classic` grew as a **bespoke consulting site** (印迹 / Blotting): seven hardcoded pages, blue–green tokens, and page UI living in the host (`frontend/src/pages/*`). It works for that customer but is a poor default for a **generic, beautiful enterprise/agency** product theme.

We need a second corporate-line theme that is:

- more **classic and editorial** in visual quality,
- **content-driven** (CMS sections, not hardcoded React pages),
- **isolated** from existing `corporate-classic` production sites.

## 2. Goals and non-goals

### Goals

1. New built-in theme **`editorial-firm`** with magazine / brand-atelier aesthetics.
2. Lean IA: **Home / About / Services / Contact** only.
3. Implementation depth: **theme package + section library**; pages are `renderMode: "dynamic"`.
4. Zero forced migration for `corporate-classic` sites (e.g. blottingconsultancy.com).

### Non-goals (v1)

- Do not modify or delete `corporate-classic` behavior.
- Do not extract a shared `corporate-base` package yet.
- Do not bump theme contract to v2.
- Do not change `DEFAULT_FALLBACK_THEME_ID` (remains `corporate-classic`).
- No Google Fonts hard dependency, no map widget, no dedicated Cases/Team routes.
- No automatic cutover of 印迹 / Blotting to the new theme.

## 3. Decisions (locked)

| Topic | Decision |
|-------|----------|
| Coexistence | **New theme** (not in-place upgrade of classic) |
| IA | Four pages: home, about, services, contact |
| Aesthetic | Editorial magazine / brand atelier (bold type, large imagery, strong grid) |
| Implementation | Theme package + `ef-*` section library; config-driven pages |
| Approach | **A** — new package; classic untouched |

## 4. Identity and host boundary

| Item | Value |
|------|--------|
| Theme id | `editorial-firm` |
| Package | `@inkless/theme-editorial-firm` |
| Path | `packages/theme-editorial-firm` |
| Names | Editorial Firm / 编辑机构 |
| Contract | `"1"` (`@inkless/theme-host`) |

### vs `corporate-classic`

| | `corporate-classic` | `editorial-firm` |
|--|---------------------|------------------|
| Customers | Existing (印迹, etc.) | New sites / optional switch |
| IA | 7 hardcoded pages | 4 dynamic pages |
| Page UI | Host `frontend/src/pages/*` | `DynamicPage` + theme sections |
| Migration | Unchanged | Independent seed + `pages.json` entry |
| Fallback default | Still `DEFAULT_FALLBACK_THEME_ID` | Not the system fallback in v1 |

### Host integration (minimal)

1. `BUILTIN_THEME_IDS.EDITORIAL_FIRM = "editorial-firm"`.
2. `backend/internal/builtinthemes/pages.json`: four pages, all `renderMode: "dynamic"`.
3. Frontend `registerBuiltIn` + contract-alignment / pages-meta tests (same pattern as blog/product).
4. Package exports `createEditorialFirmTheme()`, chrome, tokens/presets, section map.
5. Optional demo seed (neutral bilingual copy); **never** write into 印迹 production DB as part of this work.

Themes must not deep-import `@/…` from the host app (theme-contract rule). Loaders only if a future hardcoded page is unavoidable; v1 expects none.

## 5. Visual system

### Default preset: `ink-editorial`

| Token | Value | Role |
|-------|--------|------|
| `primary` | `#111111` | Nav, primary actions, heavy titles |
| `primaryDark` | `#000000` | Hover / deepen |
| `accent` | `#C45C26` | Sparse accent (links, labels) — warm terracotta, not classic blue-green |
| `accentHover` | `#A34A1C` | |
| `surface` | `#FAF8F5` | Paper-like ground |
| `surfaceAlt` | `#F0EBE3` | Alternating bands |
| `onPrimary` | `#FFFFFF` | |
| `onSurface` | `#1A1A1A` | Body |
| `onSurfaceMuted` | `#5C5C5C` | Secondary |
| `border` | `#E5DFD6` | Hairline rules |

**Second preset (v1):** `noir-gallery` — high-contrast black/white + single accent.

Optional later: `slate-atelier` (cool slate + copper).

### Typography

| Role | Stack |
|------|--------|
| `heading` | `"Iowan Old Style", "Palatino Linotype", Palatino, "Songti SC", "Noto Serif SC", serif` |
| `sans` | `system-ui, -apple-system, "PingFang SC", "Noto Sans SC", sans-serif` |

No required network webfonts in v1. Optional webfont toggle may be added via `settingSchema` later (default off).

### Layout tokens

| Token | Value | Intent |
|-------|--------|--------|
| `maxWidth` | `1280px` | Wider magazine grid |
| `borderRadius` | `0`–`0.125rem` | Near-square, editorial |
| `contentPadding` | `1.25rem` / md `2rem` | |
| `sectionSpacing` | `6rem` (mobile ~`3.5rem`) | Generous vertical rhythm |
| `contentGap` | `2.5rem` | Loose grid |

### Chrome principles

- **Header:** light/transparent → solid on scroll; bookish tracking; mobile hamburger.
- **Footer:** large wordmark / short tagline + hairline + small copyright; sparse link row — **not** classic full primary-blue footer bar.
- Logo via `useBranding()`; if missing, **uppercase wordmark** from `siteName`.
- Brand color may lightly override CSS variables when host already supports `siteConfig.brand.primaryColor`.

## 6. Information architecture

| slug | contentKey | renderMode | Nav |
|------|------------|------------|-----|
| `home` | `home` | `dynamic` | Header + Footer |
| `about` | `about` | `dynamic` | Header + Footer |
| `services` | `services` | `dynamic` | Header + Footer |
| `contact` | `contact` | `dynamic` | Header + Footer |

Notes:

- No `advantages` / `cases` / `experts` routes in v1; use sections on home/about for portfolio or people.
- Services slug is **`services`**, not classic’s `core-services`, to avoid content-key coupling.
- Routing uses existing theme pages + `DynamicPage` (host already supports dynamic public pages).

## 7. Section library

All types are **theme-scoped** with prefix `ef-` to avoid colliding with host globals (`hero`, `text-image`, …).

| type | Purpose | Key props | Visual |
|------|---------|-----------|--------|
| `ef-hero-editorial` | Cover / page open | `kicker`, `title`, `deck`, `image`, `ctaLabel`, `ctaHref`, `layout: full\|split` | Serif display + short deck; full-bleed or split |
| `ef-pull-quote` | Quote | `quote`, `attribution` | Oversized type, narrow measure |
| `ef-feature-split` | Story band | `title`, `body`, `image`, `imageSide`, `caption` | Asymmetric ~5/7 grid |
| `ef-service-index` | Offerings | `title`, `intro`, `items[]{title, summary, href?}` | Numbered list + rules, not card walls |
| `ef-mosaic` | Image grid | `title?`, `tiles[]{image, label?, href?}` | 2–3 column mosaic |
| `ef-cta-band` | Conversion | `title`, `body?`, `ctaLabel`, `ctaHref` | Full-width band, single primary CTA |
| `ef-contact-split` | Contact | labels, `showForm`, contact fields | Copy left, form right; host contact API or mailto fallback |
| `ef-rich-text` | Long copy | `title?`, `body` | `max-w-prose` reading column |

**Optional (not blocking v1):** `ef-logo-strip`, `ef-stats-inline`, `ef-person-row`.

**Explicitly not forked into the package:** full host section set. Only contact wiring reuses host submit APIs when available.

### Admin / schema

- Each `ef-*` exposes `sectionMetas` and field schemas using existing field types (`bilingual`, `media`, `array`, …).
- If the page builder only reads global `sectionSchemas`, v1 either:
  - relies on theme `sectionMetas` where sufficient, or
  - **merges** `ef-*` keys into the host schema map (additive only; no change to other themes’ types).

## 8. Default seed compositions

Neutral bilingual placeholder copy only — **no** 印迹 / Blotting branding.

**Home:**  
`ef-hero-editorial` (full) → `ef-feature-split` → `ef-service-index` (3 items → `/services`) → `ef-pull-quote` → `ef-cta-band` → `/contact`

**About:**  
`ef-hero-editorial` (split) → `ef-feature-split` ×1–2 → `ef-mosaic` → `ef-cta-band`

**Services:**  
short hero or kicker → full `ef-service-index` → optional deep `ef-feature-split` → `ef-cta-band`

**Contact:**  
`ef-contact-split` (+ optional quote; no map in v1)

## 9. Implementation slices

| Slice | Scope | Done when |
|-------|--------|-----------|
| **P0 Skeleton** | Package, tokens, 2 presets, chrome, host register, `pages.json` | Theme selectable; four routes open without white screen |
| **P1 Sections** | Eight `ef-*` components + metas/schemas | Seed/home config renders fully |
| **P2 Seed** | Four-page default configs + activation write path aligned with other built-ins | Fresh activate shows complete magazine home |
| **P3 Polish** | Responsive nav, a11y basics, contact API/mailto, tests | Lint/type-check + alignment tests green |
| **P4 Docs** | Package README; one-line note in theme-contract appendix if needed | Authors know how to extend sections/seed |

## 10. Acceptance criteria

1. **Isolation:** `corporate-classic` metas/tests and production path unchanged.
2. **Selectable:** Admin lists Editorial Firm; bootstrap `activeTheme.themeId === "editorial-firm"` when activated.
3. **Four pages:** dynamic home/about/services/contact; header/footer nav correct.
4. **Look:** default paper surface + serif headings + near-square radius; not classic blue-green defaults.
5. **Content-driven:** editing bilingual section props changes the public site without code.
6. **Resilience:** unknown section type → DEV warning / PROD skip; no object-as-React-child regressions (locale normalize).
7. **Quality gate:** `pnpm lint && pnpm type-check`; new contract/pages alignment tests.

## 11. Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Empty DB → blank dynamic pages | P2 seed; friendly empty state if config missing |
| Builder ignores `ef-*` schemas | Theme metas first; additive host schema merge if required |
| Contact API coupled to classic | Detect API; mailto / static fallback |
| Poor CJK serif fallback | Heading stack includes Songti / Noto Serif SC; body stays sans |
| Bundle size | Section-level code split; no unrelated deps |
| Dual section systems | Strict `ef-` prefix; document ownership |

## 12. Success statement

> A new site can select **Editorial Firm** and get a four-page, magazine-quality company site, fully edited via CMS sections, while **Corporate Classic** remains the stable theme for existing consulting customers.

## 13. Open implementation notes (for plan, not re-decided)

- Exact seed persistence hook (which backend seed/activate path) to mirror for built-in themes.
- Whether `services` contentKey needs any backend allowlist beyond `pages.json`.
- Contact form endpoint discovery (existing public contact handler vs mailto-only v1).
- PR plan granularity (P0–P4 above is the intended DAG order).
