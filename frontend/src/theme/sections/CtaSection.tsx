import type { SectionProps } from "../types";

export interface CtaSectionData {
  title?: string;
  subtitle?: string;
  primaryLabel?: string;
  primaryHref?: string;
  secondaryLabel?: string;
  secondaryHref?: string;
}

/** Full-width call-to-action band for dynamic pages (Host baseline). */
export default function CtaSection({ data }: SectionProps<CtaSectionData>) {
  const {
    title,
    subtitle,
    primaryLabel,
    primaryHref = "#",
    secondaryLabel,
    secondaryHref = "#",
  } = data;

  if (!title && !primaryLabel && !secondaryLabel) return null;

  return (
    <div
      className="max-w-layout w-full mx-auto px-4 md:px-content xl:px-8"
      data-testid="cta-section"
    >
      <div className="rounded-2xl bg-surface-alt border border-border/60 px-6 py-10 md:px-10 md:py-12 text-center">
        {title ? (
          <h2 className="text-2xl md:text-3xl font-semibold tracking-tight text-on-surface font-heading">
            {title}
          </h2>
        ) : null}
        {subtitle ? (
          <p className="mt-3 text-base md:text-lg text-on-surface-muted max-w-2xl mx-auto leading-relaxed">
            {subtitle}
          </p>
        ) : null}
        {(primaryLabel || secondaryLabel) && (
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            {primaryLabel ? (
              <a
                href={primaryHref || "#"}
                className="inline-flex items-center justify-center rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-white shadow-sm hover:opacity-90 transition-opacity"
              >
                {primaryLabel}
              </a>
            ) : null}
            {secondaryLabel ? (
              <a
                href={secondaryHref || "#"}
                className="inline-flex items-center justify-center rounded-lg border border-border bg-surface px-5 py-2.5 text-sm font-medium text-on-surface hover:bg-surface-alt transition-colors"
              >
                {secondaryLabel}
              </a>
            ) : null}
          </div>
        )}
      </div>
    </div>
  );
}
