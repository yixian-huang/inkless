import { EfShell } from "./shell";
import { asString, type SectionProps } from "./types";

export interface EfPullQuoteData {
  quote?: string;
  attribution?: string;
}

export default function EfPullQuote({ data }: SectionProps<EfPullQuoteData>) {
  const quote = asString(data.quote);
  const attribution = asString(data.attribution);

  if (!quote) return null;

  return (
    <EfShell>
      <figure className="max-w-3xl mx-auto text-center">
        <blockquote className="font-heading text-2xl md:text-3xl lg:text-4xl leading-snug text-on-surface font-medium">
          <span className="text-accent mr-1" aria-hidden>
            “
          </span>
          {quote}
          <span className="text-accent ml-1" aria-hidden>
            ”
          </span>
        </blockquote>
        {attribution ? (
          <figcaption className="mt-6 text-sm text-on-surface-muted tracking-wide">
            — {attribution}
          </figcaption>
        ) : null}
      </figure>
    </EfShell>
  );
}
