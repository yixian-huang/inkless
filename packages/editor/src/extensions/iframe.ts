import { Node, mergeAttributes } from "@tiptap/core";

export const Iframe = Node.create({
  name: "iframe",
  group: "block",
  atom: true,

  addAttributes() {
    return {
      src: { default: null },
      width: { default: "100%" },
      height: { default: 400 },
      frameborder: { default: 0 },
      allowfullscreen: { default: true },
    };
  },

  parseHTML() {
    return [{ tag: "iframe" }];
  },

  renderHTML({ HTMLAttributes }) {
    return ["div", { class: "iframe-wrapper" }, ["iframe", mergeAttributes(HTMLAttributes)]];
  },

  addCommands() {
    return {
      setIframe: (options: { src: string }) => ({ commands }: any) => {
        return commands.insertContent({ type: this.name, attrs: options });
      },
    } as any;
  },
});
