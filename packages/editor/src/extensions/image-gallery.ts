import { Node, mergeAttributes } from "@tiptap/core";

export interface GalleryImage {
  src: string;
  alt?: string;
}

export const ImageGallery = Node.create({
  name: "imageGallery",
  group: "block",
  atom: true,

  addAttributes() {
    return {
      images: {
        default: [],
        parseHTML: (element) => {
          try {
            return JSON.parse(element.getAttribute("data-images") || "[]");
          } catch {
            return [];
          }
        },
        renderHTML: (attributes) => ({
          "data-images": JSON.stringify(attributes.images),
        }),
      },
      columns: { default: 3 },
    };
  },

  parseHTML() {
    return [{ tag: 'div[data-type="image-gallery"]' }];
  },

  renderHTML({ node, HTMLAttributes }) {
    const imgs: GalleryImage[] = Array.isArray(node.attrs.images) ? node.attrs.images : [];
    const cols = node.attrs.columns || 3;

    const children: any[] = imgs.map((img: GalleryImage) => [
      "div",
      { class: "gallery-item" },
      ["img", { src: img.src, alt: img.alt || "" }],
    ]);

    return [
      "div",
      mergeAttributes(HTMLAttributes, {
        "data-type": "image-gallery",
        class: `image-gallery cols-${cols}`,
      }),
      ...children,
    ];
  },

  addCommands() {
    return {
      setImageGallery: (options: { images: GalleryImage[]; columns?: number }) => ({ commands }: any) => {
        return commands.insertContent({
          type: this.name,
          attrs: { images: options.images, columns: options.columns || 3 },
        });
      },
    } as any;
  },
});
