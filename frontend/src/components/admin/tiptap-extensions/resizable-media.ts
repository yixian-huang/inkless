import { Extension } from "@tiptap/core";
import { Plugin, PluginKey, NodeSelection } from "@tiptap/pm/state";
import type { EditorView } from "@tiptap/pm/view";

const MIN_WIDTH = 50;
const pluginKey = new PluginKey("resizableMedia");

function isMediaNode(node: any): boolean {
  return node && (node.type.name === "image" || node.type.name === "video");
}

/** Add the single SE resize handle + blue border to a selected media node */
function attachSelectionUI(view: EditorView, pos: number, dom: HTMLElement): () => void {
  // <img> is a void element — cannot appendChild. Use parent as container.
  const isVoid = dom.tagName === "IMG";
  const container = (isVoid ? dom.parentElement : dom) as HTMLElement;
  if (!container) return () => {};

  const origContainerPos = container.style.position;
  const origOutline = dom.style.outline;
  const origOutlineOffset = dom.style.outlineOffset;
  const origBorderRadius = dom.style.borderRadius;

  if (!origContainerPos || origContainerPos === "static") {
    container.style.position = "relative";
  }

  // Light blue border on the media element itself
  dom.style.outline = "2px solid #93c5fd";
  dom.style.outlineOffset = "3px";
  dom.style.borderRadius = "4px";

  // Single bottom-right handle — light blue circle
  const handle = document.createElement("div");
  handle.className = "media-resize-se";
  handle.contentEditable = "false";
  handle.style.cssText = [
    "position: absolute",
    "bottom: -5px",
    "right: -5px",
    "width: 10px",
    "height: 10px",
    "background: #60a5fa",
    "border: 2px solid white",
    "border-radius: 50%",
    "cursor: nwse-resize",
    "z-index: 10",
    "box-shadow: 0 1px 3px rgba(0,0,0,0.2)",
  ].join(";");

  container.appendChild(handle);

  // Drag resize
  handle.addEventListener("mousedown", (event: MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();

    const startX = event.clientX;
    const startWidth = dom.offsetWidth;
    const parentWidth = dom.parentElement?.offsetWidth || startWidth;

    const overlay = document.createElement("div");
    overlay.style.cssText = "position:fixed;inset:0;z-index:99999;cursor:nwse-resize;";
    document.body.appendChild(overlay);

    const onMove = (e: MouseEvent) => {
      const dx = e.clientX - startX;
      const newW = Math.max(MIN_WIDTH, Math.min(startWidth + dx, parentWidth));
      dom.style.width = `${newW}px`;
      dom.style.height = "auto";
    };

    const onUp = (e: MouseEvent) => {
      overlay.remove();
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);

      const dx = e.clientX - startX;
      const newW = Math.max(MIN_WIDTH, Math.min(startWidth + dx, parentWidth));
      const pct = Math.round((newW / parentWidth) * 100);

      const currentNode = view.state.doc.nodeAt(pos);
      if (currentNode) {
        const tr = view.state.tr.setNodeMarkup(pos, undefined, {
          ...currentNode.attrs,
          width: `${pct}%`,
        });
        view.dispatch(tr);
      }
    };

    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  });

  return () => {
    handle.remove();
    dom.style.outline = origOutline;
    dom.style.outlineOffset = origOutlineOffset;
    dom.style.borderRadius = origBorderRadius;
    if (!origContainerPos || origContainerPos === "static") {
      container.style.position = origContainerPos;
    }
  };
}

