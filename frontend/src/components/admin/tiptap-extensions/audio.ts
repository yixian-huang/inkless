import { Node, mergeAttributes } from "@tiptap/core";

export const Audio = Node.create({
  name: "audio",
  group: "block",
  atom: true,

  addAttributes() {
    return {
      src: { default: null },
      controls: { default: true },
    };
  },

  parseHTML() {
    return [{ tag: "audio" }];
  },

  renderHTML({ HTMLAttributes }) {
    return ["div", { class: "audio-wrapper" }, ["audio", mergeAttributes(HTMLAttributes)]];
  },

  addCommands() {
    return {
      setAudio: (options: { src: string }) => ({ commands }: any) => {
        return commands.insertContent({ type: this.name, attrs: { src: options.src, controls: true } });
      },
    } as any;
  },
});
