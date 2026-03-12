# Unified Visual Template Pages Design

**Date:** 2026-03-12
**Status:** Draft
**Scope:** Merge the block/section page system (`pages` table) and the content document system (`content_documents` table) into a single unified visual template page system.

---

## Problem Statement

The project currently has two incompatible page systems:

1. **Block/Section Pages** (`pages` table) — user-created, unlimited, section-based JSON config with visual drag-and-drop editor. Lacks versioning, draft/publish workflow, translation tracking, and rollback.

2. **Content Documents** (`content_documents` + `content_versions` tables) — 8 hardcoded pages with page-specific TypeScript schemas, form-based editor, full draft/publish dual-version workflow, version history with rollback, and per-field translation status tracking.

The two systems are bridged via `isThemePage` + `renderMode="hardcoded"` flags on the `pages` table, creating a confusing split where some page rows point to `content_documents` for their actual content.

**Problems this causes:**

- Two editing experiences for conceptually the same thing (a "page")
- No versioning or rollback for user-created pages
- No translation tracking for user-created pages
- Hardcoded page keys limit the content document system to exactly 8 pages
- Hardcoded React components and TypeScript schemas per page create maintenance burden
- Theme system cannot generate pages that participate in the content workflow

## Goals

1. One page system with one editing experience
2. All pages get draft/publish workflow, version history, rollback, and translation tracking
3. Two page modes: page-level templates (fixed structure, edit content only) and composable pages (free drag-and-drop)
4. Existing 8 fixed pages migrate to built-in page-level templates
5. Site configurations can be exported/imported as data-only themes
6. Incremental path to code-extensible themes via the plugin system (future)

## Non-Goals

- Custom component code in themes (deferred to future plugin-based phase)
- Real-time collaborative editing
- Page-level access control (handled separately by RBAC system)

---

## Design

### Two Page Modes

#### Page-Level Template (`mode: "template"`)

- Predefined page layout with a fixed set of sections in a fixed order
- Users can edit section content (text, images, links) but cannot add, remove, or reorder sections
- Suitable for structured pages like Home, About, Contact
- The 8 existing hardcoded pages become 8 built-in page-level templates
- Additional templates can be created by users and shared as part of themes

#### Composable Page (`mode: "composable"`)

- Users drag-and-drop section components from a library to build pages freely
- Sections can be added, removed, and reordered
- Each section component offers multiple layout variants (e.g., hero: fullscreen / split / solid-color)
- Per-section style settings (spacing, background, visibility)
- Suitable for landing pages, marketing pages, and any custom layout

### Unified Data Model

#### Database: `unified_pages` table

Replaces both `pages` and `content_documents` tables. Defined as GORM model (dialect-agnostic for SQLite and PostgreSQL):

```go
// UnifiedPage replaces both Page and ContentDocument.
type UnifiedPage struct {
    ID   uint   `gorm:"primaryKey" json:"id"`
    Slug string `gorm:"uniqueIndex;size:200;not null" json:"slug"`

    // Bilingual metadata
    ZhTitle       string `gorm:"size:500;not null;default:''" json:"zhTitle"`
    EnTitle       string `gorm:"size:500;not null;default:''" json:"enTitle"`
    ZhDescription string `gorm:"type:text;not null;default:''" json:"zhDescription"`
    EnDescription string `gorm:"type:text;not null;default:''" json:"enDescription"`

    // Page mode
    Mode       string `gorm:"size:20;not null;default:'composable'" json:"mode"` // "template" | "composable"
    TemplateID *uint  `gorm:"index" json:"templateId"`                           // FK to page_templates (NULL for composable)

    // Dual-version content
    DraftConfig      JSONMap `gorm:"type:text" json:"draftConfig"`
    DraftVersion     int     `gorm:"not null;default:1" json:"draftVersion"`
    PublishedConfig  JSONMap `gorm:"type:text" json:"publishedConfig"`
    PublishedVersion int     `gorm:"not null;default:0" json:"publishedVersion"`

    // Status & workflow
    Status      string     `gorm:"size:20;not null;default:'draft'" json:"status"` // "draft" | "published" | "scheduled"
    ScheduledAt *time.Time `json:"scheduledAt"`

    // Translation tracking
    TranslationStatus JSONMap `gorm:"type:text" json:"translationStatus"`

    // SEO
    ZhMetaTitle       string `gorm:"size:200;not null;default:''" json:"zhMetaTitle"`
    EnMetaTitle       string `gorm:"size:200;not null;default:''" json:"enMetaTitle"`
    ZhMetaDescription string `gorm:"size:500;not null;default:''" json:"zhMetaDescription"`
    EnMetaDescription string `gorm:"size:500;not null;default:''" json:"enMetaDescription"`
    ZhMetaKeywords    string `gorm:"size:500;not null;default:''" json:"zhMetaKeywords"`
    EnMetaKeywords    string `gorm:"size:500;not null;default:''" json:"enMetaKeywords"`

    // Navigation & ordering
    SortOrder int  `gorm:"not null;default:0" json:"sortOrder"`
    ShowInNav bool `gorm:"not null;default:false" json:"showInNav"`
    ParentID  *uint `gorm:"index" json:"parentId"`

    // Timestamps
    CreatedAt   time.Time  `gorm:"autoCreateTime" json:"createdAt"`
    UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
    PublishedAt *time.Time `json:"publishedAt"`
}
```

