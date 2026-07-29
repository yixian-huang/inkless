import { Extension } from "@tiptap/core";
import { Plugin, PluginKey, NodeSelection } from "@tiptap/pm/state";
import type { EditorView } from "@tiptap/pm/view";

const pluginKey = new PluginKey("blockToolbar");

/** Block-level node types that should show a hover toolbar */
const BLOCK_TYPES = new Set([
  "columns",
  "imageGallery",
  "iframe",
  "video",
  "audio",
  "details",
]);

function findBlockNode(
  view: EditorView,
  dom: HTMLElement,
): { pos: number; node: any; dom: HTMLElement } | null {
  // Walk up from target to find a block node we manage
  let current: HTMLElement | null = dom;
  while (current && current !== view.dom) {
    let pos: number;
    try {
      pos = view.posAtDOM(current, 0);
    } catch {
      current = current.parentElement;
      continue;
    }
    // Resolve to check node type
    const $pos = view.state.doc.resolve(pos);
    for (let d = $pos.depth; d >= 1; d--) {
      const node = $pos.node(d);
      if (BLOCK_TYPES.has(node.type.name)) {
        const nodePos = $pos.before(d);
        const nodeDom = view.nodeDOM(nodePos);
        if (nodeDom && nodeDom instanceof HTMLElement) {
          return { pos: nodePos, node, dom: nodeDom };
        }
      }
    }
    current = current.parentElement;
  }
  return null;
}

/** Suppress ProseMirror's DOM observer while mutating editor DOM. */
function withoutDomObserver(view: EditorView, fn: () => void) {
  const obs = (view as any).domObserver;
  if (obs) { obs.stop(); try { fn(); } finally { obs.start(); } }
  else fn();
}

export const BlockToolbar = Extension.create({
  name: "blockToolbar",

  addProseMirrorPlugins() {
    const editor = this.editor;
    let toolbar: HTMLDivElement | null = null;
    let currentPos: number | null = null;
    let currentDom: HTMLElement | null = null;

    const removeToolbar = (view?: EditorView) => {
      const doRemove = () => {
        if (toolbar) {
          toolbar.remove();
          toolbar = null;
        }
        if (currentDom) {
          currentDom.classList.remove("block-toolbar-active");
          currentDom = null;
        }
        currentPos = null;
      };
      if (view) withoutDomObserver(view, doRemove);
      else doRemove();
    };

    const showToolbar = (view: EditorView, pos: number, node: any, dom: HTMLElement) => {
      if (currentPos === pos && toolbar) return;
      removeToolbar(view);

      withoutDomObserver(view, () => {
        currentPos = pos;
        currentDom = dom;
        dom.classList.add("block-toolbar-active");

        toolbar = document.createElement("div");
        toolbar.className = "block-toolbar";
        toolbar.contentEditable = "false";

        // Select button
        const selectBtn = document.createElement("button");
        selectBtn.type = "button";
        selectBtn.className = "block-toolbar-btn";
        selectBtn.title = "选中";
        selectBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/></svg>`;
        selectBtn.addEventListener("mousedown", (e) => {
          e.preventDefault();
          e.stopPropagation();
          editor.chain().focus().setNodeSelection(pos).run();
        });

        // Delete button
        const deleteBtn = document.createElement("button");
        deleteBtn.type = "button";
        deleteBtn.className = "block-toolbar-btn block-toolbar-btn-danger";
        deleteBtn.title = "删除";
        deleteBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`;
        deleteBtn.addEventListener("mousedown", (e) => {
          e.preventDefault();
          e.stopPropagation();
          editor.chain().focus().setNodeSelection(pos).deleteSelection().run();
          removeToolbar();
        });

        toolbar.appendChild(selectBtn);
        toolbar.appendChild(deleteBtn);

        // Position: top-right of the block dom
        dom.style.position = "relative";
        dom.appendChild(toolbar);
      });
    };

    return [
      new Plugin({
        key: pluginKey,

        props: {
          handleDOMEvents: {
            mouseover(view, event) {
              const target = event.target as HTMLElement;
              const result = findBlockNode(view, target);
              if (!result) return false;
              showToolbar(view, result.pos, result.node, result.dom);
              return false;
            },
          },

          // Click on atom nodes to select them
          handleClickOn(view, pos, node) {
            if (node.isAtom && BLOCK_TYPES.has(node.type.name)) {
              // Ensure NodeSelection is created
              const tr = view.state.tr.setSelection(NodeSelection.create(view.state.doc, pos));
              view.dispatch(tr);
              return true;
            }
            return false;
          },
        },

        view() {
          const onMouseLeave = (event: MouseEvent) => {
            // Check if we're leaving the editor area
            const related = event.relatedTarget as HTMLElement | null;
            if (toolbar && related && toolbar.contains(related)) return;
            if (currentDom && related && currentDom.contains(related)) return;

            // Delay removal to allow moving to toolbar
            setTimeout(() => {
              if (toolbar && !toolbar.matches(":hover") && currentDom && !currentDom.matches(":hover")) {
                removeToolbar();
              }
            }, 100);
          };

          return {
            update(view) {
              withoutDomObserver(view, () => {
                // Remove toolbar if the node no longer exists at position
                if (currentPos !== null && toolbar) {
                  const node = view.state.doc.nodeAt(currentPos);
                  if (!node || !BLOCK_TYPES.has(node.type.name)) {
                    removeToolbar();
                  }
                }

                // Show selection outline for NodeSelection on managed types
                const { selection } = view.state;
                if (selection instanceof NodeSelection) {
                  const node = view.state.doc.nodeAt(selection.from);
                  if (node && BLOCK_TYPES.has(node.type.name)) {
                    const dom = view.nodeDOM(selection.from) as HTMLElement | null;
                    if (dom) {
                      dom.classList.add("block-node-selected");
                    }
                  }
                }

                // Clean up selection class from previously selected nodes
                const prevSelected = view.dom.querySelectorAll(".block-node-selected");
                prevSelected.forEach((el) => {
                  if (selection instanceof NodeSelection) {
                    const selDom = view.nodeDOM(selection.from);
                    if (el !== selDom) el.classList.remove("block-node-selected");
                  } else {
                    el.classList.remove("block-node-selected");
                  }
                });
              });
            },
            destroy() {
              removeToolbar();
              document.removeEventListener("mouseleave", onMouseLeave);
            },
          };
        },
      }),
    ];
  },

  addKeyboardShortcuts() {
    return {
      // Delete/Backspace on NodeSelection of managed block types
      Delete: ({ editor }) => {
        const { selection } = editor.state;
        if (selection instanceof NodeSelection && BLOCK_TYPES.has(selection.node.type.name)) {
          editor.chain().deleteSelection().run();
          return true;
        }
        return false;
      },
      Backspace: ({ editor }) => {
        const { selection } = editor.state;
        if (selection instanceof NodeSelection && BLOCK_TYPES.has(selection.node.type.name)) {
          editor.chain().deleteSelection().run();
          return true;
        }
        return false;
      },
    };
  },
});
