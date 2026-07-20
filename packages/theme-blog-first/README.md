# @inkless/theme-blog-first

Inkless **blog-first** theme: personal-blog home, author page, header/footer chrome, and reading-room tokens.

## Contract

| Field | Value |
|-------|--------|
| Theme id | `blog-first` |
| `contractVersion` | `1` (must match host `THEME_CONTRACT_VERSION`) |
| Host facade | `@inkless/theme-host` only (no `@/` deep imports) |

See monorepo `docs/theme-contract.md` for the frozen export inventory and version lock rules.

## Consumed by host (built-in)

Inkless frontend depends on this workspace package and calls:

```ts
import { blogFirstTheme } from "@inkless/theme-blog-first";
themeManager.registerBuiltIn(blogFirstTheme);
```

Theme source imports host APIs only from `@inkless/theme-host` (Vite alias → `frontend/src/theme-host`).

## Remote install (UMD)

```bash
pnpm -C packages/theme-blog-first build
# → dist/theme.umd.js  (and theme.es.js)

# From monorepo root (requires Playwright Chromium):
pnpm theme:umd:smoke
```

Host must expose (see `frontend/src/plugins/externals.ts`):

- `window.__INKLESS_SHARED__` — React peers + `host`
- `window.InklessThemeHost` — same as `__INKLESS_SHARED__.host`
- `window.__INKLESS_THEME_REGISTER__(theme)` — registration callback during load

Then admin sets `installed_themes.externalUrl` to the UMD URL.

## Layout

```
src/
  index.ts          # ThemePlugin export (+ contractVersion)
  register.ts       # UMD auto-register entry
  chrome/           # BlogHeader / BlogFooter / brand rules
  pages/home.tsx    # Theme home
  pages/author.tsx  # Author / about
inkless.theme.json  # Package manifest for installers
```
