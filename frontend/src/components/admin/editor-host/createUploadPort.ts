/**
 * Inkless host adapter: MediaUploadPort → mediaUploadTracked bus + API.
 * Only editor-host (and app pages) may import @/lib/mediaUploadTracked for this purpose.
 */
import { uploadAndInsertImage } from "@/lib/mediaUploadTracked";
import type { MediaUploadPort } from "@inkless/editor";

export function createInklessUploadPort(): MediaUploadPort {
  return {
    uploadImage(file, insert, opts) {
      uploadAndInsertImage(
        file,
        (url, filename) => {
          if (!url) return;
          insert({ url, filename });
        },
        opts,
      );
    },
  };
}
