import { useEffect, useMemo, useRef, useState, useCallback } from "react";
import { EditorView, basicSetup } from "codemirror";
import { EditorState } from "@codemirror/state";
import { keymap, placeholder as cmPlaceholder } from "@codemirror/view";
import { markdown } from "@codemirror/lang-markdown";
import { markdownToHtml } from "@/lib/markdown";
import MarkdownHtmlPreview from "./MermaidPreview";
import type { MarkdownSelectionApi } from "./MarkdownToolbar";

interface MarkdownModeProps {
  value: string;
  onChange: (value: string) => void;
  onImageUpload?: (file: File) => Promise<string>;
  /** Expose selection/wrap API for the external Markdown toolbar. */
  onApiReady?: (api: MarkdownSelectionApi | null) => void;
  /** Optional key identity (e.g. active locale) — forces debounce reset when it changes. */
  contentKey?: string;
  /** Show live preview pane (default true). Set false in bilingual split layouts. */
  showPreview?: boolean;
  /** Compact chrome label override */
  label?: string;
}

const PREVIEW_CLASS =
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

function wrapSelection(view: EditorView, before: string, after: string, placeholder: string) {
  const { from, to } = view.state.selection.main;
  const selected = view.state.sliceDoc(from, to) || placeholder;
  const insert = before + selected + after;
  view.dispatch({
    changes: { from, to, insert },
    selection: { anchor: from + before.length, head: from + before.length + selected.length },
  });
  view.focus();
}

