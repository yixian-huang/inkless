import { Extension } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";

export interface ImagePasteOptions {
  /** Upload a file and return its URL */
  uploadFn: (file: File) => Promise<{ url: string; filename: string }>;
  /** Max file size in bytes (default: 20MB) */
  maxSize?: number;
}

/**
 * Convert a base64 data URL to a File object.
 */
function dataUrlToFile(dataUrl: string, filename: string): File | null {
  try {
    const [header, base64] = dataUrl.split(",");
    if (!header || !base64) return null;
    const mimeMatch = header.match(/data:([^;]+)/);
    if (!mimeMatch) return null;
    const mime = mimeMatch[1];
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    return new File([bytes], filename, { type: mime });
  } catch {
    return null;
  }
}

export const ImagePaste = Extension.create<ImagePasteOptions>({
  name: "imagePaste",

  addOptions() {
    return {
      uploadFn: async () => ({ url: "", filename: "" }),
      maxSize: 20 * 1024 * 1024, // 20MB
    };
  },

  addProseMirrorPlugins() {
    const uploadFn = this.options.uploadFn;
    const maxSize = this.options.maxSize || 20 * 1024 * 1024;
    const editor = this.editor;
    let uploading = false;

    const doUpload = (file: File) => {
      if (uploading) return;
      if (file.size > maxSize) {
        console.warn(`Image too large (${(file.size / 1024 / 1024).toFixed(1)}MB), skipping upload`);
        return;
      }
      uploading = true;
      uploadFn(file)
        .then(({ url, filename }) => {
          editor.chain().focus().setImage({ src: url, alt: filename }).run();
        })
        .catch((err) => {
          console.error("Image upload failed:", err);
        })
        .finally(() => {
          uploading = false;
        });
    };

    return [
      new Plugin({
        key: new PluginKey("imagePaste"),
        props: {
          handlePaste(_view, event) {
            const clipboardData = event.clipboardData;
            if (!clipboardData) return false;

            // Check for direct image file in clipboard items (e.g. screenshot paste)
            const items = Array.from(clipboardData.items || []);
            const imageItem = items.find((i) => i.type.startsWith("image/"));

            if (imageItem) {
              const file = imageItem.getAsFile();
              if (!file) return false;
              event.preventDefault();
              doUpload(file);
              return true;
            }

            // Check if pasted HTML contains base64 images that would freeze ProseMirror
            const html = clipboardData.getData("text/html");
            if (html && /src=["']data:image\/[^"']{1000,}["']/i.test(html)) {
              // HTML contains large base64 images — strip them to prevent freeze
              // Extract base64 images and upload them
              event.preventDefault();
              const base64Regex = /src=["'](data:image\/[^"']+)["']/gi;
              const matches = [...html.matchAll(base64Regex)];

              if (matches.length > 0) {
                // Upload the first base64 image found
                const dataUrl = matches[0][1];
                const ext = dataUrl.match(/data:image\/(\w+)/)?.[1] || "png";
                const file = dataUrlToFile(dataUrl, `pasted-image.${ext}`);
                if (file) {
                  doUpload(file);
                }
              }

              // Also insert any plain text content if present
              const text = clipboardData.getData("text/plain");
              if (text && matches.length === 0) {
                editor.chain().focus().insertContent(text).run();
              }

              return true;
            }

            return false;
          },

          handleDrop(_view, event) {
            const files = Array.from(event.dataTransfer?.files || []);
            const imageFile = files.find((f) => f.type.startsWith("image/"));
            if (!imageFile) return false;

            event.preventDefault();
            doUpload(imageFile);
            return true;
          },
        },
      }),
    ];
  },
});