> **Note on SQLite/PostgreSQL compatibility:** The model uses `gorm:"type:text"` for JSON columns. GORM's JSONMap scanner handles serialization in both dialects. Raw SQL in goose migrations must provide separate up scripts per dialect or use TEXT type universally. This follows the existing pattern in `backend/internal/model/page.go` which uses `JSONMap` with `gorm:"type:jsonb"` — GORM maps this to TEXT on SQLite automatically.

#### Database: `page_versions` table

Replaces `content_versions`. Append-only history for all pages.

```go
// PageVersion stores a snapshot of a page config at publish time.
type PageVersion struct {
    ID        uint    `gorm:"primaryKey" json:"id"`
    PageID    uint    `gorm:"not null;index;uniqueIndex:idx_page_version" json:"pageId"`
    Version   int     `gorm:"not null;uniqueIndex:idx_page_version" json:"version"`
    Config    JSONMap `gorm:"type:text;not null" json:"config"`
    CreatedBy uint    `gorm:"not null" json:"createdBy"` // user who published this version
    CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}
```

> **Note:** `CreatedBy` preserves the audit trail from the existing `ContentVersion.CreatedBy` field (who created each version), rather than only tracking who published.

#### Config JSON Format

All pages (both modes) use the same config shape:

```json
{
  "sections": [
    {
      "id": "uuid-string",
      "type": "hero",
      "variant": "fullscreen",
      "locked": false,
      "data": {
        "title": { "zh": "...", "en": "..." },
        "subtitle": { "zh": "...", "en": "..." },
        "backgroundImage": "/uploads/hero.jpg",
        "cta": {
          "text": { "zh": "...", "en": "..." },
          "url": "/contact"
        }
      },
      "settings": {
        "background": "surface",
        "padding": "lg",
        "hidden": false
      }
    }
  ],
  "tokens": {
    "primaryColor": "#1a365d",
    "secondaryColor": "#2d3748",
    "accentColor": "#e53e3e",
    "fontFamily": "Inter, sans-serif",
    "borderRadius": "8px"
  }
}
```

Key differences from the current block page config:

- **`variant` field**: Each section type can have multiple layout variants
- **`locked` field**: In template mode, all sections have `locked: true` (cannot be moved/deleted by users)
- **Bilingual `data` fields**: Section data uses `{ zh, en }` objects instead of flat strings, enabling per-field translation tracking
- **`tokens` object**: Page-level style tokens that override the global theme tokens

#### Page-Level Templates

A page-level template is a named preset that defines:

```json
{
  "key": "home",
  "name": { "zh": "首页模板", "en": "Home Page Template" },
  "description": { "zh": "...", "en": "..." },
  "category": "builtin",
  "sections": [
    { "type": "hero", "variant": "fullscreen", "locked": true, "data": { ... defaults ... }, "settings": { ... } },
    { "type": "card-grid", "variant": "three-column", "locked": true, "data": { ... }, "settings": { ... } },
    { "type": "company-profile", "variant": "default", "locked": true, "data": { ... }, "settings": { ... } }
  ],
  "tokens": { ... }
}
```

