/** Count CJK chars as words + latin tokens for editor tab badges. */
export function countWords(text: string): { chars: number; words: number } {
  const cleaned = text.replace(/\s+/g, " ").trim();
  if (!cleaned) return { chars: 0, words: 0 };
  const chars = cleaned.replace(/\s/g, "").length;
  const cjk = (cleaned.match(/[\u4e00-\u9fff\u3400-\u4dbf\uf900-\ufaff]/g) || []).length;
  const latin = cleaned
    .replace(/[\u4e00-\u9fff\u3400-\u4dbf\uf900-\ufaff]/g, " ")
    .trim()
    .split(/\s+/)
    .filter(Boolean).length;
  return { chars, words: cjk + latin };
}

export function htmlToPlainText(html: string): string {
  if (!html) return "";
  if (typeof document === "undefined") {
    return html
      .replace(/<br\s*\/?>/gi, "\n")
      .replace(/<\/p>/gi, "\n")
      .replace(/<[^>]+>/g, "")
      .replace(/&nbsp;/g, " ")
      .trim();
  }
  const el = document.createElement("div");
  el.innerHTML = html;
  return (el.textContent || "").trim();
}

/** Wrap plain translated paragraphs as simple HTML for TipTap. */
export function plainTextToHtml(text: string): string {
  const parts = text
    .split(/\n{2,}/)
    .map((p) => p.trim())
    .filter(Boolean);
  if (parts.length === 0) return "<p></p>";
  return parts.map((p) => `<p>${escapeHtml(p).replace(/\n/g, "<br>")}</p>`).join("");
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
