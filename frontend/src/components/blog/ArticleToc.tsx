import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { TocHeading, TocLayout } from "@/utils/articleToc";

function TocList({
  headings,
  activeId,
  onSelect,
}: {
  headings: TocHeading[];
  activeId: string | null;
  onSelect: (id: string) => void;
}) {
  return (
    <ul className="space-y-1.5">
      {headings.map((h) => (
        <li key={h.id} className={h.level === 3 ? "pl-3" : ""}>
          <button
            type="button"
            onClick={() => onSelect(h.id)}
            className={`text-left text-sm leading-snug transition-colors w-full ${
              activeId === h.id
                ? "text-primary border-l-2 border-primary pl-2 font-medium"
                : "text-on-surface-muted hover:text-primary pl-2 border-l-2 border-transparent"
            }`}
          >
            {h.text}
          </button>
        </li>
      ))}
    </ul>
  );
}

export function ArticleTocInline({
  headings,
  layout,
  activeId,
  onSelect,
}: {
  headings: TocHeading[];
  layout: TocLayout;
  activeId: string | null;
  onSelect: (id: string) => void;
}) {
  const { t } = useTranslation("common");

  if (layout === "none" || headings.length === 0) return null;

  // On xl+ with full layout the floating sidebar is shown; keep inline for smaller screens.
  const hideOnDesktop = layout === "full";

  return (
    <details
      className={`article-page-ui font-sans mb-8 ${hideOnDesktop ? "xl:hidden" : ""}`}
      open={layout === "inline"}
    >
      <summary className="cursor-pointer list-none text-xs uppercase tracking-[0.15em] text-on-surface-muted hover:text-primary transition-colors [&::-webkit-details-marker]:hidden">
        <span className="inline-flex items-center gap-2">
          {t("blog.tocTitle")}
          <span className="text-on-surface-muted/60 normal-case tracking-normal">({headings.length})</span>
        </span>
      </summary>
      <nav className="mt-3 pt-3 border-t border-border/60" aria-label={t("blog.tocTitle")}>
        <TocList headings={headings} activeId={activeId} onSelect={onSelect} />
      </nav>
    </details>
  );
}

/**
 * Floating TOC in the right page gutter — does NOT shrink the article column.
 * Only visible when the viewport has spare horizontal space (xl+).
 */
export function ArticleTocSidebar({
  headings,
  layout,
  activeId,
  onSelect,
}: {
  headings: TocHeading[];
  layout: TocLayout;
  activeId: string | null;
  onSelect: (id: string) => void;
}) {
  const { t } = useTranslation("common");

  if (layout !== "full" || headings.length === 0) return null;

  return (
    <aside
      className="hidden xl:block absolute top-0 left-full pl-8 w-40 2xl:w-44 article-page-ui font-sans pointer-events-auto"
      aria-label={t("blog.tocTitle")}
    >
      <div className="sticky top-24 max-h-[calc(100vh-6rem)] overflow-y-auto pr-1">
        <p className="text-xs uppercase tracking-[0.15em] text-on-surface-muted mb-3">
          {t("blog.tocTitle")}
        </p>
        <TocList headings={headings} activeId={activeId} onSelect={onSelect} />
      </div>
    </aside>
  );
}

/**
 * Body always takes the full reading-column width.
 * Sidebar is absolutely positioned in the right margin outside the content box.
 */
export function ArticleTocLayout({
  layout,
  sidebar,
  children,
}: {
  layout: TocLayout;
  sidebar: ReactNode;
  children: ReactNode;
}) {
  if (layout !== "full") {
    return <>{children}</>;
  }

  return (
    <div className="relative">
      <div className="min-w-0 w-full">{children}</div>
      {sidebar}
    </div>
  );
}
