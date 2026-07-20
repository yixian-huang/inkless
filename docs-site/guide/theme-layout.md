# Theme layout & chrome

Inkless follows the same split as Ghost and headless CMS products:

| Layer | Responsibility |
|-------|----------------|
| **Site Config + Menus** | Data: site name, logo URL, author, navigation links |
| **Theme plugin** | Presentation: Header/Footer components, `defaultLayout`, theme settings, home page mapping |
| **SiteLayout** | Shell: reads active theme and renders chrome around page content |

**Active theme is the single source of truth** for site presentation (corporate vs blog-first layout, chrome, and `/` content). Features only gates public routes and blog toggles (RSS, comments)—not site mode.

## Active theme drives chrome

Each built-in theme registers layout chrome in its plugin definition:

- [`corporate-classic`](../../frontend/src/plugins/themes/corporate-classic/index.ts) — `CorporateHeader` / `CorporateFooter`, wide layout
- [`blog-first`](https://github.com/yixian-huang/inkless-theme-blog-first) — `BlogHeader` / `BlogFooter`, narrow reading layout (`@inkless/theme-blog-first`)
- [`product-first`](https://github.com/yixian-huang/inkless-theme-product-first) — product landing chrome, `maxWidth: 72rem` (`@inkless/theme-product-first`)
- [`minimal-starter`](../../frontend/src/plugins/themes/minimal-starter/index.ts) — reference theme for third-party authors (dynamic home, shared `BaseSiteHeader`)

`frontend/src/theme/layouts/PublicLayout.tsx` (also exported as `PublicLayout`) resolves:

```text
activeTheme.layoutChrome.Header/Footer
activeTheme.defaultLayout  (header/footer style, layout type)
```

Pages should not hardcode header/footer config unless overriding the theme default.

## Shared chrome utilities

Under `frontend/src/theme/layouts/chrome/`:

- `BaseSiteHeader` — shared shell (nav, mobile menu, language toggle); corporate/blog themes compose brand + utilities
- `useSiteNavigation()` — Menus → theme pages → legacy global nav, with Features gating
- `useBranding()` — logo, site name, author from Site Config
- `useHeaderSettings()` — merges theme `settingSchema` defaults with Site Config **Header** tab
- `BrandMark` — text / logo / avatar / none brand area
- `HeaderUtilities` — RSS + social links when theme/schema enables them

## Configuring blog-first header

1. **Theme settings** (Admin → Theme → Settings): default brand mode, RSS, socials
2. **Site Config → Header**: override brand mode and utility toggles
3. **Site Config → Author / Brand**: name, avatar, logo URL
4. **Menus**: primary navigation links
5. **Features**: enable `/blog`, RSS feed

Theme defaults apply when Site Config Header fields are empty.

## Adding a new theme

Copy [`minimal-starter`](../../frontend/src/plugins/themes/minimal-starter/index.ts) as a starting point—it only needs `chrome/` + `index.ts` + registration in `ThemeManagerContext`.

1. Create `plugins/themes/my-theme/chrome/MyHeader.tsx` and `MyFooter.tsx` (reuse `BaseSiteHeader` when possible)
2. Add page metadata to [`backend/internal/builtinthemes/pages.json`](../../backend/internal/builtinthemes/pages.json) for backend seed on activation
3. Register in theme plugin:

```ts
layoutChrome: { Header: MyHeader, Footer: MyFooter },
defaultLayout: { type: "default", header: { style: "sticky" }, footer: { style: "minimal" } },
```

4. Optionally add `settingSchema` for theme-specific toggles (Ghost-style custom settings)

5. Register built-in: `themeManager.registerBuiltIn(myTheme)` in `ThemeManagerContext.tsx`

No changes to `SiteLayout` are required if chrome components accept `HeaderChromeProps` / `FooterChromeProps`. Activating a theme in Admin → Theme runs backend `SeedThemePages` from `pages.json`.
