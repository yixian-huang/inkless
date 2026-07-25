import { Extension } from "@tiptap/core";

export const LineHeight = Extension.create({
  name: "lineHeight",

  addGlobalAttributes() {
    return [
      {
        types: ["paragraph", "heading"],
        attributes: {
          lineHeight: {
            default: null,
            parseHTML: (element) => element.style.lineHeight || null,
            renderHTML: (attributes) => {
              if (!attributes.lineHeight) return {};
              return { style: `line-height: ${attributes.lineHeight}` };
            },
          },
        },
      },
    ];
  },

  addCommands() {
    return {
      setLineHeight: (height: string) => ({ commands }: any) => {
        const a = commands.updateAttributes("paragraph", { lineHeight: height });
        const b = commands.updateAttributes("heading", { lineHeight: height });
        return a || b;
      },
      unsetLineHeight: () => ({ commands }: any) => {
        const a = commands.resetAttributes("paragraph", "lineHeight");
        const b = commands.resetAttributes("heading", "lineHeight");
        return a || b;
      },
    } as any;
  },
});
