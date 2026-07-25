# `@inkless/editor`

TipTap / CodeMirror editor kit for Inkless CMS.

## Boundaries

| Layer | Location | Depends on host? |
|-------|----------|------------------|
| Ports (types) | `src/ports` | No |
| TipTap extensions | `src/extensions` | Ports only |
| Presets / kit | `createEditorKit`, `presets` | Ports |
| React chrome | toolbar, bubble, markdown mode | Ports / TipTap |
| Host adapters | **app** `frontend/src/components/admin/editor-host` | Media API, pickers |

Inject media via `EditorPorts`:

```ts
import { createEditorKit } from "@inkless/editor";

const kit = createEditorKit("full", {
  upload: inklessUploadPort,
  picker: inklessPickerPort,
});
```

## Exports

- `@inkless/editor` — kit, chrome, ports types
- `@inkless/editor/markdown` — `markdownToHtml` / `htmlToMarkdown`
- `@inkless/editor/extensions` — custom TipTap nodes/plugins

## Not in this package

Article product shell (bilingual save, SEO, AI meta), media library modals, and upload bus stay in the app host.
