import { useEditor, EditorContent } from "@tiptap/react";
import { useEffect, useRef, useMemo, memo } from "react";
import type { Editor } from "@tiptap/react";
import { getPreset, fullPreset } from "@/components/admin/editor/presets";
import EditorToolbarComponent from "@/components/admin/editor/EditorToolbar";
import EditorBubbleMenu from "@/components/admin/editor/EditorBubbleMenu";
import TableBubbleMenu from "@/components/admin/editor/TableBubbleMenu";
import EditorFloatingMenu from "@/components/admin/editor/EditorFloatingMenu";
import type { ModalControls, ModalState } from "@/components/admin/editor/types-internal";
import type { EditorPorts } from "@/components/admin/editor/ports/types";
import { buildExtensions } from "@/components/admin/editor/extension-groups";
import { createInklessUploadPort } from "@/components/admin/editor-host/createUploadPort";
import { useInklessMediaPicker } from "@/components/admin/editor-host/useInklessMediaPicker";
import {
  InklessEditorModals,
} from "@/components/admin/editor-host/InklessEditorModals";

// ── Re-export backward-compatible API ──
export { ToolbarButton, ToolbarDivider } from "@/components/admin/editor/EditorToolbar";
export type { ModalControls, ModalState };
export { useModalState } from "@/components/admin/editor/useModalState";

/** @deprecated Prefer InklessEditorModals from editor-host */
export { InklessEditorModals as EditorModals };

export const EDITOR_EXTENSIONS = buildExtensions(
  { slashCommands: true, blockHandles: true, blockToolbar: true, imagePaste: false, dragDrop: true },
);

/**
 * @deprecated Prefer `getPreset("full").extensions(ports)` with explicit host ports.
 * When called with no args, injects inkless upload only (no picker).
 */
export function getEditorExtensions(ports?: EditorPorts) {
  return getPreset("full").extensions({
    ...ports,
    upload: ports?.upload ?? createInklessUploadPort(),
  });
}

/** Backward-compatible EditorToolbar — uses full preset's toolbar config */
export const EditorToolbar = memo(function EditorToolbar({
  editor,
  modals,
}: {
  editor: Editor;
  modals: ModalControls;
}) {
  if (!fullPreset.toolbar) return null;
  return <EditorToolbarComponent editor={editor} modals={modals} config={fullPreset.toolbar} />;
});

// ── Standalone RichTextEditor ──

interface RichTextEditorProps {
  value: string;
  onChange: (value: string) => void;
  preset?: "full" | "standard" | "minimal";
}

export default function RichTextEditor({
  value,
  onChange,
  preset = "full",
}: RichTextEditorProps) {
  const { modals, state, picker, consumers } = useInklessMediaPicker();
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  const presetConfig = useMemo(() => getPreset(preset), [preset]);
  const ports = useMemo<EditorPorts>(
    () => ({
      upload: createInklessUploadPort(),
      picker,
    }),
    [picker],
  );
  const extensions = useMemo(
    () => presetConfig.extensions(ports),
    [presetConfig, ports],
  );

  const editor = useEditor({
    extensions,
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
      {presetConfig.toolbar && (
        <EditorToolbarComponent editor={editor} modals={modals} config={presetConfig.toolbar} />
      )}
      {presetConfig.bubbleMenu && <EditorBubbleMenu editor={editor} />}
      {presetConfig.bubbleMenu && <TableBubbleMenu editor={editor} />}
      {presetConfig.floatingMenu && <EditorFloatingMenu editor={editor} />}
      <EditorContent editor={editor} />
      <InklessEditorModals editor={editor} state={state} consumers={consumers} />
    </div>
  );
}
