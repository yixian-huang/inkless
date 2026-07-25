import { useEffect, useId, useRef } from "react";

let mermaidReady: Promise<typeof import("mermaid").default> | null = null;

/** Cache rendered SVG HTML by diagram source to avoid re-running mermaid on every keystroke debounce. */
const mermaidSvgCache = new Map<string, string>();
const MERMAID_CACHE_MAX = 40;

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

function cachePut(source: string, svgHtml: string) {
  if (mermaidSvgCache.size >= MERMAID_CACHE_MAX) {
    const first = mermaidSvgCache.keys().next().value;
    if (first != null) mermaidSvgCache.delete(first);
  }
  mermaidSvgCache.set(source, svgHtml);
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

        const toRun: HTMLElement[] = [];
        for (const node of nodes) {
          const source =
            node.getAttribute("data-mermaid-source") || (node.textContent ?? "");
          if (!node.getAttribute("data-mermaid-source")) {
            node.setAttribute("data-mermaid-source", source);
          }
          const cached = mermaidSvgCache.get(source);
          if (cached) {
            node.innerHTML = cached;
            node.removeAttribute("data-processed");
          } else {
            toRun.push(node);
          }
        }

        if (toRun.length === 0) return;

        await mermaid.run({ nodes: toRun, suppressErrors: true });
        if (cancelled || gen !== renderGen.current) return;

        for (const node of toRun) {
          const source = node.getAttribute("data-mermaid-source") || "";
          if (source && node.innerHTML) {
            cachePut(source, node.innerHTML);
          }
        }
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
