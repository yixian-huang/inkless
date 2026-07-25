import { useEditor, EditorContent } from "@tiptap/react";
import { useEffect, useRef, useMemo, memo } from "react";
import type { Editor } from "@tiptap/react";
import EditorToolbarComponent from "@/components/admin/editor/EditorToolbar";
import EditorBubbleMenu from "@/components/admin/editor/EditorBubbleMenu";
import TableBubbleMenu from "@/components/admin/editor/TableBubbleMenu";
import EditorFloatingMenu from "@/components/admin/editor/EditorFloatingMenu";
import type { ModalControls, ModalState } from "@/components/admin/editor/types-internal";
import type { EditorPorts } from "@/components/admin/editor/ports/types";
import {
  createEditorKit,
  getEditorSurface,
  type EditorPresetName,
} from "@/components/admin/editor/createEditorKit";
import { createInklessUploadPort } from "@/components/admin/editor-host/createUploadPort";
import { useInklessMediaPicker } from "@/components/admin/editor-host/useInklessMediaPicker";
import { InklessEditorModals } from "@/components/admin/editor-host/InklessEditorModals";

// ── Re-export backward-compatible API ──
export { ToolbarButton, ToolbarDivider } from "@/components/admin/editor/EditorToolbar";
export type { ModalControls, ModalState };
export { useModalState } from "@/components/admin/editor/useModalState";

/** @deprecated Prefer InklessEditorModals from editor-host */
export { InklessEditorModals as EditorModals };

export { createEditorKit, getEditorSurface } from "@/components/admin/editor/createEditorKit";
export type { EditorKit, EditorSurface, EditorPresetName } from "@/components/admin/editor/createEditorKit";

const fullSurface = getEditorSurface("full");

/** Article / shell toolbar wired to the full preset chrome (no extension build). */
export const EditorToolbar = memo(function EditorToolbar({
  editor,
  modals,
}: {
  editor: Editor;
  modals: ModalControls;
}) {
  if (!fullSurface.toolbar) return null;
  return (
    <EditorToolbarComponent
      editor={editor}
      modals={modals}
      config={fullSurface.toolbar}
    />
  );
});

// ── Standalone RichTextEditor ──

interface RichTextEditorProps {
  value: string;
  onChange: (value: string) => void;
  preset?: EditorPresetName;
}

/**
 * Reference full-stack editor: createEditorKit + host ports + chrome + modals.
 * Article page may still disassemble pieces, but must share createEditorKit / getEditorSurface.
 */
export default function RichTextEditor({
  value,
  onChange,
  preset = "full",
}: RichTextEditorProps) {
  const { modals, state, picker, consumers } = useInklessMediaPicker();
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  const ports = useMemo<EditorPorts>(
    () => ({
      upload: createInklessUploadPort(),
      picker,
    }),
    [picker],
  );

  const kit = useMemo(
    () => createEditorKit(preset, ports),
    [preset, ports],
  );

  const editor = useEditor({
    extensions: kit.extensions,
    content: value,
    onUpdate: ({ editor: e }) => {
      onChangeRef.current(e.getHTML());
    },
    editorProps: { attributes: { class: "tiptap" } },
  });

  // Sync external value to editor
  useEffect(() => {
    if (editor && value !== editor.getHTML()) {
      editor.commands.setContent(value, { emitUpdate: false });
    }
  }, [value, editor]);

  if (!editor) return null;

  return (
    <div className="border border-slate-200 rounded-lg overflow-hidden">
      {kit.toolbar && (
        <EditorToolbarComponent editor={editor} modals={modals} config={kit.toolbar} />
      )}
      {kit.bubbleMenu && (
        <EditorBubbleMenu editor={editor} config={kit.bubbleMenu} />
      )}
      {kit.bubbleMenu && <TableBubbleMenu editor={editor} />}
      {kit.floatingMenu && <EditorFloatingMenu editor={editor} />}
      <EditorContent editor={editor} />
      <InklessEditorModals editor={editor} state={state} consumers={consumers} />
    </div>
  );
}
