import { EfShell } from "./shell";
import { asString, type SectionProps } from "./types";

export interface EfCtaBandData {
  title?: string;
  body?: string;
  ctaLabel?: string;
  ctaHref?: string;
}

/**
 * Full-width conversion band. Outer section background can still come from
 * settings; component paints primary surface for the band content area.
 */
export default function EfCtaBand({ data }: SectionProps<EfCtaBandData>) {
  const title = asString(data.title);
  const body = asString(data.body);
  const ctaLabel = asString(data.ctaLabel);
  const ctaHref = asString(data.ctaHref);

  if (!title && !ctaLabel) return null;

  return (
    <div className="bg-primary text-on-primary">
      <EfShell className="py-12 md:py-16">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-8">
          <div className="max-w-2xl">
            {title ? (
              <h2 className="font-heading text-2xl md:text-3xl lg:text-4xl font-semibold leading-snug">
                {title}
              </h2>
            ) : null}
            {body ? (
              <p className="mt-3 text-sm md:text-base opacity-90 leading-relaxed">{body}</p>
            ) : null}
          </div>
          {ctaLabel && ctaHref ? (
            <a
              href={ctaHref}
              className="inline-flex shrink-0 items-center justify-center px-7 py-3 bg-on-primary text-primary text-sm uppercase tracking-wider font-medium hover:opacity-90 transition-opacity"
            >
              {ctaLabel}
            </a>
          ) : null}
        </div>
      </EfShell>
    </div>
  );
}
