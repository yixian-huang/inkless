import { EfShell } from "./shell";
import { asString, type SectionProps } from "./types";

export interface EfHeroEditorialData {
  kicker?: string;
  title?: string;
  deck?: string;
  image?: string;
  ctaLabel?: string;
  ctaHref?: string;
  layout?: "full" | "split" | string;
}

export default function EfHeroEditorial({ data }: SectionProps<EfHeroEditorialData>) {
  const kicker = asString(data.kicker);
  const title = asString(data.title);
  const deck = asString(data.deck);
  const image = asString(data.image);
  const ctaLabel = asString(data.ctaLabel);
  const ctaHref = asString(data.ctaHref);
  const layout = asString(data.layout, "full") || "full";
  const isSplit = layout === "split";

  const copy = (
    <div className={isSplit ? "" : "relative z-10 max-w-3xl"}>
      {kicker ? (
        <p className="text-xs md:text-sm uppercase tracking-[0.2em] text-accent mb-4 font-medium">
          {kicker}
        </p>
      ) : null}
      {title ? (
        <h1 className="font-heading text-4xl md:text-5xl lg:text-6xl leading-tight text-on-surface font-semibold tracking-tight">
          {title}
        </h1>
      ) : null}
      {deck ? (
        <p className="mt-5 text-base md:text-lg text-on-surface-muted leading-relaxed max-w-2xl">
          {deck}
        </p>
      ) : null}
      {ctaLabel && ctaHref ? (
        <a
          href={ctaHref}
          className="inline-block mt-8 px-6 py-3 bg-primary text-on-primary text-sm uppercase tracking-wider font-medium hover:opacity-90 transition-opacity"
        >
          {ctaLabel}
        </a>
      ) : null}
    </div>
  );

  if (isSplit) {
    return (
      <EfShell>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-10 md:gap-14 items-center">
          <div className="order-2 md:order-1">{copy}</div>
          <div className="order-1 md:order-2">
            {image ? (
              <div className="aspect-[5/4] overflow-hidden rounded-sm bg-surface-alt">
                <img
                  src={image}
                  alt={title || ""}
                  className="w-full h-full object-cover object-center"
                />
              </div>
            ) : (
              <div className="aspect-[5/4] rounded-sm bg-surface-alt border border-border" />
            )}
          </div>
        </div>
      </EfShell>
    );
  }

  // Full-bleed: image behind/under title
  return (
    <div className="relative min-h-[320px] md:min-h-[420px] lg:min-h-[480px] flex items-end">
      {image ? (
        <>
          <img
            src={image}
            alt=""
            aria-hidden
            className="absolute inset-0 w-full h-full object-cover object-center"
          />
          <div className="absolute inset-0 bg-surface/70" />
        </>
      ) : (
        <div className="absolute inset-0 bg-surface-alt" />
      )}
      <EfShell className="relative z-10 w-full pb-12 md:pb-16 pt-24 md:pt-32">
        {copy}
      </EfShell>
    </div>
  );
}
