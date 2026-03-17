# Dynamic Form Property Editor for Section Editing

**Date:** 2026-03-16
**Status:** Approved

## Problem

The admin page editor's right-side panel currently uses a raw JSON textarea for editing section `data` and `settings`. This is error-prone and unfriendly for non-technical users.

## Solution

Replace the JSON textarea with a declarative, schema-driven dynamic form system. Each section type defines a `schema` in the existing section registry (`frontend/src/theme/sections/index.ts`), and a generic `DynamicForm` component renders the appropriate form controls based on that schema.

## Decisions

| Decision | Choice | Alternatives Considered |
|----------|--------|------------------------|
| Approach | Declarative schema + generic renderer | Per-section custom editors; Hybrid mode |
| Coverage | All 9 section types + settings | Subset first |
| Array editing | Inline expand in panel | Modal editing; Mixed |
| Bilingual fields | zh/en tab switching per field | Side-by-side columns; Global language toggle |
| Media fields | URL input + media library picker | URL-only; Upload + media library |
| Settings | Included as collapsible section | Deferred |
| Schema location | Section registry (alongside label/icon) | Separate files; Backend API |

## Schema Type System

### FieldSchema Interface

```ts
interface FieldSchema {
  key: string;            // Key in the data object
  type: FieldType;        // One of the types below
  label: string;          // Display label (Chinese)
  placeholder?: string;   // Placeholder text
  defaultValue?: unknown; // Default value for new items
  hidden?: boolean;       // If true, field is auto-managed (not shown in form)
  options?: { label: string; value: string | number }[];  // For select type
  itemSchema?: FieldSchema[];  // For array type: child field definitions
  // For "string-array" type: items are raw strings, not objects with sub-fields
}

type FieldType =
  | "text"
  | "textarea"
  | "bilingual"
  | "bilingual-textarea"
  | "media"
  | "color"
  | "select"
  | "number"
  | "boolean"
  | "array"
  | "string-array";  // Plain string[] (e.g., checklist items)
```

### Field Type Reference

| type | Control | Use Case |
|------|---------|----------|
| `text` | Single-line input | Short text (titles, labels) |
| `textarea` | Multi-line input | Descriptions, rich text content |
| `bilingual` | zh/en tab-switching single-line input | All bilingual short text |
| `bilingual-textarea` | zh/en tab-switching multi-line input | Bilingual long text |
| `media` | URL input + media library button + thumbnail preview | Image fields |
| `color` | Text input + color swatch preview | backgroundColor, accentColor |
| `select` | Dropdown | Enum values (imagePosition, alignment, columns) |
| `number` | Number input | Numeric values |
| `boolean` | Toggle switch | Boolean flags |
| `array` | Add/delete/drag-reorder list with inline sub-forms | cards[], services[], experts[] |
| `string-array` | Add/delete/reorder list of plain string inputs | checklist items (string[]) |

## Component Architecture

```
PropertiesPanel (right sidebar)
├── Section header (icon + type name)
├── DynamicForm
│   └── FieldRenderer (dispatches by type)
│       ├── TextField
│       ├── TextareaField
│       ├── BilingualField (internal zh/en tabs)
│       ├── BilingualTextareaField
│       ├── MediaField (URL input + media library + thumbnail)
│       ├── ColorField (text input + color swatch)
│       ├── SelectField
│       ├── NumberField
│       └── BooleanField
│       └── ArrayField
│           ├── Item list (drag-sortable cards)
│           │   └── Each item: group of FieldRenderers (recursive)
│           ├── Add button
│           └── Delete button per item
├── SectionSettings (collapsible: background/padding/maxWidth/hidden)
└── "Switch to JSON editing" link (fallback)
```

### File Locations

```
frontend/src/pages/admin/pages/editor/
├── components/
│   ├── DynamicForm.tsx          // Main form component
│   ├── FieldRenderer.tsx        // Type dispatcher
│   ├── SectionSettings.tsx      // Settings collapsible area
│   ├── fields/
│   │   ├── TextField.tsx
│   │   ├── TextareaField.tsx
│   │   ├── BilingualField.tsx
│   │   ├── BilingualTextareaField.tsx
│   │   ├── MediaField.tsx
│   │   ├── ColorField.tsx
│   │   ├── SelectField.tsx
│   │   ├── NumberField.tsx
│   │   ├── BooleanField.tsx
│   │   └── ArrayField.tsx
```

### Data Flow

1. User selects a section → look up `schema` from section registry
2. `DynamicForm` receives `schema` + current `data` object + `onChange` callback
3. Each field control produces immutable updates to `data`, propagates upward
4. `PropertiesPanel` receives new data, updates section's `data`, triggers preview refresh
5. Fully compatible with existing save/version mechanism — stored JSON structure unchanged

### Fallback

If a section has no `schema` defined (e.g., third-party plugin sections), the panel falls back to the current JSON editor. This ensures backward compatibility.

## Section Schemas (All 9 Types)

### 1. hero

| key | type | label |
|-----|------|-------|
| title | bilingual | 标题 |
| subtitle | bilingual | 副标题 |
| label | bilingual | 标签文字 |
| backgroundImage | media | 背景图 |
| backgroundColor | color | 背景色 |

### 2. text-image

| key | type | label | options |
|-----|------|-------|---------|
| title | bilingual | 标题 | |
| description | bilingual-textarea | 描述 | |
| image | media | 图片 | |
| imagePosition | select | 图片位置 | left, right |

### 3. card-grid

| key | type | label | options |
|-----|------|-------|---------|
| title | bilingual | 标题 | |
| columns | select | 列数 | 2, 3, 4 |
| cards | array | 卡片 | itemSchema below |

