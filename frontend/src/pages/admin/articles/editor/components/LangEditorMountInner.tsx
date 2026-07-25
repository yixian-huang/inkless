import { useEffect, useMemo, useRef } from "react";
import { useEditor, type Editor } from "@tiptap/react";
import { getPreset } from "@/components/admin/editor/presets";
import { createInklessUploadPort } from "@/components/admin/editor-host/createUploadPort";
import { sanitizePastedHtml } from "../utils/sanitizePastedHtml";

export type LangEditorMountInnerProps = {
  html: string;
  editable: boolean;
  onDirty: () => void;
  onEditor: (editor: Editor | null) => void;
  onFlushBody?: (html: string) => void;
};

/**
 * Heavy TipTap instance (extensions + lowlight + custom nodes).
 * Loaded via React.lazy from LangEditorMount — not on the article page bootstrap path.
 */
export function LangEditorMountInner({
  html,
  editable,
  onDirty,
  onEditor,
  onFlushBody,
}: LangEditorMountInnerProps) {
  const ports = useMemo(() => ({ upload: createInklessUploadPort() }), []);
  const extensions = useMemo(
    () => getPreset("full").extensions(ports),
    [ports],
  );
  const onDirtyRef = useRef(onDirty);
  onDirtyRef.current = onDirty;
  const onFlushBodyRef = useRef(onFlushBody);
  onFlushBodyRef.current = onFlushBody;

  const editor = useEditor({
    extensions,
    content: html,
    shouldRerenderOnTransaction: false,
    editable,
    editorProps: {
      attributes: { class: "tiptap" },
      transformPastedHTML: (pasted) => sanitizePastedHtml(pasted),
    },
    onUpdate: () => {
      onDirtyRef.current();
    },
  });

  useEffect(() => {
    onEditor(editor);
    return () => {
      if (editor) {
        onFlushBodyRef.current?.(editor.getHTML());
      }
      onEditor(null);
    };
  }, [editor, onEditor]);

  useEffect(() => {
    if (!editor) return;
    if (html && html !== editor.getHTML()) {
      editor.commands.setContent(html, { emitUpdate: false });
    }
  }, [html, editor]);

  useEffect(() => {
    editor?.setEditable(editable);
  }, [editor, editable]);

  return null;
}

export default LangEditorMountInner;
