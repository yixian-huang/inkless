/**
 * Host ports for the editor kit.
 * Extensions and MarkdownMode depend on these types only — never on @/api or media bus.
 */

/** Minimal media identity returned after upload / pick. */
export type MediaRef = {
  url: string;
  filename: string;
};

export type UploadImageOptions = {
  maxSize?: number;
  filename?: string;
};

/**
 * Paste / drop image pipeline.
 * Host owns progress tray + retry; editor only inserts on success.
 */
export interface MediaUploadPort {
  uploadImage(
    file: File,
    insert: (ref: MediaRef) => void,
    opts?: UploadImageOptions,
  ): void;
}

export type GalleryPick = {
  images: MediaRef[];
  columns?: number;
};

export type EmbedPick =
  | { type: "youtube"; url: string }
  | { type: "bilibili"; url: string }
  | { type: "iframe"; url: string };

/**
 * Open host media pickers. Caller supplies insert/replace callback.
 * Toolbar may open without a callback (host applies default insert into editor).
 */
export interface MediaPickerPort {
  openImage(onPick?: (ref: MediaRef) => void): void;
  openGallery(onPick?: (pick: GalleryPick) => void): void;
  openVideo(onPick?: (ref: MediaRef) => void): void;
  openAudio(onPick?: (ref: MediaRef) => void): void;
  openEmbed(onPick?: (pick: EmbedPick) => void): void;
}

/** Ports passed into buildExtensions / presets. */
export type EditorPorts = {
  upload?: MediaUploadPort;
  picker?: MediaPickerPort;
};

/** Default max image size (20MB) — mirrors historical mediaUploadTracked default. */
export const DEFAULT_EDITOR_IMAGE_MAX_BYTES = 20 * 1024 * 1024;