**Built-in templates** (migrated from the 8 existing content document schemas):

| Template Key | Sections |
|---|---|
| `home` | hero, card-grid (advantages), company-profile, service-cards, team-grid, contact-form |
| `about` | hero, company-profile, rich-text (history), team-grid |
| `advantages` | hero, checklist |
| `core-services` | hero, service-cards, rich-text |
| `cases` | hero, card-grid |
| `experts` | hero, team-grid |
| `contact` | hero, contact-form, text-image (map) |
| `global` | (site-wide settings — not a page template, see below) |
| `theme` | (active theme config — not a page template, see below) |

**`global` and `theme` content document migration:**

The `content_documents` table has 9 `PageKey` values, not 7. Two are non-page configs:

- **`global`** — site-wide settings (site name, logo, footer, navigation)
- **`theme`** — active theme config (colors, typography, token overrides)

Neither `global` nor `theme` has an existing dedicated model in the codebase — both are stored as `content_documents` rows with JSON configs accessed via `contentDocRepo.FindByPageKey()`.

**Migration target:** Create a new **`SiteConfig`** model to replace both:

```go
// SiteConfig stores site-wide configuration (replaces global and theme content documents).
type SiteConfig struct {
    ID               uint    `gorm:"primaryKey" json:"id"`
    Key              string  `gorm:"uniqueIndex;size:50;not null" json:"key"` // "global" | "theme"
    DraftConfig      JSONMap `gorm:"type:text" json:"draftConfig"`
    DraftVersion     int     `gorm:"not null;default:1" json:"draftVersion"`
    PublishedConfig  JSONMap `gorm:"type:text" json:"publishedConfig"`
    PublishedVersion int     `gorm:"not null;default:0" json:"publishedVersion"`
    CreatedAt        time.Time `gorm:"autoCreateTime" json:"createdAt"`
    UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}
```

This preserves the draft/publish dual-version workflow. Version history for `global` and `theme` is stored in `page_versions` using sentinel page IDs (`global` → page_id=9998, `theme` → page_id=9999) to maintain rollback capability without conflicting with real page IDs.

The existing `theme/handler.go` endpoints (`GET/PUT /admin/theme/config`) and `content/get_draft.go` references to `PageKeyGlobal`/`PageKeyTheme` are updated to read from `SiteConfig` instead of `content_documents`.

Templates are stored in a `page_templates` table:

```go
// PageTemplate defines a reusable page layout preset.
type PageTemplate struct {
    ID            uint   `gorm:"primaryKey" json:"id"`
    Key           string `gorm:"uniqueIndex;size:100;not null" json:"key"`
    NameZh        string `gorm:"size:200;not null" json:"nameZh"`
    NameEn        string `gorm:"size:200;not null" json:"nameEn"`
    DescriptionZh string `gorm:"type:text;not null;default:''" json:"descriptionZh"`
    DescriptionEn string `gorm:"type:text;not null;default:''" json:"descriptionEn"`
    Category      string `gorm:"size:50;not null;default:'custom'" json:"category"` // "builtin" | "custom" | "theme"
    Config        JSONMap `gorm:"type:text;not null" json:"config"`                  // template definition (sections + tokens)
    Thumbnail     string `gorm:"size:500" json:"thumbnail"`
    CreatedAt     time.Time `gorm:"autoCreateTime" json:"createdAt"`
    UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}
```

### Section Variants

> **Implementation note on existing components:** Adding variant support requires modifying existing section components. Each component currently implements a single layout. The migration adds a `variant` prop to `SectionProps` and defaults to `"default"` so existing sections continue to work unchanged. New variants are added incrementally — each section starts with only `variant: "default"` mapping to its current behavior, and additional variants are added as needed over time. This is an additive, non-breaking change to each component.

Each section component supports multiple layout variants. The variant list is defined in the frontend component registry and exposed to the editor:

```typescript
interface SectionDefinition {
  type: string;
  name: { zh: string; en: string };
  icon: string;
  variants: SectionVariant[];
  dataSchema: FieldSchema[];        // defines editable fields per variant
  settingsSchema: FieldSchema[];    // defines style settings
}

interface SectionVariant {
  key: string;
  name: { zh: string; en: string };
  thumbnail: string;                 // preview image for the variant picker
}
```

