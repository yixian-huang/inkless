import { useEffect, useMemo, useRef } from "react";
import { useEditor, type Editor } from "@tiptap/react";
import { getEditorExtensions } from "@/components/admin/RichTextEditor";

type InnerProps = {
  /** Seed / external body HTML */
  html: string;
  editable: boolean;
  onDirty: () => void;
  onEditor: (editor: Editor | null) => void;
  /** Flush latest HTML when unmounting so parent state stays authoritative. */
  onFlushBody?: (html: string) => void;
};

/**
 * Creates a TipTap instance (no DOM). Parent renders EditorContent elsewhere.
 * Used for both ZH and EN — gate with `enabled` for on-demand mount.
 */
function LangEditorMountInner({
  html,
  editable,
  onDirty,
  onEditor,
  onFlushBody,
}: InnerProps) {
  const extensions = useMemo(() => getEditorExtensions(), []);
  const onDirtyRef = useRef(onDirty);
  onDirtyRef.current = onDirty;
  const onFlushBodyRef = useRef(onFlushBody);
  onFlushBodyRef.current = onFlushBody;
  const htmlRef = useRef(html);
  htmlRef.current = html;

  const editor = useEditor({
    extensions,
    content: html,
    shouldRerenderOnTransaction: false,
    editable,
    editorProps: { attributes: { class: "tiptap" } },
    onUpdate: () => {
      onDirtyRef.current();
    },
  });

  // Publish editor instance to parent
  useEffect(() => {
    onEditor(editor);
    return () => {
      if (editor) {
        onFlushBodyRef.current?.(editor.getHTML());
      }
      onEditor(null);
    };
  }, [editor, onEditor]);

  // External html → editor (hydrate / restore / template)
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

/**
 * Gate: only mount TipTap when `enabled`. Unmounting destroys the editor.
 */
export function LangEditorMount({
  enabled,
  html,
  editable,
  onDirty,
  onEditor,
  onFlushBody,
}: InnerProps & { enabled: boolean }) {
  if (!enabled) return null;
  return (
    <LangEditorMountInner
      html={html}
      editable={editable}
      onDirty={onDirty}
      onEditor={onEditor}
      onFlushBody={onFlushBody}
    />
  );
}
