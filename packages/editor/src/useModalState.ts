import { useMemo, useState } from "react";
import type { ModalControls, ModalState } from "./types-internal";

/** Light hook — no TipTap / extension imports (safe for editor page bootstrap). */
export function useModalState(): { modals: ModalControls; state: ModalState } {
  const [showImagePicker, setShowImagePicker] = useState(false);
  const [showGalleryPicker, setShowGalleryPicker] = useState(false);
  const [showVideoPicker, setShowVideoPicker] = useState(false);
  const [showAudioPicker, setShowAudioPicker] = useState(false);
  const [showEmbedUrl, setShowEmbedUrl] = useState(false);

  const modals = useMemo<ModalControls>(
    () => ({
      openImagePicker: () => setShowImagePicker(true),
      openGalleryPicker: () => setShowGalleryPicker(true),
      openVideoPicker: () => setShowVideoPicker(true),
      openAudioPicker: () => setShowAudioPicker(true),
      openEmbedUrl: () => setShowEmbedUrl(true),
    }),
    [],
  );

  return {
    modals,
    state: {
      showImagePicker,
      setShowImagePicker,
      showGalleryPicker,
      setShowGalleryPicker,
      showVideoPicker,
      setShowVideoPicker,
      showAudioPicker,
      setShowAudioPicker,
      showEmbedUrl,
      setShowEmbedUrl,
    },
  };
}