Example for hero section:

| Variant Key | Description |
|---|---|
| `fullscreen` | Full-viewport background image with centered text overlay |
| `split` | Left text, right image (50/50 split) |
| `solid` | Solid color background with centered text |
| `video` | Background video with text overlay |

The editor shows variant thumbnails when a user clicks on a section, allowing visual selection.

### Unified Editor

The admin page editor merges the best of both current editors:

**For composable pages:**
- Left sidebar: section component library with variant previews, drag to add
- Center: live preview of the page with click-to-edit inline
- Right sidebar: selected section's content fields + style settings
- Bottom toolbar: save draft, validate, publish, version history

**For template pages:**
- Same layout, but the left sidebar shows the template's fixed sections (not draggable, not removable)
- Sections display a lock icon
- Users can only edit content and style settings, not structure

**Shared capabilities (both modes):**
- Draft/publish workflow with optimistic concurrency (`If-Match` header)
- Version history panel with rollback
- Translation status indicators per field (missing / stale / complete)
- JSON mode toggle for advanced users
- Undo/redo stack

### Theme Export/Import (Phase 1: Data-Only)

A theme is a JSON package that can be exported from a site and imported into another:

```json
{
  "name": "corporate-classic",
  "version": "1.0.0",
  "author": "...",
  "description": { "zh": "...", "en": "..." },
  "tokens": {
    "primaryColor": "#1a365d",
    "fontFamily": "Inter, sans-serif",
    ...
  },
  "pageTemplates": [
    {
      "key": "home",
      "name": { "zh": "首页", "en": "Home" },
      "sections": [ ... ]
    }
  ],
  "sectionVariants": {
    "hero": ["fullscreen", "split", "solid"],
    ...
  },
  "previewImages": [ "preview-home.png", "preview-about.png" ]
}
```

**Export flow:** Admin > Theme Settings > Export → generates a `.json` file (+ preview images as a `.zip`).

**Import flow:** Admin > Theme Settings > Import → uploads the package, creates `page_templates` rows, applies tokens.

Future phase: themes can include custom section components via the plugin system (Phase 3 plugin architecture).

### API Design

#### Page CRUD

```
GET    /admin/pages                       — list all pages (with mode filter)
POST   /admin/pages                       — create page (specify mode + optional templateId)
GET    /admin/pages/:id/draft             — get draft config + version
PUT    /admin/pages/:id/draft             — save draft (If-Match: <version>)
POST   /admin/pages/:id/validate          — validate draft
POST   /admin/pages/:id/publish           — publish draft → published
POST   /admin/pages/:id/unpublish         — unpublish
POST   /admin/pages/:id/rollback/:ver     — rollback to version
GET    /admin/pages/:id/versions          — list version history
GET    /admin/pages/:id/versions/:ver     — get specific version
DELETE /admin/pages/:id                   — delete page

GET    /public/pages                      — list published pages
GET    /public/pages/:slug                — get published page config (locale-filtered)
```

**Optimistic concurrency for `PUT /admin/pages/:id/draft`:**

The draft save endpoint uses `If-Match` header for conflict detection, following the existing `content_documents` pattern:

1. Client sends `PUT /admin/pages/:id/draft` with header `If-Match: <draft_version>`
2. Handler reads the header, parses as integer
3. Handler loads the page from DB, compares `If-Match` value with current `draft_version`
4. If mismatch → return HTTP 409 with body `{ "error": "conflict", "currentVersion": <actual_version> }`
5. If match → update `draft_config`, increment `draft_version`, save, return 200 with new version
6. Client stores the returned version for the next save

This prevents lost updates when multiple tabs or users edit the same page.

#### Template CRUD

```
GET    /admin/templates                   — list all templates
POST   /admin/templates                   — create custom template
PUT    /admin/templates/:id               — update template
DELETE /admin/templates/:id               — delete (builtin templates cannot be deleted)
POST   /admin/templates/:id/duplicate     — duplicate as new custom template
```

#### Theme Export/Import

```
POST   /admin/themes/export              — export current site as theme JSON
POST   /admin/themes/import              — import theme package
GET    /admin/themes                      — list installed themes
PUT    /admin/themes/:id/apply            — apply theme tokens to site
```

