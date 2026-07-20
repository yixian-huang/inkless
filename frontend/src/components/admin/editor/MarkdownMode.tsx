import { useEffect, useMemo, useState } from "react";
import { markdownToHtml } from "@/lib/markdown";
import MarkdownHtmlPreview from "./MermaidPreview";

interface MarkdownModeProps {
  value: string;
  onChange: (value: string) => void;
  onImageUpload?: (file: File) => Promise<string>;
}

export default function MarkdownMode({ value, onChange, onImageUpload }: MarkdownModeProps) {
  // Debounce preview slightly for smoother typing with mermaid.
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const t = window.setTimeout(() => setDebounced(value), 200);
    return () => window.clearTimeout(t);
  }, [value]);

  const previewHtml = useMemo(() => markdownToHtml(debounced), [debounced]);

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    if (!onImageUpload) return;
    const files = Array.from(e.dataTransfer.files).filter((f) => f.type.startsWith("image/"));
    for (const file of files) {
      const url = await onImageUpload(file);
      const md = `![${file.name}](${url})`;
      onChange(value + "\n" + md);
    }
  };

  const handlePaste = async (e: React.ClipboardEvent) => {
    if (!onImageUpload) return;
    const items = Array.from(e.clipboardData.items);
    for (const item of items) {
      if (item.type.startsWith("image/")) {
        e.preventDefault();
        const file = item.getAsFile();
        if (file) {
          const url = await onImageUpload(file);
          const md = `![image](${url})`;
          onChange(value + "\n" + md);
        }
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    const { selectionStart, selectionEnd } = e.currentTarget;
    const selected = value.substring(selectionStart, selectionEnd);

    if (e.ctrlKey || e.metaKey) {
      if (e.key === "b") {
        e.preventDefault();
        onChange(value.substring(0, selectionStart) + `**${selected || "bold"}**` + value.substring(selectionEnd));
      } else if (e.key === "i") {
        e.preventDefault();
        onChange(value.substring(0, selectionStart) + `*${selected || "italic"}*` + value.substring(selectionEnd));
      } else if (e.key === "k") {
        e.preventDefault();
        onChange(value.substring(0, selectionStart) + `[${selected || "text"}](url)` + value.substring(selectionEnd));
      }
    }

    // Tab inserts 2 spaces
    if (e.key === "Tab") {
      e.preventDefault();
      const start = selectionStart;
      const end = selectionEnd;
      const next = value.substring(0, start) + "  " + value.substring(end);
      onChange(next);
      requestAnimationFrame(() => {
        e.currentTarget.selectionStart = e.currentTarget.selectionEnd = start + 2;
      });
    }
  };

  return (
    <div className="flex h-full min-h-0 gap-0 border border-gray-200 rounded-lg overflow-hidden bg-white">
      {/* Editor */}
      <div className="flex-1 min-w-0 min-h-0 flex flex-col border-r border-gray-200">
        <div className="flex-shrink-0 px-3 py-1.5 text-xs font-medium text-gray-500 bg-gray-50 border-b border-gray-200">
          Markdown
        </div>
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onDrop={handleDrop}
          onPaste={handlePaste}
          onKeyDown={handleKeyDown}
          className="flex-1 min-h-0 w-full font-mono text-sm p-4 resize-none focus:ring-0 focus:outline-none bg-white"
          placeholder={"# 标题\n\n支持 **Markdown** 与 ```mermaid 图表``` 实时预览…"}
          spellCheck={false}
        />
      </div>

      {/* Live preview */}
      <div className="flex-1 min-w-0 min-h-0 flex flex-col bg-white">
        <div className="flex-shrink-0 px-3 py-1.5 text-xs font-medium text-gray-500 bg-gray-50 border-b border-gray-200">
          实时预览
        </div>
        <div className="flex-1 min-h-0 overflow-auto p-4">
          <MarkdownHtmlPreview
            html={previewHtml}
            className="markdown-preview article-typography max-w-none text-sm leading-relaxed
              [&_h1]:text-2xl [&_h1]:font-bold [&_h1]:mb-3
              [&_h2]:text-xl [&_h2]:font-semibold [&_h2]:mb-2 [&_h2]:mt-4
              [&_h3]:text-lg [&_h3]:font-semibold [&_h3]:mb-2 [&_h3]:mt-3
              [&_p]:mb-3 [&_p]:leading-relaxed
              [&_ul]:list-disc [&_ul]:pl-6 [&_ul]:mb-3
              [&_ol]:list-decimal [&_ol]:pl-6 [&_ol]:mb-3
              [&_blockquote]:border-l-4 [&_blockquote]:border-gray-300 [&_blockquote]:pl-3 [&_blockquote]:text-gray-600 [&_blockquote]:italic
              [&_a]:text-blue-600 [&_a]:underline
              [&_code]:bg-gray-100 [&_code]:px-1 [&_code]:rounded [&_code]:text-[0.9em]
              [&_pre]:bg-gray-900 [&_pre]:text-gray-100 [&_pre]:p-3 [&_pre]:rounded-lg [&_pre]:overflow-x-auto [&_pre]:mb-3
              [&_pre_code]:bg-transparent [&_pre_code]:p-0
              [&_img]:max-w-full [&_img]:rounded
              [&_table]:w-full [&_table]:border-collapse [&_th]:border [&_td]:border [&_th]:px-2 [&_td]:px-2 [&_th]:bg-gray-50
              [&_.mermaid]:my-4 [&_.mermaid]:flex [&_.mermaid]:justify-center [&_.mermaid]:overflow-x-auto"
          />
        </div>
      </div>
    </div>
  );
}
