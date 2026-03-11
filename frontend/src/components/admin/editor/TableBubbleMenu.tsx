import { BubbleMenuPlugin } from "@tiptap/extension-bubble-menu";
import { useState, useEffect, useRef } from "react";
import type { Editor } from "@tiptap/react";

interface TableBubbleMenuProps {
  editor: Editor;
}

export default function TableBubbleMenu({ editor }: TableBubbleMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (!menuRef.current) return;

    const plugin = BubbleMenuPlugin({
      pluginKey: "tableBubbleMenu",
      editor,
      element: menuRef.current,
      shouldShow: ({ editor: e }) => e.isActive("table"),
    });

    editor.registerPlugin(plugin);
    const el = menuRef.current;
    const observer = new MutationObserver(() => {
      setVisible(el.style.visibility !== "hidden" && el.style.display !== "none");
    });
    observer.observe(el, { attributes: true, attributeFilter: ["style"] });

    return () => {
      editor.unregisterPlugin("tableBubbleMenu");
      observer.disconnect();
    };
  }, [editor]);

  return (
    <div ref={menuRef} className="bubble-menu-wrapper" style={{ visibility: "hidden" }}>
      {visible && (
        <div className="bubble-menu">
          <div className="flex items-center gap-0.5 flex-wrap">
            <TableBtn onClick={() => editor.chain().focus().addColumnBefore().run()} title="添加列 (左)">
              <span className="text-xs">+列←</span>
            </TableBtn>
            <TableBtn onClick={() => editor.chain().focus().addColumnAfter().run()} title="添加列 (右)">
              <span className="text-xs">+列→</span>
            </TableBtn>
            <TableBtn onClick={() => editor.chain().focus().deleteColumn().run()} title="删除列" danger>
              <span className="text-xs">删列</span>
            </TableBtn>
            <TableDivider />
            <TableBtn onClick={() => editor.chain().focus().addRowBefore().run()} title="添加行 (上)">
              <span className="text-xs">+行↑</span>
            </TableBtn>
            <TableBtn onClick={() => editor.chain().focus().addRowAfter().run()} title="添加行 (下)">
              <span className="text-xs">+行↓</span>
            </TableBtn>
            <TableBtn onClick={() => editor.chain().focus().deleteRow().run()} title="删除行" danger>
              <span className="text-xs">删行</span>
            </TableBtn>
            <TableDivider />
            <TableBtn onClick={() => editor.chain().focus().mergeCells().run()} title="合并单元格">
              <span className="text-xs">合并</span>
            </TableBtn>
            <TableBtn onClick={() => editor.chain().focus().splitCell().run()} title="拆分单元格">
              <span className="text-xs">拆分</span>
            </TableBtn>
            <TableBtn onClick={() => editor.chain().focus().toggleHeaderRow().run()} title="切换表头">
              <span className="text-xs">表头</span>
            </TableBtn>
            <TableDivider />
            <TableBtn onClick={() => editor.chain().focus().deleteTable().run()} title="删除表格" danger>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                <polyline points="3 6 5 6 21 6" />
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
              </svg>
            </TableBtn>
          </div>
        </div>
      )}
    </div>
  );
}

function TableBtn({ onClick, title, children, danger }: {
  onClick: () => void; title: string; children: React.ReactNode; danger?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={title}
      className={`bubble-menu-btn ${danger ? "bubble-menu-btn-danger" : ""}`}
    >
      {children}
    </button>
  );
}

function TableDivider() {
  return <div className="w-px h-4 bg-gray-600 mx-0.5" />;
}
