/** Modal controls interface — shared between toolbar and editor */
export interface ModalControls {
  openImagePicker: () => void;
  openGalleryPicker: () => void;
  openVideoPicker: () => void;
  openAudioPicker: () => void;
  openEmbedUrl: () => void;
}

export interface ModalState {
  showImagePicker: boolean;
  setShowImagePicker: (v: boolean) => void;
  showGalleryPicker: boolean;
  setShowGalleryPicker: (v: boolean) => void;
  showVideoPicker: boolean;
  setShowVideoPicker: (v: boolean) => void;
  showAudioPicker: boolean;
  setShowAudioPicker: (v: boolean) => void;
  showEmbedUrl: boolean;
  setShowEmbedUrl: (v: boolean) => void;
}
