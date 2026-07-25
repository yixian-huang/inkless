import { marked } from "marked";
import TurndownService from "turndown";

/** Escape HTML special characters for safe embedding in attributes/text. */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function unescapeHtml(text: string): string {
  return text
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&amp;/g, "&");
}

let markedConfigured = false;

function ensureMarkedConfig() {
  if (markedConfigured) return;
  markedConfigured = true;

  const renderer = new marked.Renderer();
  const originalCode = renderer.code.bind(renderer);

  renderer.code = (token: { text: string; lang?: string }) => {
    const lang = (token.lang || "").trim().split(/\s+/)[0]?.toLowerCase() ?? "";
    if (lang === "mermaid") {
      // Stable shape TipTap Mermaid node + public renderer both understand.
      return `<div class="mermaid" data-type="mermaid">${escapeHtml(token.text)}</div>\n`;
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
 * Convert Markdown to HTML.
 * - GFM tables / task lists / strikethrough via marked gfm
 * - Mermaid fences → `<div class="mermaid" data-type="mermaid">…</div>`
 */
export function markdownToHtml(source: string): string {
  ensureMarkedConfig();
  return marked.parse(source ?? "", { async: false }) as string;
}

function cellText(cell: Element): string {
  return (cell.textContent || "")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/\|/g, "\\|");
}

function tableToMarkdown(table: HTMLTableElement): string {
  const rows = Array.from(table.querySelectorAll("tr"));
  if (rows.length === 0) return "";

  const matrix = rows.map((tr) =>
    Array.from(tr.querySelectorAll("th, td")).map((cell) => cellText(cell)),
  );
  const colCount = Math.max(0, ...matrix.map((r) => r.length));
  if (colCount === 0) return "";

  const pad = (row: string[]) => {
    const next = row.slice();
    while (next.length < colCount) next.push("");
    return next;
  };

  const header = pad(matrix[0]);
  const sep = header.map(() => "---");
  const body = matrix.slice(1).map(pad);

  const lines = [
    `| ${header.join(" | ")} |`,
    `| ${sep.join(" | ")} |`,
    ...body.map((r) => `| ${r.join(" | ")} |`),
  ];
  return lines.join("\n");
}

let turndownService: TurndownService | null = null;

function getTurndown(): TurndownService {
  if (turndownService) return turndownService;

  const td = new TurndownService({
    headingStyle: "atx",
    codeBlockStyle: "fenced",
    bulletListMarker: "-",
    emDelimiter: "*",
    strongDelimiter: "**",
    fence: "```",
  });

  // Keep HTML for elements we intentionally round-trip as structured blocks.
  td.keep(["iframe", "video", "audio"]);

  // Mermaid diagram blocks (TipTap node + markdown fences)
  td.addRule("mermaid", {
    filter: (node) => {
      if (!(node instanceof HTMLElement)) return false;
      if (node.getAttribute("data-type") === "mermaid") return true;
      if (node.classList?.contains("mermaid")) return true;
      return false;
    },
    replacement: (_content, node) => {
      const el = node as HTMLElement;
      const source =
        el.getAttribute("data-mermaid-source") ||
        el.getAttribute("data-source") ||
        unescapeHtml(el.textContent || "").trim();
      return `\n\n\`\`\`mermaid\n${source}\n\`\`\`\n\n`;
    },
  });

  // Code blocks with language (including language-mermaid as fence)
  td.addRule("fencedCodeBlock", {
    filter: (node) =>
      node.nodeName === "PRE" &&
      node.firstChild != null &&
      (node.firstChild as HTMLElement).nodeName === "CODE",
    replacement: (_content, node) => {
      const code = (node as HTMLElement).querySelector("code");
      if (!code) return "";
      const className = code.getAttribute("class") || "";
      const langMatch = className.match(/(?:language|lang)-([a-z0-9_+-]+)/i);
      const lang = langMatch?.[1] ?? "";
      const text = code.textContent || "";
      if (lang.toLowerCase() === "mermaid") {
        return `\n\n\`\`\`mermaid\n${text.replace(/\n$/, "")}\n\`\`\`\n\n`;
      }
      return `\n\n\`\`\`${lang}\n${text.replace(/\n$/, "")}\n\`\`\`\n\n`;
    },
  });

  // GFM tables
  td.addRule("table", {
    filter: "table",
    replacement: (_content, node) => {
      const md = tableToMarkdown(node as HTMLTableElement);
      return md ? `\n\n${md}\n\n` : "";
    },
  });

  // Task lists
  td.addRule("taskListItem", {
    filter: (node) => {
      if (node.nodeName !== "LI") return false;
      const parent = node.parentElement;
      if (!parent || (parent.getAttribute("data-type") !== "taskList" && parent.getAttribute("data-type") !== "taskList".toLowerCase())) {
        // also match ul with checkbox
        const input = (node as HTMLElement).querySelector?.('input[type="checkbox"]');
        return !!input;
      }
      return true;
    },
    replacement: (content, node) => {
      const input = (node as HTMLElement).querySelector?.('input[type="checkbox"]') as HTMLInputElement | null;
      const checked = input?.checked || (node as HTMLElement).getAttribute("data-checked") === "true";
      const body = content
        .replace(/^\n+/, "")
        .replace(/\n+$/, "")
        .replace(/\n/gm, "\n  ");
      return `- [${checked ? "x" : " "}] ${body}\n`;
    },
  });

  // Strikethrough
  td.addRule("strikethrough", {
    filter: (node) => {
      const name = node.nodeName;
      return name === "DEL" || name === "S" || name === "STRIKE";
    },
    replacement: (content) => `~~${content}~~`,
  });

  // Underline — no native MD; keep as HTML so richtext ↔ md preserves it
  td.addRule("underline", {
    filter: (node) => node.nodeName === "U",
    replacement: (content) => `<u>${content}</u>`,
  });

  // Images keep alt/src
  td.addRule("image", {
    filter: "img",
    replacement: (_content, node) => {
      const el = node as HTMLImageElement;
      const alt = el.getAttribute("alt") || "";
      const src = el.getAttribute("src") || "";
      const title = el.getAttribute("title");
      return title ? `![${alt}](${src} "${title}")` : `![${alt}](${src})`;
    },
  });

  turndownService = td;
  return td;
}

/**
 * Convert HTML (TipTap document) back to Markdown.
 * Round-trips mermaid blocks, GFM tables, fenced code, task lists.
 */
export function htmlToMarkdown(html: string): string {
  if (!html || html === "<p></p>") return "";
  const md = getTurndown().turndown(html);
  // Normalize excessive blank lines introduced by block rules
  return md.replace(/\n{3,}/g, "\n\n").trim() + (md.trim() ? "\n" : "");
}

export function isMermaidHtml(html: string): boolean {
  return /class=["']mermaid["']|data-type=["']mermaid["']/.test(html);
}
