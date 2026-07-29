import { EditorContent, type Editor } from "@tiptap/react";
import {
  EditorBubbleMenu,
  TableBubbleMenu,
  EditorFloatingMenu,
  getEditorSurface,
  type EditorPresetName,
} from "@inkless/editor";
import ArticleTypographyRoot from "@/components/blog/ArticleTypographyRoot";

const DEFAULT_SURFACE = getEditorSurface("full");

/**
 * TipTap canvas + bubble/floating menus.
 * Lazy-loaded with the richtext surface so Markdown-only sessions skip this chunk.
 */
export function RichTextSurface({
  editor,
  showMenus,
  metadata,
  preset = "full",
}: {
  editor: Editor;
  showMenus: boolean;
  metadata: Record<string, unknown>;
  /** Must match the preset used when mounting the editor (default: full). */
  preset?: EditorPresetName;
}) {
  const surface = preset === "full" ? DEFAULT_SURFACE : getEditorSurface(preset);

  return (
    <div className="h-full overflow-y-auto">
      {showMenus && (
        <>
          {surface.bubbleMenu && (
            <EditorBubbleMenu editor={editor} config={surface.bubbleMenu} />
          )}
          {surface.bubbleMenu && <TableBubbleMenu editor={editor} />}
          {surface.floatingMenu && <EditorFloatingMenu editor={editor} />}
        </>
      )}
      <ArticleTypographyRoot
        mode="editor"
        articleMetadata={metadata}
        className="h-full article-editor-content"
      >
        <EditorContent editor={editor} className="h-full" />
      </ArticleTypographyRoot>
    </div>
  );
}

export default RichTextSurface;
