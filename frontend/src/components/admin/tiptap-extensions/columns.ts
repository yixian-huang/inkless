import { Node, mergeAttributes } from "@tiptap/core";

export const Column = Node.create({
  name: "column",
  group: "columns",
  content: "block+",
  isolating: true,

  parseHTML() {
    return [{ tag: 'div[data-type="column"]' }];
  },

  renderHTML({ HTMLAttributes }) {
    return ["div", mergeAttributes(HTMLAttributes, { "data-type": "column", class: "column" }), 0];
  },
});

export const Columns = Node.create({
  name: "columns",
  group: "block",
  content: "column{2,4}",
  isolating: true,

  addAttributes() {
    return {
      count: { default: 2 },
    };
  },

  parseHTML() {
    return [{ tag: 'div[data-type="columns"]' }];
  },

  renderHTML({ HTMLAttributes }) {
    const { count, ...rest } = HTMLAttributes;
    return [
      "div",
      mergeAttributes(rest, { "data-type": "columns", class: `columns cols-${count || 2}` }),
      0,
    ];
  },

  addCommands() {
    return {
      setColumns: (count: number = 2) => ({ commands }: any) => {
        const cols = Array.from({ length: count }, () => ({
          type: "column",
          content: [{ type: "paragraph" }],
        }));
        return commands.insertContent({ type: this.name, attrs: { count }, content: cols });
      },
    } as any;
  },
});
