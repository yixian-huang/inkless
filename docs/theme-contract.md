# Inkless Theme Contract

**Status**: Draft (ADR-level)  
**Audience**: Theme authors + Inkless core maintainers  
**Related**: `frontend/src/plugins/types.ts`, `ThemeManager`, `externals.ts`

## 1. Purpose

Inkless is the **host**: CMS data, auth, admin, public APIs, and a stable theme runtime.

Themes (e.g. **blog-first**) own **presentation**: tokens, chrome, home information architecture, and optional page components.

Themes must not fork CMS business logic. Host must not hardcode personal-site hero copy.

## 2. ThemePlugin surface

A theme is a `ThemePlugin` registered with the host:

| Field | Required | Notes |
|-------|----------|--------|
| `manifest` | yes | `id`, names, version, `type: "theme"` |
| `defaultTokens` | yes | `ThemeTokens` |
| `pages` | yes | At least home if the theme replaces `/` |
| `layoutChrome` | recommended | `Header` / `Footer` components |
| `settingSchema` | optional | Admin-editable theme options |
| `tokenPresets` | optional | Named color/font packs |
| `sections` / `sectionMetas` | optional | Builder blocks |

### 2.1 Registration

**Built-in (bundled with host)**

```ts
themeManager.registerBuiltIn(theme);
```

**External (UMD script URL)**

```ts
// Host loads script; theme calls:
window.__INKLESS_THEME_REGISTER__(themePlugin);
```

Shared host libraries (do **not** bundle these in the theme):

```ts
window.__INKLESS_SHARED__ = { React, ReactDOM, ReactRouterDOM, ReactI18next };
```

Contract version: themes should declare peer compatibility in `inkless.theme.json` (see §5).

## 3. Ownership matrix

| Concern | Inkless (host) | Theme |
|---------|----------------|--------|
| Articles / media / comments APIs | ✓ | — |
| Article list / post body primitives | ✓ (shared UI) | styles via tokens |
| Home hero layout & density | — | ✓ |
| Header/Footer structure | shell + settings | chrome components |
| Site name / tagline / logo URLs | site config | reads via hooks |
| Tokens (color, type, max-width) | merge/publish | defaults + presets |
| Admin install theme from URL | ✓ | ships UMD |

### 3.1 Routes

| Route | Owner |
|-------|--------|
| `/` (theme home) | Theme `pages[]` |
| `/blog`, `/blog/:slug` | Host |
| `/categories/*`, `/tags/*` | Host |
| Dynamic CMS pages | Host + sections |

## 4. Tokens

Themes must use host CSS variables (applied by `ThemeProvider`), not hardcoded brand hex in components (except optional decorative monogram backgrounds that can fall back to `var(--color-on-surface)`).

Required token groups: `colors`, `fonts`, `layout` (see `frontend/src/theme/tokens.ts`).

Host merges published site theme config over theme defaults.

## 5. Package / GitHub layout (target)

```
inkless-theme-blog-first/
  inkless.theme.json    # { "id", "version", "contractVersion", "umd", "esm" }
  dist/theme.umd.js
  dist/theme.es.js
  src/...
```

Install paths:

1. **Default built-in** — host depends on a pinned release (npm/git) and `registerBuiltIn`.
2. **Remote** — admin sets `installed_themes.externalUrl` → `loadExternal(url)`.

## 6. Extraction roadmap (blog-first)

1. **UX polish** — compact home hero; socials only in header.  
2. **This contract doc** + freeze public hook list.  
3. **Move home page into theme folder** (break reverse import from `@/pages/blog-home`).  
4. **Monorepo package** `packages/theme-blog-first`.  
5. **Separate GitHub repo** + pin in Inkless; document Release assets.

## 7. Non-goals

- Themes shipping their own React copy.
- Themes calling admin APIs with secrets.
- Personal site copy inside theme source (use site config).

## 8. Success criteria

- New themes installable via URL without core PR.
- blog-first default for blank sites remains.
- Style PRs land primarily in the theme package/repo.