### Migration Plan

One-time migration from the two existing systems to the unified system.

#### Step 1: Create new tables

Create `unified_pages`, `page_versions`, `page_templates` via GORM AutoMigrate + goose migration for indexes.

#### Step 2: Migrate content documents → page templates + unified pages

For each of the 7 page-type `content_documents` rows (excluding `global` and `theme`, which migrate to `SiteConfig` — see above):

**2a. ID assignment:** Assign deterministic integer IDs to each content document for the FK mapping:

| PageKey | Assigned unified_pages.id |
|---|---|
| `home` | 1 |
| `about` | 2 |
| `advantages` | 3 |
| `core-services` | 4 |
| `cases` | 5 |
| `experts` | 6 |
| `contact` | 7 |

Block pages migrated in Step 3 start from ID 100+ (or use auto-increment above 7).

**2b. Config conversion — content document schema → sections array:**

Content documents use object-keyed configs (e.g., `{ hero: {...}, about: {...}, advantages: {...} }`). Each top-level key maps to a section type:

| Content key | Section type | Field mapping |
|---|---|---|
| `hero` | `hero` | `hero.title` → `data.title`, `hero.subtitle` → `data.subtitle`, `hero.backgroundImage` → `data.backgroundImage` |
| `about` / `companyProfile` | `company-profile` | `about.descriptions[]` → `data.descriptions[]`, `about.image` → `data.image` |
| `advantages` / `cards` | `card-grid` or `checklist` | `advantages.cards[].title` → `data.cards[].title`, etc. |
| `services` | `service-cards` | `services.items[]` → `data.cards[]` |
| `team` | `team-grid` | `team.members[]` → `data.members[]` |
| `contact` | `contact-form` | `contact.fields` → `data.fields` |
| `history` / `richText` | `rich-text` | `history.content` → `data.content` |
| `map` | `text-image` | `map.image` → `data.image`, `map.address` → `data.text` |

The implementation writes a Go migration function (not raw SQL) that reads each `content_documents` row, parses the JSON, and transforms it key-by-key into the sections array format. Each schema is small and known at migration time, so the mapping is exhaustive and testable.

**Bilingual fields:** Content document configs already use `{ zh: "...", en: "..." }` objects for localized text (confirmed by the `normalizeConfigForLocale()` function in the frontend). These are preserved as-is — no transformation needed.

**2c. Version history migration:**

For each `content_versions` row:
- Map `content_versions.page_key` (string) → `page_versions.page_id` (int) using the ID assignment table above
- Apply the same config transformation (object-keyed → sections array) to each version's config
- Map `content_versions.created_by` → `page_versions.created_by`
- Map `content_versions.version` → `page_versions.version`

**2d. Migrate `global` content document:**

See the `global` migration note in the Templates section above. `global` config fields map to `GlobalConfig` columns; version history is preserved separately.

#### Step 3: Migrate block pages → unified pages

For each existing `pages` row (where `isThemePage` is false):

**3a. Config format:** Block pages already use `{ sections: [{type, data, settings}] }` format. The migration:
- Copies `config` → both `draft_config` and `published_config`
- Adds `variant: "default"` to each section (new field, backward-compatible)
- Adds `locked: false` to each section
- Sets `draft_version: 1`, `published_version: 1` (if status was `published`), `published_version: 0` (if draft)

**3b. Draft/publish contract for migrated pages:**
- Pages with `status: "published"` → copy `config` to both `draft_config` and `published_config`, set `draft_version: 1`, `published_version: 1`
- Pages with `status: "draft"` → copy `config` to `draft_config` only, set `published_config: NULL`, `draft_version: 1`, `published_version: 0`
- **Public endpoint behavior:** `GET /public/pages/:slug` returns 404 if `published_config IS NULL`. This matches the existing check in `page/handler.go` which returns 404 for non-published pages.

