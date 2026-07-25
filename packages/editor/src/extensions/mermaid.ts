import { Node, mergeAttributes } from "@tiptap/core";

export interface MermaidOptions {
  HTMLAttributes: Record<string, unknown>;
}

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    mermaid: {
      /** Insert a mermaid diagram block with the given source. */
      setMermaid: (options: { source: string }) => ReturnType;
    };
  }
}

/**
 * TipTap block for Mermaid diagrams.
 * Serializes to `<div class="mermaid" data-type="mermaid">source</div>` so
 * public pages and markdown round-trip share the same HTML shape.
 */
export const Mermaid = Node.create<MermaidOptions>({
  name: "mermaid",
  group: "block",
  atom: true,
  draggable: true,
  selectable: true,

  addOptions() {
    return {
      HTMLAttributes: {},
    };
  },

  addAttributes() {
    return {
      source: {
        default: "",
        parseHTML: (element) => {
          const fromAttr =
            element.getAttribute("data-source") ||
            element.getAttribute("data-mermaid-source");
          if (fromAttr) return fromAttr;
          return (element.textContent || "").trim();
        },
        renderHTML: (attributes) => {
          // Prefer text content for public mermaid.run(); also keep data-source.
          return {
            "data-source": attributes.source || "",
          };
        },
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: 'div[data-type="mermaid"]',
        priority: 60,
      },
      {
        tag: "div.mermaid",
        priority: 60,
      },
      {
        // Prefer Mermaid node over CodeBlockLowlight for language-mermaid fences
        tag: "pre",
        priority: 60,
        getAttrs: (node) => {
          const el = node as HTMLElement;
          const code = el.querySelector("code");
          const cls = code?.getAttribute("class") || "";
          if (!/language-mermaid|lang-mermaid/.test(cls)) return false;
          return { source: (code?.textContent || "").trim() };
        },
      },
    ];
  },

  renderHTML({ node, HTMLAttributes }) {
    const source = (node.attrs.source as string) || "";
    return [
      "div",
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes, {
        class: "mermaid",
        "data-type": "mermaid",
      }),
      source,
    ];
  },

  addCommands() {
    return {
      setMermaid:
        (options: { source: string }) =>
        ({ commands }) =>
          commands.insertContent({
            type: this.name,
            attrs: { source: options.source },
          }),
    };
  },

  addNodeView() {
    return ({ node }) => {
      const dom = document.createElement("div");
      dom.className = "mermaid-node-view";
      dom.setAttribute("data-type", "mermaid");
      dom.contentEditable = "false";

      const badge = document.createElement("div");
      badge.className = "mermaid-node-view__badge";
      badge.textContent = "Mermaid";

      const sourceEl = document.createElement("pre");
      sourceEl.className = "mermaid-node-view__source";
      sourceEl.textContent = node.attrs.source || "";

      const preview = document.createElement("div");
      preview.className = "mermaid mermaid-node-view__preview";
      preview.textContent = node.attrs.source || "";

      dom.appendChild(badge);
      dom.appendChild(sourceEl);
      dom.appendChild(preview);

      // Best-effort live render in the editor (non-blocking).
      void import("mermaid")
        .then((mod) => {
          const mermaid = mod.default;
          mermaid.initialize({
            startOnLoad: false,
            securityLevel: "strict",
            theme: "neutral",
            fontFamily: "inherit",
          });
          return mermaid.run({ nodes: [preview], suppressErrors: true });
        })
        .catch(() => {
          /* ignore preview failures in editor */
        });

      return {
        dom,
        update: (updatedNode) => {
          if (updatedNode.type.name !== "mermaid") return false;
          const next = (updatedNode.attrs.source as string) || "";
          if (sourceEl.textContent !== next) {
            sourceEl.textContent = next;
            preview.textContent = next;
            preview.removeAttribute("data-processed");
            void import("mermaid")
              .then((mod) =>
                mod.default.run({ nodes: [preview], suppressErrors: true }),
              )
              .catch(() => undefined);
          }
          return true;
        },
        ignoreMutation: () => true,
      };
    };
  },
});
