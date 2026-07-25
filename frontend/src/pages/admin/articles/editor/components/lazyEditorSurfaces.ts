import { lazy } from "react";

/** CodeMirror Markdown editor chunk */
export const LazyMarkdownMode = lazy(() =>
  import("@/components/admin/editor/MarkdownMode").then((m) => ({ default: m.default })),
);

/** TipTap canvas + bubble menus chunk */
export const LazyRichTextSurface = lazy(() =>
  import("./RichTextSurface").then((m) => ({ default: m.RichTextSurface })),
);

/** TipTap formatting toolbar */
export const LazyEditorToolbar = lazy(() =>
  import("@/components/admin/RichTextEditor").then((m) => ({ default: m.EditorToolbar })),
);

/** Markdown formatting toolbar (no CodeMirror) */
export const LazyMarkdownToolbar = lazy(() =>
  import("@/components/admin/editor/MarkdownToolbar").then((m) => ({ default: m.default })),
);

/** TipTap media/embed modals (inkless host adapter) */
export const LazyEditorModals = lazy(() =>
  import("@/components/admin/editor-host/InklessEditorModals").then((m) => ({
    default: m.InklessEditorModals,
  })),
);

/** Prefetch helpers (e.g. on mode-switcher hover). */
export function prefetchMarkdownEditor() {
  void import("@/components/admin/editor/MarkdownMode");
  void import("@/components/admin/editor/MarkdownToolbar");
  void import("@/components/admin/editor/markdownCmSetup");
}

export function prefetchRichTextEditor() {
  void import("./LangEditorMountInner");
  void import("./RichTextSurface");
  void import("@/components/admin/RichTextEditor");
}
