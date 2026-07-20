import { useState, useEffect, useRef } from "react";
import type { Editor } from "@tiptap/react";

interface LinkEditPopoverProps {
  editor: Editor;
  onClose: () => void;
}

export default function LinkEditPopover({ editor, onClose }: LinkEditPopoverProps) {
  const existingHref = editor.getAttributes("link").href || "";
  const existingTarget = editor.getAttributes("link").target || "";
  const [url, setUrl] = useState(existingHref);
  const [openInNewTab, setOpenInNewTab] = useState(existingTarget === "_blank");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  const handleSave = () => {
    if (!url.trim()) {
      handleRemove();
      return;
    }
    editor
      .chain()
      .focus()
      .extendMarkRange("link")
      .setLink({ href: url, target: openInNewTab ? "_blank" : null })
      .run();
    onClose();
  };

  const handleRemove = () => {
    editor.chain().focus().extendMarkRange("link").unsetLink().run();
    onClose();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleSave();
    }
    if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    }
  };

  return (
    <div className="link-edit-popover" onKeyDown={handleKeyDown}>
      <div className="flex items-center gap-1.5">
        <input
          ref={inputRef}
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://..."
          className="link-edit-input"
          aria-label="链接地址"
        />
        <button
          type="button"
          onClick={handleSave}
          className="link-edit-btn link-edit-btn-save"
          title="保存"
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden
          >
            <polyline points="20 6 9 17 4 12" />
          </svg>
        </button>
        {existingHref ? (
          <button
            type="button"
            onClick={handleRemove}
            className="link-edit-btn link-edit-btn-remove"
            title="删除链接"
          >
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        ) : null}
      </div>
      <label className="link-edit-check">
        <input
          type="checkbox"
          checked={openInNewTab}
          onChange={(e) => setOpenInNewTab(e.target.checked)}
        />
        新窗口打开
      </label>
    </div>
  );
}
