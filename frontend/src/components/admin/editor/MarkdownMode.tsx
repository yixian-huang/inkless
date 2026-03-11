import { useState, useEffect } from "react";
import { marked } from "marked";

interface MarkdownModeProps {
  value: string;
  onChange: (value: string) => void;
  onImageUpload?: (file: File) => Promise<string>;
}

export default function MarkdownMode({ value, onChange, onImageUpload }: MarkdownModeProps) {
  const [preview, setPreview] = useState("");

  useEffect(() => {
    let cancelled = false;
    const html = marked(value) as string;
    if (!cancelled) {
      setPreview(html);
    }
    return () => { cancelled = true; };
  }, [value]);

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    if (!onImageUpload) return;
    const files = Array.from(e.dataTransfer.files).filter(f => f.type.startsWith("image/"));
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
  };

  return (
    <div className="flex gap-4 h-full">
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onDrop={handleDrop}
        onPaste={handlePaste}
        onKeyDown={handleKeyDown}
        className="flex-1 font-mono text-sm p-4 border rounded-lg resize-none focus:ring-2 focus:ring-blue-500 focus:outline-none"
        placeholder="Write Markdown here..."
      />
      <div
        className="flex-1 p-4 border rounded-lg overflow-auto prose prose-sm max-w-none"
        dangerouslySetInnerHTML={{ __html: preview }}
      />
    </div>
  );
}
