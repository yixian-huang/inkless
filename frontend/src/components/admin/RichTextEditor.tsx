import { useEditor, EditorContent } from "@tiptap/react";
import { useEffect, useRef, useMemo, memo } from "react";
import type { Editor } from "@tiptap/react";
import {
  createEditorKit,
  getEditorSurface,
  EditorToolbar as EditorToolbarComponent,
  EditorBubbleMenu,
  TableBubbleMenu,
  EditorFloatingMenu,
  ToolbarButton,
  ToolbarDivider,
  useModalState,
  type ModalControls,
  type ModalState,
  type EditorPorts,
  type EditorPresetName,
  type EditorKit,
  type EditorSurface,
} from "@inkless/editor";
import { createInklessUploadPort } from "@/components/admin/editor-host/createUploadPort";
import { useInklessMediaPicker } from "@/components/admin/editor-host/useInklessMediaPicker";
import { InklessEditorModals } from "@/components/admin/editor-host/InklessEditorModals";

// ── Re-export backward-compatible API ──
export { ToolbarButton, ToolbarDivider, useModalState };
export type { ModalControls, ModalState, EditorKit, EditorSurface, EditorPresetName };
export { createEditorKit, getEditorSurface };

/** @deprecated Prefer InklessEditorModals from editor-host */
export { InklessEditorModals as EditorModals };

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

// ── Standalone RichTextEditor (host-wired reference implementation) ──

interface RichTextEditorProps {
  value: string;
  onChange: (value: string) => void;
  preset?: EditorPresetName;
}

/**
 * Full-stack editor for the app: @inkless/editor kit + inkless media host.
 * Article page may still disassemble pieces, but should share createEditorKit / getEditorSurface.
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
