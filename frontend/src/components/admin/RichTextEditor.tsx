import { useEditor, EditorContent } from "@tiptap/react";
import { NodeSelection } from "@tiptap/pm/state";
import { useState, useEffect, useRef, useMemo } from "react";
import type { Editor } from "@tiptap/react";
import ImagePickerModal from "@/components/admin/ImagePickerModal";
import MediaPickerModal from "@/components/admin/MediaPickerModal";
import GalleryPickerModal from "@/components/admin/GalleryPickerModal";
import EmbedUrlModal from "@/components/admin/EmbedUrlModal";
import { getPreset } from "@/components/admin/editor/presets";
import EditorToolbarComponent from "@/components/admin/editor/EditorToolbar";
import EditorBubbleMenu from "@/components/admin/editor/EditorBubbleMenu";
import TableBubbleMenu from "@/components/admin/editor/TableBubbleMenu";
import EditorFloatingMenu from "@/components/admin/editor/EditorFloatingMenu";
import type { ModalControls, ModalState } from "@/components/admin/editor/types-internal";

// ── Re-export backward-compatible API ──
export { ToolbarButton, ToolbarDivider } from "@/components/admin/editor/EditorToolbar";
export type { ModalControls, ModalState };

// Legacy exports: EDITOR_EXTENSIONS and getEditorExtensions
import { buildExtensions } from "@/components/admin/editor/extension-groups";

export const EDITOR_EXTENSIONS = buildExtensions(
  { slashCommands: true, blockHandles: true, blockToolbar: true, imagePaste: false, dragDrop: true },
);

/** @deprecated Use getPreset("full").extensions() instead */
// eslint-disable-next-line react-refresh/only-export-components
export function getEditorExtensions() {
  return getPreset("full").extensions();
}

// ── EditorToolbar wrapper (backward-compatible signature) ──
import { memo } from "react";
import { fullPreset } from "@/components/admin/editor/presets";

/** Backward-compatible EditorToolbar — uses full preset's toolbar config */
export const EditorToolbar = memo(function EditorToolbar({ editor, modals }: { editor: Editor; modals: ModalControls }) {
  if (!fullPreset.toolbar) return null;
  return <EditorToolbarComponent editor={editor} modals={modals} config={fullPreset.toolbar} />;
});

// ── Editor Modals (exported for external use) ──
export function EditorModals({ editor, state }: { editor: Editor; state: ModalState }) {
  const handleImageSelect = (item: { url: string; filename: string }) => {
    const { selection } = editor.state;
    if (selection instanceof NodeSelection && selection.node.type.name === "image") {
      const tr = editor.state.tr.setNodeMarkup(selection.from, undefined, {
        ...selection.node.attrs,
        src: item.url,
        alt: item.filename,
      });
      editor.view.dispatch(tr);
      editor.commands.focus();
    } else {
      editor.chain().focus().setImage({ src: item.url, alt: item.filename }).run();
    }
    state.setShowImagePicker(false);
  };

  return (
    <>
      <ImagePickerModal
        open={state.showImagePicker}
        onClose={() => state.setShowImagePicker(false)}
        onSelect={handleImageSelect}
      />
      <GalleryPickerModal
        open={state.showGalleryPicker}
        onClose={() => state.setShowGalleryPicker(false)}
        onConfirm={(items) => {
          const images = items.map((i) => ({ src: i.url, alt: i.filename }));
          (editor.commands as any).setImageGallery({ images, columns: Math.min(images.length, 3) });
          state.setShowGalleryPicker(false);
        }}
      />
      <MediaPickerModal
        open={state.showVideoPicker}
        onClose={() => state.setShowVideoPicker(false)}
        onSelect={(item) => { (editor.commands as any).setVideo({ src: item.url }); state.setShowVideoPicker(false); }}
        accept="video/*" type="video" title="选择视频"
      />
      <MediaPickerModal
        open={state.showAudioPicker}
        onClose={() => state.setShowAudioPicker(false)}
        onSelect={(item) => { (editor.commands as any).setAudio({ src: item.url }); state.setShowAudioPicker(false); }}
        accept="audio/*" type="audio" title="选择音频"
      />
      <EmbedUrlModal
        open={state.showEmbedUrl}
        onClose={() => state.setShowEmbedUrl(false)}
        onConfirm={(result) => {
          if (result.type === "youtube") editor.commands.setYoutubeVideo({ src: result.url });
          else (editor.commands as any).setIframe({ src: result.url });
          state.setShowEmbedUrl(false);
        }}
      />
    </>
  );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useModalState(): { modals: ModalControls; state: ModalState } {
  const [showImagePicker, setShowImagePicker] = useState(false);
  const [showGalleryPicker, setShowGalleryPicker] = useState(false);
  const [showVideoPicker, setShowVideoPicker] = useState(false);
  const [showAudioPicker, setShowAudioPicker] = useState(false);
  const [showEmbedUrl, setShowEmbedUrl] = useState(false);

  const modals = useMemo<ModalControls>(() => ({
    openImagePicker: () => setShowImagePicker(true),
    openGalleryPicker: () => setShowGalleryPicker(true),
    openVideoPicker: () => setShowVideoPicker(true),
    openAudioPicker: () => setShowAudioPicker(true),
    openEmbedUrl: () => setShowEmbedUrl(true),
  }), []);

  return {
    modals,
    state: {
      showImagePicker, setShowImagePicker,
      showGalleryPicker, setShowGalleryPicker,
      showVideoPicker, setShowVideoPicker,
      showAudioPicker, setShowAudioPicker,
      showEmbedUrl, setShowEmbedUrl,
    },
  };
}

// ── Standalone RichTextEditor ──

interface RichTextEditorProps {
  value: string;
  onChange: (value: string) => void;
  preset?: "full" | "standard" | "minimal";
}

export default function RichTextEditor({ value, onChange, preset = "full" }: RichTextEditorProps) {
  const { modals, state } = useModalState();
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  const presetConfig = useMemo(() => getPreset(preset), [preset]);
  const extensions = useMemo(() => presetConfig.extensions(), [presetConfig]);

  const editor = useEditor({
    extensions,
    content: value,
    onUpdate: ({ editor: e }) => {
      onChangeRef.current(e.getHTML());
    },
    editorProps: { attributes: { class: "tiptap" } },
  });

  // Wire up the "替换" button from ResizableMedia
  useEffect(() => {
    if (!editor) return;
    const handleReplace = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (detail?.type === "image") state.setShowImagePicker(true);
      else if (detail?.type === "video") state.setShowVideoPicker(true);
    };
    document.addEventListener("editor-replace-media", handleReplace);
    return () => document.removeEventListener("editor-replace-media", handleReplace);
  }, [editor, state]);

  // Sync external value to editor
  useEffect(() => {
    if (editor && value !== editor.getHTML()) {
      editor.commands.setContent(value, { emitUpdate: false });
    }
  }, [value, editor]);

  if (!editor) return null;

  return (
    <div className="border border-gray-300 rounded-lg overflow-hidden">
      {presetConfig.toolbar && (
        <EditorToolbarComponent editor={editor} modals={modals} config={presetConfig.toolbar} />
      )}
      {presetConfig.bubbleMenu && <EditorBubbleMenu editor={editor} />}
      {presetConfig.bubbleMenu && <TableBubbleMenu editor={editor} />}
      {presetConfig.floatingMenu && <EditorFloatingMenu editor={editor} />}
      <EditorContent editor={editor} />
      <EditorModals editor={editor} state={state} />
    </div>
  );
}
