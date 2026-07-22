import { EfShell } from "./shell";
import { asString, type SectionProps } from "./types";

export interface EfRichTextData {
  title?: string;
  body?: string;
}

/** Plain-text reading column — `\n` / blank lines → paragraphs. No raw HTML in v1. */
export default function EfRichText({ data }: SectionProps<EfRichTextData>) {
  const title = asString(data.title);
  const body = asString(data.body);

  if (!title && !body) return null;

  const paragraphs = body
    .split(/\n\n+/)
    .map((p) => p.replace(/\n/g, " ").trim())
    .filter(Boolean);

  return (
    <EfShell>
      <div className="max-w-prose">
        {title ? (
          <h2 className="font-heading text-2xl md:text-3xl text-on-surface font-semibold mb-6">
            {title}
          </h2>
        ) : null}
        {paragraphs.map((para, i) => (
          <p
            key={i}
            className="text-base md:text-lg text-on-surface-muted leading-relaxed mb-4 last:mb-0"
          >
            {para}
          </p>
        ))}
      </div>
    </EfShell>
  );
}
