import { useLayoutEffect, useRef, type RefObject } from "react";

interface ArticlePostBodyProps {
  html: string;
  contentRef?: RefObject<HTMLElement | null>;
  onClick?: (e: React.MouseEvent) => void;
}

let mermaidReady: Promise<typeof import("mermaid").default> | null = null;
let mermaidSeq = 0;

function loadMermaid() {
  if (!mermaidReady) {
    mermaidReady = import("mermaid").then((mod) => {
      const mermaid = mod.default;
      mermaid.initialize({
        startOnLoad: false,
        securityLevel: "loose",
        theme: "neutral",
        fontFamily: "inherit",
      });
      return mermaid;
    });
  }
  return mermaidReady;
}

/** Collect mermaid source nodes, including code fences that were not rewritten. */
function collectMermaidNodes(root: HTMLElement): HTMLElement[] {
  const out: HTMLElement[] = [];

  root.querySelectorAll<HTMLElement>(".mermaid, [data-type='mermaid']").forEach((el) => {
    // Skip nodes already turned into SVG-only containers without source
    out.push(el);
  });

  root.querySelectorAll<HTMLElement>("pre > code.language-mermaid, pre > code.lang-mermaid").forEach((code) => {
    const pre = code.parentElement as HTMLElement | null;
    if (!pre || pre.closest(".mermaid")) return;
    // Promote fence to a mermaid div so the renderer has a clean target
    const div = document.createElement("div");
    div.className = "mermaid";
    div.setAttribute("data-type", "mermaid");
    div.textContent = code.textContent ?? "";
    pre.replaceWith(div);
    out.push(div);
  });

  return out;
}

function sourceOf(node: HTMLElement): string {
  return (
    node.getAttribute("data-mermaid-source") ||
    node.getAttribute("data-source") ||
    node.textContent ||
    ""
  ).trim();
}

async function renderMermaidNodes(nodes: HTMLElement[]) {
  if (nodes.length === 0) return;
  const mermaid = await loadMermaid();

  for (const node of nodes) {
    const source = sourceOf(node);
    if (!source) continue;

    // Preserve source so re-renders / React updates can recover
    node.setAttribute("data-mermaid-source", source);
    node.setAttribute("data-type", "mermaid");
    node.classList.add("mermaid");

    // Reset previous mermaid mutations
    node.removeAttribute("data-processed");
    node.innerHTML = source;

    try {
      // Prefer run() for batch; fall back to render() for stubborn graphs
      await mermaid.run({ nodes: [node], suppressErrors: false });
    } catch {
      try {
        const id = `mermaid-pub-${++mermaidSeq}`;
        const { svg } = await mermaid.render(id, source);
        node.innerHTML = svg;
        node.setAttribute("data-processed", "true");
      } catch (err) {
        console.warn("Mermaid render failed:", err);
        node.innerHTML = `<pre class="mermaid-error">${source.replace(/</g, "&lt;")}</pre>`;
      }
    }
  }
}

/** Article body HTML — must sit inside `ArticleTypographyRoot`. */
export default function ArticlePostBody({ html, contentRef, onClick }: ArticlePostBodyProps) {
  const localRef = useRef<HTMLElement | null>(null);
  const renderGen = useRef(0);

  // useLayoutEffect: run after DOM commit (dangerouslySetInnerHTML applied) so
  // mermaid nodes exist before we try to render them.
  useLayoutEffect(() => {
    const el = contentRef?.current ?? localRef.current;
    if (!el || !html) return;

    const gen = ++renderGen.current;
    let cancelled = false;

    // Double rAF: ensure nested layout (TOC ids) has settled
    const raf = requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        if (cancelled || gen !== renderGen.current) return;
        const nodes = collectMermaidNodes(el);
        void renderMermaidNodes(nodes).catch((err) => {
          if (!cancelled) console.warn("Mermaid batch failed:", err);
        });
      });
    });

    return () => {
      cancelled = true;
      cancelAnimationFrame(raf);
    };
  }, [html, contentRef]);

  return (
    <article
      ref={(node) => {
        localRef.current = node;
        if (contentRef) {
          (contentRef as React.MutableRefObject<HTMLElement | null>).current = node;
        }
      }}
      className="tiptap ProseMirror max-w-none article-public-view"
      dangerouslySetInnerHTML={{ __html: html }}
      onClick={onClick}
    />
  );
}
