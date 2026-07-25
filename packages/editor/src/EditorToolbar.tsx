import { memo } from "react";
import { useEditorState } from "@tiptap/react";
import type { Editor } from "@tiptap/react";
import type { ToolbarConfig } from "./types";
import type { ModalControls } from "./types-internal";
import { getToolbarItem } from "./toolbar-registry";
import ToolbarColorPicker from "./ToolbarColorPicker";
import ToolbarDropdown from "./ToolbarDropdown";

interface EditorToolbarProps {
  editor: Editor;
  modals: ModalControls;
  config: ToolbarConfig;
}

/** Data-driven toolbar: renders rows of items from a ToolbarConfig */
const EditorToolbar = memo(function EditorToolbar({ editor, modals, config }: EditorToolbarProps) {
  useEditorState({
    editor,
    selector: ({ transactionNumber }) => transactionNumber,
  });

  return (
    <>
      {config.rows.map((row, rowIdx) => (
        <div key={rowIdx} className="flex flex-wrap items-center gap-0.5 px-2 py-1.5 border-b border-slate-200 bg-slate-50">
          {row.map((item, itemIdx) => {
            if (item === "divider") {
              return <ToolbarDivider key={`d-${itemIdx}`} />;
            }
            const reg = getToolbarItem(item);
            if (!reg) return null;

            if (reg.type === "dropdown" && reg.getOptions && reg.onSelect) {
              return (
                <ToolbarDropdown
                  key={reg.name}
                  label={reg.getLabel?.(editor) || reg.title}
                  title={reg.title}
                  options={reg.getOptions(editor)}
                  onSelect={(v) => reg.onSelect!(editor, v)}
                />
              );
            }

            if (reg.type === "color" && reg.getColor && reg.onColorChange && reg.onColorReset) {
              const isHighlight = reg.name === "highlight";
              return (
                <ToolbarColorPicker
                  key={reg.name}
                  color={reg.getColor(editor)}
                  onChange={(c) => reg.onColorChange!(editor, c)}
                  onReset={() => reg.onColorReset!(editor)}
                  icon={
                    isHighlight
                      ? <span className="text-xs font-bold bg-yellow-200 px-0.5">H</span>
                      : <span className="text-xs font-bold">A</span>
                  }
                  title={reg.title}
                />
              );
            }

            // Default: button
            return (
              <ToolbarButton
                key={reg.name}
                active={reg.isActive(editor)}
                onClick={() => reg.run(editor, modals)}
                title={reg.title}
              >
                {renderButtonContent(reg.name, reg.icon)}
              </ToolbarButton>
            );
          })}
        </div>
      ))}
    </>
  );
});

export default EditorToolbar;

/** Render button content — preserves the original visual styling */
function renderButtonContent(name: string, icon: string) {
  switch (name) {
    case "bold": return <strong>B</strong>;
    case "italic": return <em>I</em>;
    case "underline": return <span className="underline">U</span>;
    case "strike": return <s>S</s>;
    case "superscript": return <span className="text-xs">X<sup>2</sup></span>;
    case "subscript": return <span className="text-xs">X<sub>2</sub></span>;
    case "alignLeft": return <span className="text-xs">&#9776;</span>;
    case "alignCenter": return <span className="text-xs">&#8801;</span>;
    case "alignRight": return <span className="text-xs leading-none" style={{ transform: "scaleX(-1)", display: "inline-block" }}>&#9776;</span>;
    case "alignJustify": return <span className="text-xs">&#9783;</span>;
    case "codeBlock": return <span className="text-xs font-mono">&lt;/&gt;</span>;
    default: return <span className="text-xs">{icon}</span>;
  }
}

// ── Shared sub-components ──

export function ToolbarButton({ active, onClick, title, children }: {
  active: boolean; onClick: () => void; title: string; children: React.ReactNode;
}) {
  return (
    <button
      type="button" onClick={onClick} title={title}
      className={`px-2 py-1 text-sm rounded transition-colors ${
        active ? "bg-blue-100 text-blue-700" : "text-slate-600 hover:bg-slate-200 hover:text-slate-900"
      }`}
    >
      {children}
    </button>
  );
}

export function ToolbarDivider() {
  return <div className="w-px h-5 bg-slate-300 mx-1" />;
}
