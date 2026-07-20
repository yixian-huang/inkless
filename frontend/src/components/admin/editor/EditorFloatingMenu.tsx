import { FloatingMenuPlugin } from "@tiptap/extension-floating-menu";
import { useState, useEffect, useRef } from "react";
import type { Editor } from "@tiptap/react";

interface EditorFloatingMenuProps {
  editor: Editor;
}

export default function EditorFloatingMenu({ editor }: EditorFloatingMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (!menuRef.current) return;

    const plugin = FloatingMenuPlugin({
      pluginKey: "editorFloatingMenu",
      editor,
      element: menuRef.current,
      shouldShow: ({ editor: e, state }) => {
        const { selection } = state;
        const { $anchor, empty } = selection;
        if (!empty) return false;
        const isEmptyParagraph =
          $anchor.parent.type.name === "paragraph" && $anchor.parent.content.size === 0;
        if (e.isActive("codeBlock")) return false;
        return isEmptyParagraph;
      },
    });

    editor.registerPlugin(plugin);
    const el = menuRef.current;
    const observer = new MutationObserver(() => {
      setVisible(el.style.visibility !== "hidden" && el.style.display !== "none");
    });
    observer.observe(el, { attributes: true, attributeFilter: ["style"] });

    return () => {
      editor.unregisterPlugin("editorFloatingMenu");
      observer.disconnect();
    };
  }, [editor]);

  const handleClick = () => {
    editor.chain().focus().insertContent("/").run();
  };

  return (
    <div ref={menuRef} className="floating-menu-wrapper" style={{ visibility: "hidden" }}>
      {visible && (
        <button
          type="button"
          onClick={handleClick}
          className="floating-menu-btn"
          title="插入内容块"
          aria-label="插入内容块"
        >
          <svg
            width="15"
            height="15"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.25"
            strokeLinecap="round"
            aria-hidden
          >
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
        </button>
      )}
    </div>
  );
}