/** Create a hover "替换" button for media nodes */
function createReplaceButton(dom: HTMLElement, nodeTypeName: string, view: EditorView, nodePos: number): HTMLButtonElement {
  const btn = document.createElement("button");
  btn.className = "media-replace-btn";
  btn.textContent = "替换";
  btn.contentEditable = "false";
  btn.style.cssText = [
    "position: absolute",
    "top: 6px",
    "right: 6px",
    "padding: 2px 10px",
    "font-size: 12px",
    "background: rgba(255,255,255,0.92)",
    "color: #374151",
    "border: 1px solid #d1d5db",
    "border-radius: 9999px",
    "cursor: pointer",
    "z-index: 10",
    "box-shadow: 0 1px 4px rgba(0,0,0,0.1)",
    "transition: background 0.15s",
    "line-height: 1.5",
    "pointer-events: auto",
  ].join(";");

  btn.addEventListener("mouseenter", () => {
    btn.style.background = "#f3f4f6";
  });
  btn.addEventListener("mouseleave", () => {
    btn.style.background = "rgba(255,255,255,0.92)";
  });
  btn.addEventListener("mousedown", (e) => {
    e.preventDefault();
    e.stopPropagation();
    // Select the media node so the replace handler can find it
    const tr = view.state.tr.setSelection(NodeSelection.create(view.state.doc, nodePos));
    view.dispatch(tr);
    document.dispatchEvent(
      new CustomEvent("editor-replace-media", { detail: { type: nodeTypeName } })
    );
  });

  return btn;
}

/** Suppress ProseMirror's DOM observer while mutating editor DOM.
 *  Without this, plugin-added DOM elements trigger MutationObserver → readDOMChange → new transaction → infinite loop. */
function withoutDomObserver(view: EditorView, fn: () => void) {
  const obs = (view as any).domObserver;
  if (obs) { obs.stop(); try { fn(); } finally { obs.start(); } }
  else fn();
}

export const ResizableMedia = Extension.create({
  name: "resizableMedia",

  addProseMirrorPlugins() {
    let selectionCleanup: (() => void) | null = null;
    let hoverTarget: HTMLElement | null = null;
    let hoverBtn: HTMLButtonElement | null = null;

    const removeHoverBtn = (view?: EditorView) => {
      const doRemove = () => {
        if (hoverBtn) {
          hoverBtn.remove();
          hoverBtn = null;
        }
        if (hoverTarget) {
          hoverTarget.style.position = hoverTarget.dataset.origPos || "";
          hoverTarget = null;
        }
      };
      if (view) withoutDomObserver(view, doRemove);
      else doRemove();
    };

    return [
      new Plugin({
        key: pluginKey,

        props: {
          handleDOMEvents: {
            mouseover(view, event) {
              const target = event.target as HTMLElement;
              const el = target.closest("img, video") as HTMLElement | null;

              if (!el) return false;

              // <img> is void — use parent as hover container
              const isVoid = el.tagName === "IMG";
              const container = (isVoid ? el.parentElement : el) as HTMLElement | null;
              if (!container || container === hoverTarget) return false;

              // Remove previous
              removeHoverBtn(view);

              // Find the ProseMirror position to determine node type
              let pos: number;
              try {
                pos = view.posAtDOM(el, 0);
              } catch {
                return false;
              }
              const node = view.state.doc.nodeAt(pos);
              if (!node || !isMediaNode(node)) return false;

              withoutDomObserver(view, () => {
                hoverTarget = container;
                const origPos = container.style.position;
                container.dataset.origPos = origPos;
                if (!origPos || origPos === "static") {
                  container.style.position = "relative";
                }

                hoverBtn = createReplaceButton(container, node.type.name, view, pos);
                container.appendChild(hoverBtn);
              });

              return false;
            },

            mouseleave(view, event) {
              const related = event.relatedTarget as HTMLElement | null;
              // Don't remove if moving within the hover container or to the button
              if (related && hoverTarget && (hoverTarget.contains(related) || related === hoverBtn)) {
                return false;
              }
              removeHoverBtn(view);
              return false;
            },
          },
        },

        view() {
          return {
            update(view) {
              withoutDomObserver(view, () => {
                if (selectionCleanup) {
                  selectionCleanup();
                  selectionCleanup = null;
                }

                const { selection } = view.state;
                if (!(selection instanceof NodeSelection)) return;

                const node = view.state.doc.nodeAt(selection.from);
                if (!node || !isMediaNode(node)) return;

                const dom = view.nodeDOM(selection.from) as HTMLElement | null;
                if (!dom) return;

                selectionCleanup = attachSelectionUI(view, selection.from, dom);
              });
            },
            destroy() {
              if (selectionCleanup) {
                selectionCleanup();
                selectionCleanup = null;
              }
              removeHoverBtn();
            },
          };
        },
      }),
    ];
  },
});
