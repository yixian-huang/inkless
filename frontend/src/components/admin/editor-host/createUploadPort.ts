/**
 * Inkless host adapter: MediaUploadPort → mediaUploadTracked bus + API.
 * Only editor-host (and app pages) may import @/lib/mediaUploadTracked for this purpose.
 */
import { uploadAndInsertImage } from "@/lib/mediaUploadTracked";
import type { MediaUploadPort } from "@/components/admin/editor/ports/types";

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
