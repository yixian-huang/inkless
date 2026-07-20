# Inkless Theme Contract

**Status**: Locked (v1)  
**Audience**: Theme authors + Inkless core maintainers  
**Related**: `frontend/src/theme-host/`, `frontend/src/plugins/types.ts`, `ThemeManager`, `externals.ts`

## 1. Purpose

Inkless is the **host**: CMS data, auth, admin, public APIs, and a stable theme runtime.

Themes (e.g. **blog-first**) own **presentation**: tokens, chrome, home information architecture, and optional page components.

Themes must not fork CMS business logic. Host must not hardcode personal-site hero copy.

## 2. Contract version lock

| Constant | Location | Current |
|----------|----------|---------|
| `THEME_CONTRACT_VERSION` | `@inkless/theme-host` / `theme-host/contract.ts` | **`"1"`** |
| `THEME_CONTRACT_SUPPORTED` | same | `["1"]` |
| Theme declaration | `ThemePlugin.contractVersion` + `inkless.theme.json#contractVersion` | must be supported |

**Rules**

1. Themes set `contractVersion` on the plugin object (and mirror it in `inkless.theme.json`).
2. `ThemeManager.registerBuiltIn` / `registerExternal` call `assertThemeContractCompatible`.
3. Missing `contractVersion` is treated as `"1"` **only while** the host still supports `"1"` (legacy UMD grace).
4. **Breaking** host facade changes (remove/rename export or change semantics) → bump `THEME_CONTRACT_VERSION` and update `THEME_CONTRACT_SUPPORTED` deliberately.
5. **Additive** exports may stay on the same contract major version; still update the inventory.

CI enforces inventory parity and UMD smoke (`scripts/theme-umd-smoke.mjs`).

## 3. Host facade export inventory (`@inkless/theme-host`)

Source of truth: `frontend/src/theme-host/exports.inventory.ts`  
Runtime re-export barrel: `frontend/src/theme-host/index.ts`  
Globals for UMD: `window.InklessThemeHost` and `window.__INKLESS_SHARED__.host`

### 3.1 Value exports (runtime)

| Export | Role |
|--------|------|
| `THEME_CONTRACT_VERSION` | Host contract major |
| `THEME_CONTRACT_SUPPORTED` | Accepted contract versions |
| `THEME_HOST_VALUE_EXPORTS` / `THEME_HOST_TYPE_EXPORTS` | Inventory lists |
| `normalizeThemeContractVersion` | Parse version string |
| `isThemeContractCompatible` | Boolean check |
| `resolveThemeContractVersion` | Effective version (legacy default) |
| `assertThemeContractCompatible` | Throw if incompatible |
| `BLOG_DEFAULT_LAYOUT` | Reading layout default |
| `BaseSiteHeader` | Header shell |
| `BrandMark` | Logo / avatar / text brand |
| `HeaderUtilities` | RSS + socials |
| `useHeaderSettings` | brandMode / showRss / showSocials |
| `useBranding` | Site identity view |
| `useContentMaxWidth` | Theme max width |
| `useIsReadingLayout` | contentProfile === reading |
| `useIsThemeHomePath` | Path is theme home |
| `useGlobalConfig` | Published global config |
| `useSEODefaults` | Title / description helpers |
| `useLocaleMode` | Locale mode triple |
| `SeoHead` | Document meta |
| `BlogPageShell` | Content column shell |
| `AuthorIntro` | Author hero / profile block |
| `ArticleList` | Public article list (real links) |
| `AuthorSocialLinks` | Social row |
| `ArticleAdjacentNav` | Prev / next posts |
| `getPublicArticles` | Public articles API |
| `pickLocaleValue` | Locale string picker |
| `SITE_CONFIG_GLOBAL_DEFAULT` | Default site config |

### 3.2 Type-only exports

`ThemePlugin`, `ThemePageDefinition`, `ThemeLayoutChrome`, `HeaderChromeProps`, `FooterChromeProps`, `ThemeSettingGroup`, `TokenPreset`, `ThemeTokens`, `LayoutConfig`, `HeaderConfig`, `FooterConfig`, `HeaderBrandMode`, `BrandingView`, `ThemeContractVersion`, `ThemeHostValueExport`

Themes **must not** deep-import `@/…` from the host app.

## 4. ThemePlugin surface

A theme is a `ThemePlugin` registered with the host:

| Field | Required | Notes |
|-------|----------|--------|
| `manifest` | yes | `id`, names, version, `type: "theme"` |
| `contractVersion` | yes (external) | Host contract major, e.g. `"1"` |
| `defaultTokens` | yes | `ThemeTokens` |
| `pages` | yes | At least home if the theme replaces `/` |
| `layoutChrome` | recommended | `Header` / `Footer` components |
| `settingSchema` | optional | Admin-editable theme options |
| `tokenPresets` | optional | Named color/font packs |
| `sections` / `sectionMetas` | optional | Builder blocks |

### 4.1 Registration

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
window.__INKLESS_SHARED__ = { React, ReactDOM, ReactRouterDOM, ReactI18next, host };
window.InklessThemeHost = host; // same as host
```

## 5. Ownership matrix

| Concern | Inkless (host) | Theme |
|---------|----------------|--------|
| Articles / media / comments APIs | ✓ | — |
| Article list / post body primitives | ✓ (shared UI) | styles via tokens |
| Home hero layout & density | — | ✓ |
| Author / about page (blog-first) | route merge | page component |
| Header/Footer structure | shell + settings | chrome components |
| Site name / tagline / logo URLs | site config | reads via hooks |
| Tokens (color, type, max-width) | merge/publish | defaults + presets |
| Admin install theme from URL | ✓ | ships UMD |

### 5.1 Routes

| Route | Owner |
|-------|--------|
| `/` (theme home) | Theme `pages[]` |
| `/author` (blog-first) | Theme `pages[]` |
| `/blog`, `/blog/:slug` | Host |
| `/categories/*`, `/tags/*` | Host |
| Dynamic CMS pages | Host + sections |

## 6. Tokens

Themes must use host CSS variables (applied by `ThemeProvider`), not hardcoded brand hex in components (except optional decorative monogram backgrounds that can fall back to `var(--color-on-surface)`).

Required token groups: `colors`, `fonts`, `layout` (see `frontend/src/theme/tokens.ts`).

Host merges published site theme config over theme defaults.

## 7. Package / GitHub layout (target extraction)

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

### 7.1 Local monorepo commands

```bash
pnpm -C packages/theme-blog-first build   # → dist/theme.umd.js + theme.es.js
pnpm theme:umd:smoke                      # build + Playwright register smoke
```

CI: Quality Gate runs UMD build + smoke after Playwright Chromium install.

## 8. Extraction roadmap (blog-first)

1. UX polish — done (iterative).
2. **Contract lock + export inventory + UMD CI** — done (this doc + code).
3. Move home page into theme folder — done.
4. Monorepo package `packages/theme-blog-first` — done.
5. **Separate GitHub repo** + pin in Inkless; document Release assets — next.

## 9. Non-goals

- Themes shipping their own React copy.
- Themes calling admin APIs with secrets.
- Personal site copy inside theme source (use site config).

## 10. Success criteria

- New themes installable via URL without core PR.
- blog-first default for blank sites remains.
- Style PRs land primarily in the theme package/repo.
- Host facade changes are inventory-tested and version-locked.
