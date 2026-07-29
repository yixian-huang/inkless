interface DynamicPageHeaderProps {
  title: string;
  description?: string;
  /** Constrain to reading / content column when true */
  narrow?: boolean;
  maxWidth?: string;
}

/** Host-owned title block for reading-style dynamic pages (not theme chrome). */
export default function DynamicPageHeader({
  title,
  description,
  narrow = true,
  maxWidth,
}: DynamicPageHeaderProps) {
  if (!title && !description) return null;

  return (
    <header
      className="w-full border-b border-border/60 bg-surface"
      data-testid="dynamic-page-header"
    >
      <div
        className={[
          "mx-auto px-4 md:px-content xl:px-8 py-8 md:py-10",
          narrow ? "w-full" : "max-w-layout w-full",
        ].join(" ")}
        style={narrow && maxWidth ? { maxWidth } : undefined}
      >
        {title ? (
          <h1 className="text-2xl md:text-3xl lg:text-4xl font-semibold tracking-tight text-on-surface font-heading">
            {title}
          </h1>
        ) : null}
        {description ? (
          <p className="mt-3 text-base md:text-lg text-on-surface-muted leading-relaxed max-w-prose">
            {description}
          </p>
        ) : null}
      </div>
    </header>
  );
}
