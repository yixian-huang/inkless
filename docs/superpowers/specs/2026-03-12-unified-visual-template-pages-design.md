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
- Changing the existing section component implementations (hero, card-grid, etc.)

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

Replaces both `pages` and `content_documents` tables.

```sql
CREATE TABLE unified_pages (
    id              SERIAL PRIMARY KEY,
    slug            VARCHAR(200) NOT NULL UNIQUE,

    -- Bilingual metadata
    zh_title        VARCHAR(500) NOT NULL DEFAULT '',
    en_title        VARCHAR(500) NOT NULL DEFAULT '',
    zh_description  TEXT NOT NULL DEFAULT '',
    en_description  TEXT NOT NULL DEFAULT '',

    -- Page mode
    mode            VARCHAR(20) NOT NULL DEFAULT 'composable',  -- 'template' | 'composable'
    template_id     VARCHAR(100),                                -- references a PageTemplate key (NULL for composable)

    -- Dual-version content
    draft_config       JSONB NOT NULL DEFAULT '{}',
    draft_version      INT NOT NULL DEFAULT 1,
    published_config   JSONB,
    published_version  INT NOT NULL DEFAULT 0,

    -- Status & workflow
    status             VARCHAR(20) NOT NULL DEFAULT 'draft',  -- 'draft' | 'published' | 'scheduled'
    scheduled_at       TIMESTAMP,

    -- Translation tracking
    translation_status JSONB NOT NULL DEFAULT '{}',  -- per-field translation state

    -- SEO
    zh_meta_title       VARCHAR(200) NOT NULL DEFAULT '',
    en_meta_title       VARCHAR(200) NOT NULL DEFAULT '',
    zh_meta_description VARCHAR(500) NOT NULL DEFAULT '',
    en_meta_description VARCHAR(500) NOT NULL DEFAULT '',
    zh_meta_keywords    VARCHAR(500) NOT NULL DEFAULT '',
    en_meta_keywords    VARCHAR(500) NOT NULL DEFAULT '',

    -- Navigation & ordering
    sort_order     INT NOT NULL DEFAULT 0,
    show_in_nav    BOOLEAN NOT NULL DEFAULT false,
    parent_id      INT REFERENCES unified_pages(id),

    -- Timestamps
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at   TIMESTAMP
);
```

#### Database: `page_versions` table

Replaces `content_versions`. Append-only history for all pages.

```sql
CREATE TABLE page_versions (
    id          SERIAL PRIMARY KEY,
    page_id     INT NOT NULL REFERENCES unified_pages(id) ON DELETE CASCADE,
    version     INT NOT NULL,
    config      JSONB NOT NULL,
    published_by INT REFERENCES users(id),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(page_id, version)
);
```

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
| `global` | (site-wide settings — migrated to theme tokens, not a page template) |

The `global` content document is special — it holds site-wide config (site name, logo, footer, navigation). This migrates to a separate `site_config` table or the existing `global_configs` mechanism, not to a page template.

Templates are stored in a `page_templates` table:

```sql
CREATE TABLE page_templates (
    id          SERIAL PRIMARY KEY,
    key         VARCHAR(100) NOT NULL UNIQUE,
    name_zh     VARCHAR(200) NOT NULL,
    name_en     VARCHAR(200) NOT NULL,
    description_zh TEXT NOT NULL DEFAULT '',
    description_en TEXT NOT NULL DEFAULT '',
    category    VARCHAR(50) NOT NULL DEFAULT 'custom', -- 'builtin' | 'custom' | 'theme'
    config      JSONB NOT NULL,                         -- template definition (sections + tokens)
    thumbnail   VARCHAR(500),                           -- preview image path
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Section Variants

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

One-time migration from the two existing systems to the unified system:

**Step 1: Create new tables**
- `unified_pages` (or alter `pages` to add new columns)
- `page_versions`
- `page_templates`

**Step 2: Migrate content documents → page templates + unified pages**
- For each of the 8 `content_documents` rows:
  - Convert its TypeScript schema → a `page_templates` row with section definitions
  - Convert its `draft_config` / `published_config` → unified section-based config JSON
  - Create a `unified_pages` row with `mode: "template"` referencing the template
  - Copy version history from `content_versions` → `page_versions`

**Step 3: Migrate block pages → unified pages**
- For each existing `pages` row (where `isThemePage` is false):
  - Copy `config.sections` → `draft_config` and `published_config`
  - Add bilingual data wrappers to section data fields
  - Set `mode: "composable"`
  - Set `draft_version: 1`, `published_version: 1` (if status was published)

**Step 4: Update frontend**
- Remove hardcoded page components (`pages/about/page.tsx`, etc.) and their TypeScript schemas
- Remove `content_documents` API client code
- Update `DynamicPage.tsx` to handle both modes via `SectionRenderer`
- Build the unified editor (replaces both current editors)
- Add variant support to each section component

**Step 5: Update backend**
- Remove `content_document` model, repository, service, and handler
- Remove `content_versions` model and repository
- Extend page handler with draft/publish/version endpoints
- Add template handler
- Add theme export/import handler

**Step 6: Drop old tables**
- Drop `content_documents` and `content_versions` tables via goose migration

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
