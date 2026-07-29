import type { SectionProps } from "../types";
import { sanitizePublicHtml } from "@/utils/sanitizePublicHtml";

export interface RichTextSectionData {
  /** Plain text or HTML (TipTap / seed content). May be bilingual before SectionRenderer resolve. */
  content?: string;
  alignment?: "left" | "center";
}

function looksLikeHtml(value: string): boolean {
  return /<[a-zA-Z!/?]/.test(value);
}

export default function RichTextSection({ data }: SectionProps<RichTextSectionData>) {
  const { content, alignment = "left" } = data;

  if (!content || typeof content !== "string") return null;

  const alignClass = alignment === "center" ? "text-center" : "text-left";
  const shellClass = `max-w-layout mx-auto px-4 md:px-content xl:px-8 ${alignClass}`;

  // HTML bodies (admin rich text / agent seeds) must not be React-escaped as text.
  if (looksLikeHtml(content)) {
    const safe = sanitizePublicHtml(content);
    if (!safe) return null;
    return (
      <div className={shellClass}>
        <div
          className="prose prose-slate max-w-none text-on-surface-muted
            prose-headings:text-on-surface prose-a:text-primary
            prose-pre:bg-surface-alt prose-pre:text-sm
            prose-code:before:content-none prose-code:after:content-none"
          dangerouslySetInnerHTML={{ __html: safe }}
        />
      </div>
    );
  }

  const paragraphs = content.split(/\n\n+/).filter(Boolean);
  return (
    <div className={shellClass}>
      {paragraphs.map((para, i) => (
        <p
          key={i}
          className="text-base text-on-surface-muted leading-relaxed mb-4 last:mb-0 whitespace-pre-wrap"
        >
          {para}
        </p>
      ))}
    </div>
  );
}
