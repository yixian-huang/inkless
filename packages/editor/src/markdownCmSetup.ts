import { keymap, placeholder as cmPlaceholder } from "@codemirror/view";
import { EditorView } from "@codemirror/view";
import { markdown } from "@codemirror/lang-markdown";
import { search, searchKeymap, highlightSelectionMatches } from "@codemirror/search";
import { basicSetup } from "codemirror";

function wrapSelection(
  view: EditorView,
  before: string,
  after: string,
  placeholder: string,
) {
  const { from, to } = view.state.selection.main;
  const selected = view.state.sliceDoc(from, to) || placeholder;
  const insert = before + selected + after;
  view.dispatch({
    changes: { from, to, insert },
    selection: {
      anchor: from + before.length,
      head: from + before.length + selected.length,
    },
  });
  view.focus();
}

export const MARKDOWN_PLACEHOLDER =
  "# 标题\n\n支持 **Markdown**、表格与 ```mermaid 图表```…";

export const markdownEditorTheme = EditorView.theme({
  "&": {
    height: "100%",
    fontSize: "13px",
    backgroundColor: "#fff",
  },
  "&.cm-focused": { outline: "none" },
  ".cm-scroller": {
    overflow: "auto",
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    lineHeight: "1.5",
  },
  ".cm-content": {
    padding: "12px 8px",
    caretColor: "#111827",
  },
  ".cm-gutters": {
    backgroundColor: "#f9fafb",
    color: "#9ca3af",
    borderRight: "1px solid #f3f4f6",
  },
  ".cm-activeLineGutter": { backgroundColor: "#f3f4f6" },
  ".cm-activeLine": { backgroundColor: "#f9fafb" },
});

export const markdownFormatKeymap = keymap.of([
  {
    key: "Mod-b",
    run: (view) => {
      wrapSelection(view, "**", "**", "粗体");
      return true;
    },
  },
  {
    key: "Mod-i",
    run: (view) => {
      wrapSelection(view, "*", "*", "斜体");
      return true;
    },
  },
  {
    key: "Mod-k",
    run: (view) => {
      wrapSelection(view, "[", "](url)", "链接文字");
      return true;
    },
  },
]);

/** Base extensions shared by every MarkdownMode instance. */
export function createMarkdownBaseExtensions() {
  return [
    basicSetup,
    search({ top: true }),
    highlightSelectionMatches(),
    keymap.of(searchKeymap),
    markdown(),
    cmPlaceholder(MARKDOWN_PLACEHOLDER),
    EditorView.lineWrapping,
    markdownFormatKeymap,
    markdownEditorTheme,
  ];
}

export const PREVIEW_TYPOGRAPHY_CLASS =
  "markdown-preview article-typography max-w-none text-sm leading-relaxed " +
  "[&_h1]:text-2xl [&_h1]:font-bold [&_h1]:mb-3 " +
  "[&_h2]:text-xl [&_h2]:font-semibold [&_h2]:mb-2 [&_h2]:mt-4 " +
  "[&_h3]:text-lg [&_h3]:font-semibold [&_h3]:mb-2 [&_h3]:mt-3 " +
  "[&_p]:mb-3 [&_p]:leading-relaxed " +
  "[&_ul]:list-disc [&_ul]:pl-6 [&_ul]:mb-3 " +
  "[&_ol]:list-decimal [&_ol]:pl-6 [&_ol]:mb-3 " +
  "[&_blockquote]:border-l-4 [&_blockquote]:border-gray-300 [&_blockquote]:pl-3 [&_blockquote]:text-gray-600 [&_blockquote]:italic " +
  "[&_a]:text-blue-600 [&_a]:underline " +
  "[&_code]:bg-gray-100 [&_code]:px-1 [&_code]:rounded [&_code]:text-[0.9em] " +
  "[&_pre]:bg-gray-900 [&_pre]:text-gray-100 [&_pre]:p-3 [&_pre]:rounded-lg [&_pre]:overflow-x-auto [&_pre]:mb-3 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  "[&_img]:max-w-full [&_img]:rounded " +
  "[&_table]:w-full [&_table]:border-collapse [&_th]:border [&_td]:border [&_th]:px-2 [&_td]:px-2 [&_th]:bg-gray-50 " +
  "[&_.mermaid]:my-4 [&_.mermaid]:flex [&_.mermaid]:justify-center [&_.mermaid]:overflow-x-auto";

/** Sync scroll ratio between two overflow containers; returns release cleanup. */
export function syncScrollRatio(
  from: HTMLElement,
  to: HTMLElement,
  lock: { current: "editor" | "preview" | null },
  source: "editor" | "preview",
) {
  if (lock.current && lock.current !== source) return;
  const fromMax = from.scrollHeight - from.clientHeight;
  const toMax = to.scrollHeight - to.clientHeight;
  if (fromMax <= 0 || toMax <= 0) return;
  lock.current = source;
  to.scrollTop = (from.scrollTop / fromMax) * toMax;
  requestAnimationFrame(() => {
    lock.current = null;
  });
}
