import { useEffect, useId, useRef } from "react";

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

/**
 * Renders markdown HTML and runs mermaid on `.mermaid` blocks.
 */
export default function MarkdownHtmlPreview({
  html,
  className = "",
}: {
  html: string;
  className?: string;
}) {
  const rootRef = useRef<HTMLDivElement>(null);
  const reactId = useId();
  const renderGen = useRef(0);

  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;
    root.innerHTML = html || '<p class="text-gray-400 text-sm italic">预览将显示在这里…</p>';

    const nodes = Array.from(root.querySelectorAll<HTMLElement>(".mermaid"));
    if (nodes.length === 0) return;

    const gen = ++renderGen.current;
    let cancelled = false;

    void (async () => {
      try {
        const mermaid = await loadMermaid();
        if (cancelled || gen !== renderGen.current) return;

        for (const node of nodes) {
          // Capture original source before mermaid mutates the node.
          if (!node.getAttribute("data-mermaid-source")) {
            node.setAttribute("data-mermaid-source", node.textContent ?? "");
          }
        }

        await mermaid.run({ nodes, suppressErrors: true });
      } catch (err) {
        if (!cancelled) {
          console.warn("Mermaid render failed:", err);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [html]);

  return (
    <div
      ref={rootRef}
      data-preview-id={reactId}
      className={className}
    />
  );
}