**3c. Bilingual data fields:** Current block page section data uses flat strings (e.g., `"title": "关于我们"`). The migration wraps these in bilingual objects:
- For each string field in `data`: `"title": "关于我们"` → `"title": { "zh": "关于我们", "en": "" }`
- The source locale is assumed to be `zh` (the site's primary locale)
- Non-string fields (numbers, booleans, arrays of objects) are left unchanged
- The migration script identifies localizable fields per section type using a whitelist:

| Section type | Localizable fields (wrapped in `{zh, en}`) |
|---|---|
| `hero` | `title`, `subtitle`, `cta.text` |
| `card-grid` | `title`, `subtitle`, `cards[].title`, `cards[].description` |
| `rich-text` | `content`, `title` |
| `contact-form` | `title`, `subtitle`, `submitText`, `fields[].label`, `fields[].placeholder` |
| `service-cards` | `title`, `subtitle`, `cards[].title`, `cards[].description` |
| `team-grid` | `title`, `subtitle`, `members[].name`, `members[].role`, `members[].bio` |
| `text-image` | `title`, `text` |
| `checklist` | `title`, `items[].title`, `items[].description` |
| `company-profile` | `title`, `description`, `stats[].label` |

**3d. Down migration safety:** The down migration reverses the bilingual wrapping by extracting the `zh` value from each `{ zh, en }` object back to a flat string. Since `zh` is the primary locale and the source of all existing data, no information is lost on rollback.

#### Step 4: Update frontend

- **Update `SectionData` type** in `frontend/src/theme/types.ts` to add `variant?: string` and `locked?: boolean` fields — this is the wire-format type used by `SectionRenderer`, `DynamicPage`, and the editor
- **Update `SectionRenderer`** to read `section.variant` and dispatch to the correct variant component (falling back to default)
- **Update `SectionProps`** to include `variant?: string` prop
- Update each section component to accept `variant` and render its current layout as the `"default"` variant
- Add `useLocalizedData(data)` hook that extracts current locale from `{ zh, en }` objects
- Remove hardcoded page components (`pages/about/page.tsx`, etc.) and their TypeScript schemas
- Remove `content_documents` API client code
- Update `DynamicPage.tsx` to handle both modes via `SectionRenderer`
- Build the unified editor (replaces both current editors)

#### Step 5: Update backend

- Remove `content_document` model, repository, service, and handler
- Remove `content_versions` model and repository
- Extend page handler with draft/publish/version endpoints
- Add template handler
- Add theme export/import handler

#### Step 6: Drop old tables

- Drop `content_documents` and `content_versions` tables via goose migration
- Goose down migration restores old tables and runs reverse data migration

### Translation Tracking

The unified system tracks translation status per localizable field in a section:

```json
{
  "sections[0].data.title": {
    "status": "stale",
    "sourceLocale": "zh",
    "sourceHash": "abc123",
    "translatedAt": "2026-03-10T..."
  },
  "sections[1].data.description": {
    "status": "missing",
    "sourceLocale": "zh"
  }
}
```

Statuses: `missing` (target locale empty), `stale` (source changed since last translation), `complete` (up to date).

The editor shows colored indicators next to each field and a page-level translation completeness percentage.

### Rendering

Public-facing rendering remains through `SectionRenderer`, with these enhancements:

1. **Variant dispatch**: `SectionRenderer` reads the `variant` field and renders the appropriate variant component
2. **Locale extraction**: A `useLocalizedData(data)` hook extracts the current locale's string from `{ zh, en }` objects
3. **Token application**: Page-level `tokens` override the global theme CSS variables via a `<style>` block
4. **Template lock enforcement**: In the editor, template-mode sections render with a lock overlay and prevent structural changes

---

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Data loss during migration | Goose migration with transactions; backup before migration; down migration to restore old tables |
| Breaking existing published pages | Migration script preserves all published configs; run validation pass after migration |
| Editor complexity | Build incrementally: composable mode first (extends existing editor), template mode second |
| Section variant explosion | Start with 2-3 variants per section; add more based on user demand |
| Theme import conflicts | Import creates new templates with `category: "theme"`; does not overwrite builtin or user-created templates |
| Performance with large version history | Paginate version list API; consider pruning old versions after N entries |

## Future Extensions

- **Plugin-based themes** (Phase 2): Themes can include custom section components loaded via the plugin system
- **Real-time preview**: WebSocket-based live preview while editing
- **A/B testing**: Publish multiple page variants and split traffic
- **Section marketplace**: Share individual section components across sites