cards itemSchema:

| key | type | label |
|-----|------|-------|
| title | bilingual | 标题 |
| titleEn | text | 英文标题 (legacy) |
| description | bilingual-textarea | 描述 |
| image | media | 图片 |

> Note: `titleEn` is a legacy field used by CardGridSection when locale is zh to show an English subtitle. Retained for backward compatibility with existing data. New content should use the bilingual `title` field instead.

### 4. service-cards

| key | type | label |
|-----|------|-------|
| title | bilingual | 标题 |
| services | array | 服务项 |

services itemSchema:

| key | type | label |
|-----|------|-------|
| title | bilingual | 标题 |
| description | bilingual-textarea | 描述 |
| image | media | 图片 |
| link | text | 链接 |

### 5. team-grid

| key | type | label |
|-----|------|-------|
| sectionTitle | bilingual | 区块标题 |
| experts | array | 专家列表 |

experts itemSchema:

| key | type | label | notes |
|-----|------|-------|-------|
| id | text | ID | hidden: true; auto-generated via crypto.randomUUID() on item creation; preserved on edit |
| name | bilingual | 姓名 | |
| title | bilingual | 职位 | |
| image | media | 头像 | |
| bio | bilingual-textarea | 简介 | |

### 6. checklist

| key | type | label |
|-----|------|-------|
| categories | array | 分类列表 |

categories itemSchema:

| key | type | label |
|-----|------|-------|
| title | bilingual | 分类标题 |
| items | string-array | 检查项 |

> Note: `items` is a plain `string[]` in the ChecklistSection component. The `string-array` type renders as a list of simple text inputs with add/delete/reorder, storing raw strings (not objects).

### 7. contact-form

| key | type | label |
|-----|------|-------|
| title | bilingual | 标题 |
| subtitle | bilingual | 副标题 |
| nameLabel | bilingual | 姓名标签 |
| namePlaceholder | bilingual | 姓名占位符 |
| emailLabel | bilingual | 邮箱标签 |
| emailPlaceholder | bilingual | 邮箱占位符 |
| messageLabel | bilingual | 留言标签 |
| messagePlaceholder | bilingual | 留言占位符 |
| submit | bilingual | 提交按钮文字 |
| phone | bilingual | 电话 |
| address | bilingual | 地址 |
| accentColor | color | 主题色 |

### 8. company-profile

| key | type | label |
|-----|------|-------|
| title | bilingual | 标题 |
| description | bilingual-textarea | 描述 |

### 9. rich-text

| key | type | label | options |
|-----|------|-------|---------|
| content | bilingual-textarea | 内容 | |
| alignment | select | 对齐方式 | left, center |

### Shared Settings Schema

| key | type | label | options |
|-----|------|-------|---------|
| background | select | 背景 | surface, surface-alt, primary |
| padding | select | 内边距 | none, sm, md, lg |
| maxWidth | select | 最大宽度 | layout, full |
| hidden | boolean | 隐藏 | |

## Interaction Details

### ArrayField

- Each array item renders as a bordered card
- Title row: index number + first text field value as summary (e.g., "卡片 1 - 企业咨询")
- Top-right: move up / move down / delete buttons
- Card body: expanded sub-form fields
- Bottom of list: "+ 添加" button, inserts item with defaultValues
- Drag-to-reorder: extract shared `useDragSort` hook from existing `makeDragHandlers` in page.tsx
- Supports recursive nesting (checklist's categories → items)

### BilingualField

- `zh | en` tabs displayed to the right of the field label
- Default to zh tab
- Switching tabs swaps input content; each language stored independently
- Legacy compatibility: if current value is a plain string, display in zh tab, en tab empty

### MediaField

- Input shows current URL
- "选择" button to the right opens media library modal
- If value exists, thumbnail preview below input (~80x80)
- Selecting from media library auto-fills the URL

### Fallback Toggle

- Bottom of panel: "切换到 JSON 编辑" link
- Toggles between DynamicForm and raw JSON textarea
- Allows advanced users / debugging escape hatch

## Implementation Notes

### SelectField Numeric Coercion

HTML `<select>` always returns string values. `SelectField` must coerce the value back to `number` when the option's `value` is a number (e.g., columns: 2/3/4). Check `typeof option.value === "number"` and apply `Number()` on change.

### Bilingual Field Serialization

- Bilingual fields always store `{ zh: string, en: string }` objects.
- Empty `en` values are stored as empty strings (`""`), not omitted, to ensure stable JSON round-trips.
- Legacy compatibility: if the existing value is a plain string, the zh tab shows it and en tab is empty. On first edit, the value is migrated to `{ zh: originalString, en: "" }`.

### Variant Field

`SectionData.variant` editing is deferred from this spec. Variant remains editable only via JSON fallback mode. A future enhancement can add a variant selector to the panel header if sections define available variants in their metadata.

### ArrayField Drag Logic

The existing section list drag logic in `page.tsx` (`makeDragHandlers`) is inline and not reusable. Implementation should extract a shared `useDragSort` hook from the existing logic, then use it in both the section list and `ArrayField`.

### Theme Plugin Sections

Theme-contributed sections (e.g., corporate-classic's `stats-counter`) that do not have a schema will fall back to JSON editing. This is intentional for v1. Future enhancement: extend `ThemePlugin` with an optional `sectionSchemas?: Record<string, FieldSchema[]>` field and merge in `useSectionRegistry`.

## Scope Exclusions

- No field-level real-time validation (keep simple; saving is not blocked)
- No drag-to-upload on media fields (use existing media library flow)
- No undo/redo (existing version system covers rollback needs)
- No backend changes (schema is purely frontend; stored JSON structure unchanged)
