# `@inkless/theme-editorial-firm`

Inkless **editorial-firm** theme — magazine / atelier firm site layout, tokens, and four-page IA (`home`, `about`, `services`, `contact`).

- **Theme id**: `editorial-firm` (stable; do not rename without a data migration)
- **Contract**: v1 (`@inkless/theme-host`)
- **Pages** are CMS section-driven (`renderMode: "dynamic"`). Chrome and custom sections land in follow-up work.

## Host consumption

```ts
import {
  createEditorialFirmTheme,
  EDITORIAL_FIRM_THEME_ID,
} from "@inkless/theme-editorial-firm";

export const editorialFirmTheme = createEditorialFirmTheme();
```

## Token presets

| id | name | surface |
|----|------|---------|
| `ink-editorial` | Ink Editorial / 墨色编辑 | warm paper + ink |
| `noir-gallery` | Noir Gallery / 黑白画廊 | near-black + gold |

## Workspace

Monorepo package under `packages/theme-editorial-firm` (same pattern as `theme-corporate-classic`).
