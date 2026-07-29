import type { SectionProps } from "../types";

export interface FaqItem {
  question?: string;
  answer?: string;
}

export interface FaqSectionData {
  title?: string;
  items?: FaqItem[];
}

/** Accordion-style FAQ for policies and product Q&A on dynamic pages. */
export default function FaqSection({ data }: SectionProps<FaqSectionData>) {
  const { title, items = [] } = data;
  const visible = items.filter((it) => it.question || it.answer);
  if (!title && visible.length === 0) return null;

  return (
    <div
      className="max-w-layout w-full mx-auto px-4 md:px-content xl:px-8"
      data-testid="faq-section"
    >
      {title ? (
        <h2 className="text-2xl md:text-3xl font-semibold tracking-tight text-on-surface font-heading mb-6 md:mb-8">
          {title}
        </h2>
      ) : null}
      <div className="divide-y divide-border border border-border/70 rounded-xl overflow-hidden bg-surface">
        {visible.map((item, i) => (
          <details
            key={`${item.question?.slice(0, 32) ?? "q"}-${i}`}
            className="group"
          >
            <summary className="cursor-pointer list-none flex items-center justify-between gap-3 px-4 py-4 md:px-5 md:py-4 text-left text-on-surface font-medium hover:bg-surface-alt/60 transition-colors [&::-webkit-details-marker]:hidden">
              <span>{item.question || "问题"}</span>
              <span
                className="text-on-surface-muted text-lg leading-none transition-transform group-open:rotate-45"
                aria-hidden
              >
                +
              </span>
            </summary>
            {item.answer ? (
              <div className="px-4 pb-4 md:px-5 md:pb-5 text-sm md:text-base text-on-surface-muted leading-relaxed whitespace-pre-wrap">
                {item.answer}
              </div>
            ) : null}
          </details>
        ))}
      </div>
    </div>
  );
}
