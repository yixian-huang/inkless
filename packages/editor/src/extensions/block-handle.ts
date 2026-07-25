import { Extension } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import type { EditorView } from "@tiptap/pm/view";
import type { Node as PmNode } from "@tiptap/pm/model";

const pluginKey = new PluginKey("blockHandle");

/** Resolve a top-level block position from a mouse event */
function resolveBlockPos(view: EditorView, event: MouseEvent): { pos: number; node: PmNode; dom: HTMLElement } | null {
  const posInfo = view.posAtCoords({ left: event.clientX, top: event.clientY });
  if (!posInfo) return null;

  const resolved = view.state.doc.resolve(posInfo.pos);
  let depth = resolved.depth;
  while (depth > 1) depth--;

  if (depth < 1) return null;

  const pos = resolved.before(depth);
  const node = view.state.doc.nodeAt(pos);
  if (!node) return null;

  const dom = view.nodeDOM(pos);
  if (!dom || !(dom instanceof HTMLElement)) return null;

  return { pos, node, dom };
}

export const BlockHandle = Extension.create({
  name: "blockHandle",

  addOptions() {
    return {
      dropCursorColor: "#3b82f6",
    };
  },

  addProseMirrorPlugins() {
    const editor = this.editor;
    let handleContainer: HTMLDivElement | null = null;
    let currentBlockPos: number | null = null;
    let menuEl: HTMLDivElement | null = null;
    // Drag state
    let dragSourcePos: number | null = null;
    let dragSourceDom: HTMLElement | null = null;

    const removeMenu = () => {
      if (menuEl) {
        menuEl.remove();
        menuEl = null;
      }
    };

    const removeHandles = () => {
      if (handleContainer) {
        handleContainer.remove();
        handleContainer = null;
      }
      currentBlockPos = null;
      removeMenu();
    };

    const showHandles = (view: EditorView, pos: number, node: PmNode, dom: HTMLElement) => {
      if (currentBlockPos === pos && handleContainer) return;

      removeHandles();
      currentBlockPos = pos;

      handleContainer = document.createElement("div");
      handleContainer.className = "block-handle-container";
      handleContainer.contentEditable = "false";
      handleContainer.style.cssText = [
        "position: absolute",
        "display: flex",
        "align-items: center",
        "gap: 2px",
        "z-index: 20",
        "user-select: none",
      ].join(";");

      // + button
      const plusBtn = document.createElement("button");
      plusBtn.className = "block-handle-btn block-handle-plus";
      plusBtn.type = "button";
      plusBtn.innerHTML = "<svg width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><line x1='12' y1='5' x2='12' y2='19'/><line x1='5' y1='12' x2='19' y2='12'/></svg>";
      plusBtn.title = "添加内容块";

      plusBtn.addEventListener("mousedown", (e) => {
        e.preventDefault();
        e.stopPropagation();
        const endPos = pos + node.nodeSize;
        editor.chain().focus().insertContentAt(endPos, { type: "paragraph", content: [{ type: "text", text: "/" }] }).setTextSelection(endPos + 2).run();
      });

      // Grip button — now draggable
      const gripBtn = document.createElement("button");
      gripBtn.className = "block-handle-btn block-handle-grip";
      gripBtn.type = "button";
      gripBtn.draggable = true;
      gripBtn.innerHTML = "<svg width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><circle cx='9' cy='5' r='1.5' fill='currentColor'/><circle cx='15' cy='5' r='1.5' fill='currentColor'/><circle cx='9' cy='12' r='1.5' fill='currentColor'/><circle cx='15' cy='12' r='1.5' fill='currentColor'/><circle cx='9' cy='19' r='1.5' fill='currentColor'/><circle cx='15' cy='19' r='1.5' fill='currentColor'/></svg>";
      gripBtn.title = "拖拽排序 / 点击操作";

      // Click: open context menu
      gripBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        toggleMenu(pos);
      });

      // Drag start: serialize current block
      gripBtn.addEventListener("dragstart", (e) => {
        e.stopPropagation();
        removeMenu();

        const currentNode = editor.state.doc.nodeAt(pos);
        if (!currentNode || !e.dataTransfer) {
          e.preventDefault();
          return;
        }

        dragSourcePos = pos;
        dragSourceDom = dom;

        // Create a slice from the node
        const slice = editor.state.doc.slice(pos, pos + currentNode.nodeSize);
        const serialized = JSON.stringify({
          type: "block-handle-drag",
          pos,
          nodeSize: currentNode.nodeSize,
        });
        e.dataTransfer.setData("application/x-block-handle", serialized);
        e.dataTransfer.effectAllowed = "move";

        // Store slice on plugin state for the drop handler
        (view as any).__blockHandleDragSlice = slice;

        // Visual feedback: fade source block
        dom.classList.add("block-handle-dragging");
      });

      gripBtn.addEventListener("dragend", () => {
        // Clean up drag state
        if (dragSourceDom) {
          dragSourceDom.classList.remove("block-handle-dragging");
        }
        dragSourcePos = null;
        dragSourceDom = null;
        (view as any).__blockHandleDragSlice = undefined;
      });

      handleContainer.appendChild(plusBtn);
      handleContainer.appendChild(gripBtn);

      // Position to the left of the block
      const editorParent = view.dom.parentElement;
      const editorRect = view.dom.getBoundingClientRect();
      const blockRect = dom.getBoundingClientRect();
      const scrollTop = editorParent?.scrollTop || 0;

      handleContainer.style.top = `${blockRect.top - editorRect.top + scrollTop}px`;
      handleContainer.style.left = "-44px";
      handleContainer.style.height = `${Math.min(blockRect.height, 28)}px`;

      if (editorParent) {
        if (getComputedStyle(editorParent).position === "static") {
          editorParent.style.position = "relative";
        }
        editorParent.appendChild(handleContainer);
      }
    };

    const toggleMenu = (pos: number) => {
      if (menuEl) {
        removeMenu();
        return;
      }

      menuEl = document.createElement("div");
      menuEl.className = "block-handle-menu";
      menuEl.contentEditable = "false";

      const items = [
        {
          label: "转化为...",
          action: () => {
            removeMenu();
            const currentNode = editor.state.doc.nodeAt(pos);
            if (!currentNode) return;
            editor.chain().focus()
              .setNodeSelection(pos)
              .deleteSelection()
              .insertContentAt(pos, { type: "paragraph", content: [{ type: "text", text: "/" }] })
              .setTextSelection(pos + 2).run();
          },
        },
        {
          label: "复制",
          action: () => {
            removeMenu();
            editor.chain().focus().setNodeSelection(pos).run();
            document.execCommand("copy");
          },
        },
        {
          label: "删除",
          action: () => {
            removeMenu();
            editor.chain().focus().setNodeSelection(pos).deleteSelection().run();
          },
          danger: true,
        },
      ];

      items.forEach((item) => {
        const btn = document.createElement("button");
        btn.className = `block-handle-menu-item${(item as any).danger ? " block-handle-menu-danger" : ""}`;
        btn.type = "button";
        btn.textContent = item.label;
        btn.addEventListener("mousedown", (e) => {
          e.preventDefault();
          e.stopPropagation();
          item.action();
        });
        menuEl!.appendChild(btn);
      });

      if (handleContainer) {
        menuEl.style.cssText = [
          "position: absolute",
          "top: 100%",
          "left: 0",
          "margin-top: 6px",
        ].join(";");
        handleContainer.appendChild(menuEl);
      }
    };

    // Close menu on outside click
    const onDocClick = (e: MouseEvent) => {
      if (menuEl && !menuEl.contains(e.target as Node)) {
        removeMenu();
      }
    };

    return [
      new Plugin({
        key: pluginKey,

        props: {
          handleDOMEvents: {
            mousemove(view, event) {
              if (menuEl) return false;
              // Don't show handles during drag
              if (dragSourcePos !== null) return false;

              const result = resolveBlockPos(view, event);
              if (!result) {
                const editorRect = view.dom.getBoundingClientRect();
                if (event.clientX < editorRect.left - 60 || event.clientX > editorRect.right + 20 ||
                    event.clientY < editorRect.top - 10 || event.clientY > editorRect.bottom + 10) {
                  removeHandles();
                }
                return false;
              }

              showHandles(view, result.pos, result.node, result.dom);
              return false;
            },

            // Handle drop for block handle drag
            drop(view, event) {
              const slice = (view as any).__blockHandleDragSlice;
              if (!slice || dragSourcePos === null) return false;

              event.preventDefault();
              (view as any).__blockHandleDragSlice = undefined;

              const dropCoords = view.posAtCoords({ left: event.clientX, top: event.clientY });
              if (!dropCoords) return false;

              const dropResolved = view.state.doc.resolve(dropCoords.pos);
              let dropDepth = dropResolved.depth;
              while (dropDepth > 1) dropDepth--;
              if (dropDepth < 1) return false;

              let dropPos = dropResolved.before(dropDepth);
              const sourceNode = view.state.doc.nodeAt(dragSourcePos);
              if (!sourceNode) return false;

              const sourceEnd = dragSourcePos + sourceNode.nodeSize;

              // Determine if we should drop before or after the target block
              const dropNode = view.state.doc.nodeAt(dropPos);
              if (dropNode) {
                const dropDom = view.nodeDOM(dropPos);
                if (dropDom instanceof HTMLElement) {
                  const dropRect = dropDom.getBoundingClientRect();
                  const midY = dropRect.top + dropRect.height / 2;
                  if (event.clientY > midY) {
                    dropPos = dropPos + dropNode.nodeSize;
                  }
                }
              }

              // Don't move to same position
              if (dropPos >= dragSourcePos && dropPos <= sourceEnd) return false;

              // Execute the move as a single transaction
              const tr = view.state.tr;
              const nodeToMove = sourceNode.copy(sourceNode.content);

              if (dropPos > dragSourcePos) {
                // Moving down: delete first, then insert at adjusted position
                tr.delete(dragSourcePos, sourceEnd);
                const adjustedPos = dropPos - sourceNode.nodeSize;
                tr.insert(adjustedPos, nodeToMove);
              } else {
                // Moving up: insert first, then delete at adjusted position
                tr.insert(dropPos, nodeToMove);
                const adjustedSourcePos = dragSourcePos + sourceNode.nodeSize;
                tr.delete(adjustedSourcePos, adjustedSourcePos + sourceNode.nodeSize);
              }

              view.dispatch(tr);

              // Clean up
              if (dragSourceDom) {
                dragSourceDom.classList.remove("block-handle-dragging");
              }
              dragSourcePos = null;
              dragSourceDom = null;
              removeHandles();

              return true;
            },

            dragover(_view, event) {
              if (dragSourcePos !== null) {
                event.preventDefault();
                if (event.dataTransfer) {
                  event.dataTransfer.dropEffect = "move";
                }
              }
              return false;
            },
          },
        },

        view() {
          document.addEventListener("mousedown", onDocClick);
          return {
            update(view) {
              if (currentBlockPos === null || !handleContainer) return;
              const node = view.state.doc.nodeAt(currentBlockPos);
              if (!node) {
                removeHandles();
                return;
              }
              const dom = view.nodeDOM(currentBlockPos);
              if (!dom || !(dom instanceof HTMLElement)) {
                removeHandles();
                return;
              }
              const editorRect = view.dom.getBoundingClientRect();
              const blockRect = dom.getBoundingClientRect();
              const scrollTop = view.dom.parentElement?.scrollTop || 0;
              handleContainer.style.top = `${blockRect.top - editorRect.top + scrollTop}px`;
              handleContainer.style.height = `${Math.min(blockRect.height, 28)}px`;
            },
            destroy() {
              removeHandles();
              document.removeEventListener("mousedown", onDocClick);
            },
          };
        },
      }),
    ];
  },
});
