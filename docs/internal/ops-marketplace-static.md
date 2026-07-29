# Official marketplace static assets (A15)

## Public URLs (after frontend deploy)

| Asset | URL |
|-------|-----|
| Catalog index | `https://inkless.run/marketplace/v1/themes.json` |
| product-first UMD | `https://inkless.run/marketplace/v1/themes/product-first/0.1.5/theme.umd.js` |
| blog-first UMD | `https://inkless.run/marketplace/v1/themes/blog-first/1.0.0/theme.umd.js` |
| editorial-firm UMD | `https://inkless.run/marketplace/v1/themes/editorial-firm/0.1.0/theme.umd.js` |

Source files live in the monorepo:

```
frontend/public/marketplace/v1/
```

They ship with the SPA `FRONTEND_DIR` (shared tree on gomami: `/opt/inkless/frontend/current`).

## GitHub Release truth

| Theme | Release |
|-------|---------|
| product-first | https://github.com/yixian-huang/inkless-theme-product-first/releases/tag/v0.1.5 |
| blog-first | https://github.com/yixian-huang/inkless-theme-blog-first/releases/tag/v1.0.0 |
| editorial-firm | https://github.com/yixian-huang/inkless-theme-editorial-firm/releases/tag/v0.1.0 |

## Product instance env

`/opt/inkless-ops/backend/.env`:

```bash
INKLESS_THEME_CATALOG_URL=https://inkless.run/marketplace/v1/themes.json
```

Personal `/opt/inkless` may omit this (embedded fallback still works).

## Isolation reminder

- Product: `inkless-ops` :8089 → inkless.run
- Personal: `inkless` :8088
- Code symlink shared; **data and .env must stay separate**
