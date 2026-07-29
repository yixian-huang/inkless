import { NodeSelection } from "@tiptap/pm/state";
import type { Editor } from "@tiptap/react";
import ImagePickerModal from "@/components/admin/ImagePickerModal";
import MediaPickerModal from "@/components/admin/MediaPickerModal";
import GalleryPickerModal from "@/components/admin/GalleryPickerModal";
import EmbedUrlModal from "@/components/admin/EmbedUrlModal";
import type { ModalState } from "@inkless/editor";
import type { MediaPickConsumers } from "./useInklessMediaPicker";

/**
 * Host media modals for a TipTap editor instance.
 * If a pending picker callback exists (slash / replace), it runs first;
 * otherwise applies the default toolbar insert / NodeSelection replace path.
 */
export function InklessEditorModals({
  editor,
  state,
  consumers,
}: {
  editor: Editor;
  state: ModalState;
  consumers: MediaPickConsumers;
}) {
  const handleImageSelect = (item: { url: string; filename: string }) => {
    const ref = { url: item.url, filename: item.filename };
    if (consumers.consumeImagePick(ref)) {
      state.setShowImagePicker(false);
      return;
    }

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
          const images = items.map((i) => ({
            url: i.url,
            filename: i.filename,
          }));
          if (
            consumers.consumeGalleryPick({
              images,
              columns: Math.min(images.length, 3),
            })
          ) {
            state.setShowGalleryPicker(false);
            return;
          }
          (editor.commands as any).setImageGallery({
            images: images.map((i) => ({ src: i.url, alt: i.filename })),
            columns: Math.min(images.length, 3),
          });
          state.setShowGalleryPicker(false);
        }}
      />
      <MediaPickerModal
        open={state.showVideoPicker}
        onClose={() => state.setShowVideoPicker(false)}
        onSelect={(item) => {
          const ref = { url: item.url, filename: item.filename || "video" };
          if (consumers.consumeVideoPick(ref)) {
            state.setShowVideoPicker(false);
            return;
          }
          const { selection } = editor.state;
          if (selection instanceof NodeSelection && selection.node.type.name === "video") {
            const tr = editor.state.tr.setNodeMarkup(selection.from, undefined, {
              ...selection.node.attrs,
              src: item.url,
            });
            editor.view.dispatch(tr);
            editor.commands.focus();
          } else {
            (editor.commands as any).setVideo({ src: item.url });
          }
          state.setShowVideoPicker(false);
        }}
        accept="video/*"
        type="video"
        title="选择视频"
      />
      <MediaPickerModal
        open={state.showAudioPicker}
        onClose={() => state.setShowAudioPicker(false)}
        onSelect={(item) => {
          const ref = { url: item.url, filename: item.filename || "audio" };
          if (consumers.consumeAudioPick(ref)) {
            state.setShowAudioPicker(false);
            return;
          }
          (editor.commands as any).setAudio({ src: item.url });
          state.setShowAudioPicker(false);
        }}
        accept="audio/*"
        type="audio"
        title="选择音频"
      />
      <EmbedUrlModal
        open={state.showEmbedUrl}
        onClose={() => state.setShowEmbedUrl(false)}
        onConfirm={(result) => {
          if (consumers.consumeEmbedPick(result)) {
            state.setShowEmbedUrl(false);
            return;
          }
          if (result.type === "youtube") {
            editor.commands.setYoutubeVideo({ src: result.url });
          } else {
            (editor.commands as any).setIframe({ src: result.url });
          }
          state.setShowEmbedUrl(false);
        }}
      />
    </>
  );
}

export default InklessEditorModals;