export default function MarkdownMode({
  value,
  onChange,
  onImageUpload,
  onApiReady,
  contentKey,
  showPreview = true,
  label = "Markdown",
}: MarkdownModeProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const previewScrollRef = useRef<HTMLDivElement>(null);
  const syncingRef = useRef<"editor" | "preview" | null>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onImageUploadRef = useRef(onImageUpload);
  onImageUploadRef.current = onImageUpload;
  const valueRef = useRef(value);
  valueRef.current = value;
  /** Skip echoing CM updates that we ourselves dispatched from props. */
  const applyingExternalRef = useRef(false);

  const [debounced, setDebounced] = useState(value);
  const [cursorLine, setCursorLine] = useState(1);

  useEffect(() => {
    setDebounced(value);
  }, [contentKey]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const t = window.setTimeout(() => setDebounced(value), 150);
    return () => window.clearTimeout(t);
  }, [value]);

  const previewHtml = useMemo(() => markdownToHtml(debounced), [debounced]);

  // Mount CodeMirror once
  useEffect(() => {
    if (!hostRef.current) return;

    const syncFromEditor = (view: EditorView) => {
      if (applyingExternalRef.current) return;
      const next = view.state.doc.toString();
      valueRef.current = next;
      onChangeRef.current(next);
    };

    const state = EditorState.create({
      doc: valueRef.current,
      extensions: [
        basicSetup,
        markdown(),
        cmPlaceholder("# 标题\n\n支持 **Markdown**、表格与 ```mermaid 图表```…"),
        EditorView.lineWrapping,
        keymap.of([
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
        ]),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            syncFromEditor(update.view);
          }
          if (update.selectionSet || update.docChanged) {
            const line = update.state.doc.lineAt(update.state.selection.main.head).number;
            setCursorLine(line);
          }
          if (update.geometryChanged || update.docChanged) {
            // scroll sync handled via dom scroll listener
          }
        }),
        EditorView.domEventHandlers({
          drop: (event, view) => {
            const upload = onImageUploadRef.current;
            if (!upload || !event.dataTransfer) return false;
            const files = Array.from(event.dataTransfer.files).filter((f) =>
              f.type.startsWith("image/"),
            );
            if (files.length === 0) return false;
            event.preventDefault();
            void (async () => {
              let insert = "";
              for (const file of files) {
                const url = await upload(file);
                insert += `\n![${file.name}](${url})\n`;
              }
              const pos = view.state.selection.main.head;
              view.dispatch({
                changes: { from: pos, insert },
                selection: { anchor: pos + insert.length },
              });
            })();
            return true;
          },
          paste: (event, view) => {
            const upload = onImageUploadRef.current;
            if (!upload || !event.clipboardData) return false;
            const items = Array.from(event.clipboardData.items);
            const images = items.filter((i) => i.type.startsWith("image/"));
            if (images.length === 0) return false;
            event.preventDefault();
            void (async () => {
              for (const item of images) {
                const file = item.getAsFile();
                if (!file) continue;
                const url = await upload(file);
                const md = `![image](${url})`;
                const pos = view.state.selection.main.head;
                view.dispatch({
                  changes: { from: pos, insert: md },
                  selection: { anchor: pos + md.length },
                });
              }
            })();
            return true;
          },
          scroll: () => {
            // bubbled from scroller — handled below via scroller listener
            return false;
          },
        }),
        EditorView.theme({
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
        }),
      ],
    });

    const view = new EditorView({
      state,
      parent: hostRef.current,
    });
    viewRef.current = view;

    const scroller = view.scrollDOM;
    const onEditorScroll = () => {
      if (syncingRef.current === "preview") return;
      const previewEl = previewScrollRef.current;
      if (!previewEl) return;
      const fromMax = scroller.scrollHeight - scroller.clientHeight;
      const toMax = previewEl.scrollHeight - previewEl.clientHeight;
      if (fromMax <= 0 || toMax <= 0) return;
      syncingRef.current = "editor";
      previewEl.scrollTop = (scroller.scrollTop / fromMax) * toMax;
      requestAnimationFrame(() => {
        syncingRef.current = null;
      });
    };
    scroller.addEventListener("scroll", onEditorScroll, { passive: true });

    return () => {
      scroller.removeEventListener("scroll", onEditorScroll);
      view.destroy();
      viewRef.current = null;
    };
    // Mount once; external value sync is handled below.
  }, []);

  // Sync external value → CodeMirror (locale switch / restore / mode change)
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const current = view.state.doc.toString();
    if (current === value) return;
    applyingExternalRef.current = true;
    view.dispatch({
      changes: { from: 0, to: current.length, insert: value },
    });
    applyingExternalRef.current = false;
    valueRef.current = value;
  }, [value, contentKey]);

  // Selection API for MarkdownToolbar
  useEffect(() => {
    if (!onApiReady) return;
    const api: MarkdownSelectionApi = {
      getValue: () => viewRef.current?.state.doc.toString() ?? valueRef.current,
      setValue: (next, cursor) => {
        const view = viewRef.current;
        if (!view) {
          onChangeRef.current(next);
          return;
        }
        const cur = view.state.doc.toString();
        view.dispatch({
          changes: { from: 0, to: cur.length, insert: next },
          selection: cursor
            ? { anchor: cursor.start, head: cursor.end }
            : { anchor: next.length },
        });
        view.focus();
      },
      getSelection: () => {
        const view = viewRef.current;
        if (!view) return { start: 0, end: 0 };
        const { from, to } = view.state.selection.main;
        return { start: from, end: to };
      },
      focus: () => viewRef.current?.focus(),
    };
    onApiReady(api);
    return () => onApiReady(null);
  }, [onApiReady]);

  // Cursor line → preview scroll (ratio by line / total lines)
  useEffect(() => {
    if (!showPreview) return;
    const previewEl = previewScrollRef.current;
    const view = viewRef.current;
    if (!previewEl || !view) return;
    const total = Math.max(1, view.state.doc.lines);
    const ratio = (cursorLine - 1) / total;
    const toMax = previewEl.scrollHeight - previewEl.clientHeight;
    if (toMax <= 0) return;
    if (syncingRef.current === "editor") return;
    syncingRef.current = "editor";
    previewEl.scrollTop = ratio * toMax;
    requestAnimationFrame(() => {
      syncingRef.current = null;
    });
  }, [cursorLine, showPreview, debounced]);

  const handlePreviewScroll = useCallback(() => {
    if (syncingRef.current === "editor") return;
    const view = viewRef.current;
    const previewEl = previewScrollRef.current;
    if (!view || !previewEl) return;
    const scroller = view.scrollDOM;
    const fromMax = previewEl.scrollHeight - previewEl.clientHeight;
    const toMax = scroller.scrollHeight - scroller.clientHeight;
    if (fromMax <= 0 || toMax <= 0) return;
    syncingRef.current = "preview";
    scroller.scrollTop = (previewEl.scrollTop / fromMax) * toMax;
    requestAnimationFrame(() => {
      syncingRef.current = null;
    });
  }, []);

  return (
    <div className="flex h-full min-h-0 gap-0 border border-gray-200 rounded-lg overflow-hidden bg-white">
      <div
        className={`min-w-0 min-h-0 flex flex-col ${showPreview ? "flex-1 border-r border-gray-200" : "flex-1"}`}
      >
        <div className="flex-shrink-0 px-3 py-1.5 text-xs font-medium text-gray-500 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
          <span>{label}</span>
          <span className="text-[10px] text-gray-400 tabular-nums">L{cursorLine}</span>
        </div>
        <div ref={hostRef} className="flex-1 min-h-0 min-w-0 overflow-hidden" />
      </div>

      {showPreview && (
        <div className="flex-1 min-w-0 min-h-0 flex flex-col bg-white">
          <div className="flex-shrink-0 px-3 py-1.5 text-xs font-medium text-gray-500 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
            <span>实时预览</span>
            {contentKey ? (
              <span className="text-[10px] uppercase tracking-wide text-gray-400">{contentKey}</span>
            ) : null}
          </div>
          <div
            ref={previewScrollRef}
            onScroll={handlePreviewScroll}
            className="flex-1 min-h-0 overflow-auto p-4"
          >
            <MarkdownHtmlPreview html={previewHtml} className={PREVIEW_CLASS} />
          </div>
        </div>
      )}
    </div>
  );
}
