import { useEffect, useRef, type RefObject } from "react";

interface ArticlePostBodyProps {
  html: string;
  contentRef?: RefObject<HTMLElement | null>;
  onClick?: (e: React.MouseEvent) => void;
}

let mermaidReady: Promise<typeof import("mermaid").default> | null = null;

function loadMermaid() {
  if (!mermaidReady) {
    mermaidReady = import("mermaid").then((mod) => {
      const mermaid = mod.default;
      mermaid.initialize({
        startOnLoad: false,
        securityLevel: "strict",
        theme: "neutral",
        fontFamily: "inherit",
      });
      return mermaid;
    });
  }
  return mermaidReady;
}

/** Article body HTML — must sit inside `ArticleTypographyRoot`. */
export default function ArticlePostBody({ html, contentRef, onClick }: ArticlePostBodyProps) {
  const localRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    const el = contentRef?.current ?? localRef.current;
    if (!el) return;
    const nodes = Array.from(el.querySelectorAll<HTMLElement>(".mermaid"));
    if (nodes.length === 0) return;

    let cancelled = false;
    void (async () => {
      try {
        const mermaid = await loadMermaid();
        if (cancelled) return;
        for (const node of nodes) {
          if (!node.getAttribute("data-mermaid-source")) {
            node.setAttribute("data-mermaid-source", node.textContent ?? "");
          }
        }
        await mermaid.run({ nodes, suppressErrors: true });
      } catch (err) {
        if (!cancelled) console.warn("Mermaid render failed:", err);
      }
    })();
    return () => {
      cancelled = true;
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
