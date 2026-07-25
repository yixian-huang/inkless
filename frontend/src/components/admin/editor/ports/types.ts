/**
 * Host ports for the editor kit (Phase 0–1: upload only).
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

/** Ports passed into buildExtensions / presets (picker lands in Phase 2). */
export type EditorPorts = {
  upload?: MediaUploadPort;
};

/** Default max image size (20MB) — mirrors historical mediaUploadTracked default. */
export const DEFAULT_EDITOR_IMAGE_MAX_BYTES = 20 * 1024 * 1024;
