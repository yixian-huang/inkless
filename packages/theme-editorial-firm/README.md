# `@inkless/theme-editorial-firm`

Inkless **editorial-firm** theme — magazine / atelier firm site layout, tokens, chrome, and four-page IA (`home`, `about`, `services`, `contact`).

- **Theme id**: `editorial-firm` (stable; do not rename without a data migration)
- **Contract**: v1 (`@inkless/theme-host`)
- **Pages** are CMS section-driven (`renderMode: "dynamic"`)
- **Does not replace `corporate-classic`**: separate built-in theme id, package, and seeds. Existing classic sites and `DEFAULT_FALLBACK_THEME_ID` stay on `corporate-classic`.

## Host consumption

```ts
import {
  createEditorialFirmTheme,
  EDITORIAL_FIRM_THEME_ID,
} from "@inkless/theme-editorial-firm";

export const editorialFirmTheme = createEditorialFirmTheme();
```

Register via the package `./register` entry or the host built-in plugin path (`frontend/src/plugins/themes/editorial-firm`).

## Token presets

| id | name | surface |
|----|------|---------|
| `ink-editorial` | Ink Editorial / 墨色编辑 | warm paper + ink (default) |
| `noir-gallery` | Noir Gallery / 黑白画廊 | near-black + gold |

Semantic tokens use design-system roles (`surface`, `on-surface`, `primary`, `on-primary`, `accent`, `border`, …). CTA band uses `bg-primary` + `text-on-primary` for contrast.

## Page IA

| contentKey | slug | nav (en / zh) |
|------------|------|----------------|
| `home` | `home` | Home / 首页 |
| `about` | `about` | About / 关于 |
| `services` | `services` | Services / 服务 |
| `contact` | `contact` | Contact / 联系 |

All four use `renderMode: "dynamic"` — content is section JSON on the UnifiedPage, not hardcoded React page loaders.

## Section library (`ef-*`)

| type | label | role |
|------|-------|------|
| `ef-hero-editorial` | Editorial Hero | Full or split hero with kicker/title/deck/CTA |
| `ef-pull-quote` | Pull Quote | Large quote + attribution |
| `ef-feature-split` | Feature Split | Image + body, left/right |
| `ef-service-index` | Service Index | Service cards list |
| `ef-mosaic` | Image Mosaic | Tile grid with optional labels/links |
| `ef-cta-band` | CTA Band | Full-width conversion band (`on-primary` text) |
| `ef-contact-split` | Contact Split | Contact details + form → `POST /public/form-submissions` |
| `ef-rich-text` | Rich Text | Long-form prose block |

Schemas for the page builder live in `src/sections/schemas.ts`. Metas (`sectionMetas`) drive the admin “add block” picker.

## Chrome

- **Header** (`EditorialHeader`): sticky paper surface, uppercase wordmark or logo, host `BaseSiteHeader` nav + locale toggle. Nav is a landmark (`aria-label`); mobile menu button has `aria-expanded`.
- **Footer** (`EditorialFooter`): surface-alt, tagline, footer nav landmark, copyright / ICP, product mark.

## Seeds & activation

Default bilingual placeholder sections: `src/seed/pageConfigs.ts` (also embedded on the backend as `EditorialFirmSeedsJSON`).

On theme **activate**, the backend:

1. Seeds theme page rows (`SeedThemePages`)
2. For `editorial-firm` only: applies unified-page section seeds when a page is missing **or** published sections are empty (never overwrites existing sections)

**Customize seed content**

- **Operators / authors**: edit section props in Admin (page builder) after activate — preferred for live sites.
- **Package defaults**: edit `src/seed/pageConfigs.ts`, keep the backend embed in sync (`EditorialFirmSeedsJSON`), then re-activate only on sites with empty sections (seeds never overwrite non-empty published configs).

Operators who switch from another theme with non-empty shared slugs must clear sections or recreate pages to pick up editorial copy.

## Authoring tips

1. Prefer CMS props already used in seeds (`kicker`, `title`, `deck`, bilingual `{ zh, en }` objects resolved by the host).
2. Images: provide meaningful `caption` / tile `label` so sections can set non-empty `alt`; decorative full-bleed heroes use `alt=""`.
3. Contact form posts `formType: "contact"` with `locale` (`zh` | `en`); on API failure the UI offers mailto using the section email prop.
4. Empty pages (no visible sections) show a muted empty state via host `DynamicPage` (`status.pageEmpty`).
5. Do **not** change `EDITORIAL_FIRM_THEME_ID` or the host fallback theme without a coordinated data migration.

## Workspace

Monorepo package under `packages/theme-editorial-firm` (same pattern as `theme-corporate-classic`).

```bash
# from repo root
pnpm -C packages/theme-editorial-firm test
pnpm -C frontend type-check
```
