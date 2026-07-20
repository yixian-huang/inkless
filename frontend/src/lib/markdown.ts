import { marked } from "marked";

/** Escape HTML special characters for safe embedding in attributes/text. */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

let configured = false;

function ensureMarkedConfig() {
  if (configured) return;
  configured = true;

  const renderer = new marked.Renderer();
  const originalCode = renderer.code.bind(renderer);

  renderer.code = (token: { text: string; lang?: string }) => {
    const lang = (token.lang || "").trim().split(/\s+/)[0]?.toLowerCase() ?? "";
    if (lang === "mermaid") {
      // Preserve source for client-side mermaid rendering (editor + public).
      return `<div class="mermaid">${escapeHtml(token.text)}</div>\n`;
    }
    return originalCode(token as never);
  };

  marked.setOptions({
    gfm: true,
    breaks: false,
  });
  marked.use({ renderer });
}

/**
 * Convert Markdown to HTML. Mermaid fences become `<div class="mermaid">…</div>`.
 */
export function markdownToHtml(source: string): string {
  ensureMarkedConfig();
  return marked.parse(source ?? "", { async: false }) as string;
}

/**
 * Rough HTML → Markdown for mode switch. Prefer turndown when available;
 * this is only a fallback helper for plain text recovery.
 */
export function isMermaidHtml(html: string): boolean {
  return /class=["']mermaid["']/.test(html);
}
