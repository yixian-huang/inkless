import { BubbleMenuPlugin } from "@tiptap/extension-bubble-menu";
import { useState, useEffect, useRef, Fragment } from "react";
import type { Editor } from "@tiptap/react";
import LinkEditPopover from "./LinkEditPopover";
import type { BubbleMenuConfig } from "./types";
import { fullPreset } from "./presets";

interface EditorBubbleMenuProps {
  editor: Editor;
  /** When omitted, uses full preset bubble items. */
  config?: BubbleMenuConfig | null;
}

/** Visual grouping for auto-dividers between bubble items. */
const ITEM_GROUP: Record<string, number> = {
  bold: 0,
  italic: 0,
  underline: 0,
  strike: 0,
  code: 1,
  link: 1,
  highlight: 2,
  textColor: 2,
};

export default function EditorBubbleMenu({ editor, config }: EditorBubbleMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);
  const [showLinkEdit, setShowLinkEdit] = useState(false);

  const items = config?.items ?? fullPreset.bubbleMenu?.items ?? [];

  useEffect(() => {
    if (!menuRef.current) return;

    const plugin = BubbleMenuPlugin({
      pluginKey: "textBubbleMenu",
      editor,
      element: menuRef.current,
      shouldShow: ({ editor: e, from, to }) => {
        if (from === to) return false;
        if (e.state.selection.constructor.name === "NodeSelection") return false;
        if (e.isActive("codeBlock")) return false;
        if (e.isActive("table")) return false;
        return true;
      },
    });

    editor.registerPlugin(plugin);
    const el = menuRef.current;
    const observer = new MutationObserver(() => {
      const isVisible = el.style.visibility !== "hidden" && el.style.display !== "none";
      setVisible(isVisible);
      if (!isVisible) setShowLinkEdit(false);
    });
    observer.observe(el, { attributes: true, attributeFilter: ["style"] });

    return () => {
      editor.unregisterPlugin("textBubbleMenu");
      observer.disconnect();
    };
  }, [editor]);

  if (!items.length) return null;

  return (
    <div ref={menuRef} className="bubble-menu-wrapper" style={{ visibility: "hidden" }}>
      {visible && (
        <div className="bubble-menu" role="toolbar" aria-label="文本格式">
          {showLinkEdit ? (
            <LinkEditPopover editor={editor} onClose={() => setShowLinkEdit(false)} />
          ) : (
            <div className="flex items-center gap-0.5">
              {items.map((name, i) => {
                const prev = items[i - 1];
                const showDivider =
                  i > 0 &&
                  (ITEM_GROUP[prev] ?? 0) !== (ITEM_GROUP[name] ?? 0);
                const btn = renderBubbleItem(name, editor, () => setShowLinkEdit(true));
                if (!btn) return null;
                return (
                  <Fragment key={name}>
                    {showDivider ? <BubbleDivider /> : null}
                    {btn}
                  </Fragment>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function renderBubbleItem(
  name: string,
  editor: Editor,
  onLink: () => void,
): React.ReactNode {
  switch (name) {
    case "bold":
      return (
        <BubbleButton
          active={editor.isActive("bold")}
          onClick={() => editor.chain().focus().toggleBold().run()}
          title="粗体"
        >
          <strong>B</strong>
        </BubbleButton>
      );
    case "italic":
      return (
        <BubbleButton
          active={editor.isActive("italic")}
          onClick={() => editor.chain().focus().toggleItalic().run()}
          title="斜体"
        >
          <em>I</em>
        </BubbleButton>
      );
    case "underline":
      return (
        <BubbleButton
          active={editor.isActive("underline")}
          onClick={() => editor.chain().focus().toggleUnderline().run()}
          title="下划线"
        >
          <span className="underline">U</span>
        </BubbleButton>
      );
    case "strike":
      return (
        <BubbleButton
          active={editor.isActive("strike")}
          onClick={() => editor.chain().focus().toggleStrike().run()}
          title="删除线"
        >
          <s>S</s>
        </BubbleButton>
      );
    case "code":
      return (
        <BubbleButton
          active={editor.isActive("code")}
          onClick={() => editor.chain().focus().toggleCode().run()}
          title="行内代码"
        >
          <span className="font-mono text-[11px]">{"</>"}</span>
        </BubbleButton>
      );
    case "link":
      return (
        <BubbleButton
          active={editor.isActive("link")}
          onClick={onLink}
          title="链接"
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
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
          </svg>
        </BubbleButton>
      );
    case "highlight":
      return (
        <BubbleButton
          active={editor.isActive("highlight")}
          onClick={() =>
            editor.chain().focus().toggleHighlight({ color: "#fef08a" }).run()
          }
          title="高亮"
        >
          <span className="bubble-menu-highlight-swatch">H</span>
        </BubbleButton>
      );
    case "textColor":
      return (
        <BubbleButton
          active={!!editor.getAttributes("textStyle").color}
          onClick={() => {
            const cur = editor.getAttributes("textStyle").color;
            if (cur) editor.chain().focus().unsetColor().run();
            else editor.chain().focus().setColor("#2563eb").run();
          }}
          title="文字颜色"
        >
          <span className="font-semibold" style={{ color: "#2563eb" }}>
            A
          </span>
        </BubbleButton>
      );
    default:
      return null;
  }
}

function BubbleButton({
  active,
  onClick,
  title,
  children,
}: {
  active: boolean;
  onClick: () => void;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={title}
      aria-pressed={active}
      className={`bubble-menu-btn ${active ? "bubble-menu-btn-active" : ""}`}
    >
      {children}
    </button>
  );
}

function BubbleDivider() {
  return <div className="bubble-menu-divider" aria-hidden />;
}
