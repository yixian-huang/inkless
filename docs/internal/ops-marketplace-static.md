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

## Optional theme auto-update

After deploy of a binary that includes `ThemeAutoUpdateService`:

| API | Purpose |
|-----|---------|
| `GET /admin/extensions/themes/auto-update` | Settings + last report |
| `PUT /admin/extensions/themes/auto-update` | Enable / interval / scope |
| `POST /admin/extensions/themes/auto-update/run` | `{ "dryRun": true\|false }` manual check or apply |

Admin UI: **主题市场** → 顶部「可选自动更新」面板。

- Default **off**. When on, host polls catalog on an interval (min 15m) and upgrades installed marketplace UMD pointers without redeploying the site process for theme package bumps.
- Set `INKLESS_THEME_CATALOG_URL` so polls hit the live index (not only embedded fallback).
- Major theme redesigns: keep auto-update off or use dry-run; apply major bumps manually from the market cards.

## Isolation reminder

- Product: `inkless-ops` :8089 → inkless.run
- Personal: `inkless` :8088
- Code symlink shared; **data and .env must stay separate**

