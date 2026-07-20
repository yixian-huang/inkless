import { Link } from "react-router-dom";
import type { Article } from "@/api/articles";
import { useIsReadingLayout } from "@/plugins/hooks";
import {
  articleTitle,
  articleBody,
  articleExcerpt,
  formatArticleDate,
} from "@/utils/articleLocale";
import type { LocaleMode, Locale } from "@/lib/locale";

interface ArticleListProps {
  articles: Article[];
  localeMode: LocaleMode;
  defaultLocale: Locale;
  currentLocale: Locale;
  /**
   * Optional side-effect when a post is activated (analytics, focus).
   * Navigation uses a real `<Link>` so middle-click / open-in-new-tab work.
   */
  onSelect?: (slug: string) => void;
  /** Path builder; default `/blog/:slug`. */
  hrefForSlug?: (slug: string) => string;
  loading?: boolean;
  loadingLabel?: string;
  emptyLabel?: string;
}

function ArticleListSkeleton({ isReading }: { isReading: boolean }) {
  return (
    <ul className="divide-y divide-border" aria-hidden="true">
      {[0, 1, 2].map((key) => (
        <li key={key} className={isReading ? "py-6 first:pt-0 animate-pulse" : "py-6 first:pt-0 animate-pulse"}>
          <div className="h-3.5 w-28 bg-surface-alt rounded" />
          <div className="mt-4 h-6 w-3/4 bg-surface-alt rounded" />
          <div className="mt-3 h-4 w-full bg-surface-alt rounded" />
        </li>
      ))}
    </ul>
  );
}

export default function ArticleList({
  articles,
  localeMode,
  defaultLocale,
  currentLocale,
  onSelect,
  hrefForSlug = (slug) => `/blog/${slug}`,
  loading = false,
  loadingLabel = "Loading...",
  emptyLabel = "No posts yet.",
}: ArticleListProps) {
  const isReading = useIsReadingLayout();

  if (loading) {
    return (
      <div>
        <p className="sr-only">{loadingLabel}</p>
        <ArticleListSkeleton isReading={isReading} />
      </div>
    );
  }

  if (articles.length === 0) {
    return (
      <p className={`text-on-surface-muted ${isReading ? "py-10 text-center text-base" : "py-8"}`}>
        {emptyLabel}
      </p>
    );
  }

  return (
    <ul className="divide-y divide-border">
      {articles.map((article) => {
        const title = articleTitle(article, localeMode, defaultLocale, currentLocale);
        const body = articleBody(article, localeMode, defaultLocale, currentLocale);
        const href = hrefForSlug(article.slug);
        return (
          <li key={article.id} className={isReading ? "py-6 first:pt-0" : "py-6 first:pt-0"}>
            <Link
              to={href}
              onClick={() => onSelect?.(article.slug)}
              className="block text-left w-full group focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 rounded-sm"
            >
              {isReading ? (
                <div className="flex items-baseline justify-between gap-4">
                  <h3 className="min-w-0 flex-1 text-xl font-heading font-normal text-on-surface group-hover:text-primary transition-colors">
                    <span className="group-hover:underline decoration-border underline-offset-4">
                      {title}
                    </span>
                  </h3>
                  <time
                    className="shrink-0 tabular-nums text-xs font-sans uppercase tracking-wider text-on-surface-muted"
                    dateTime={article.publishedAt || article.createdAt}
                  >
                    {formatArticleDate(article.publishedAt || article.createdAt, currentLocale)}
                  </time>
                </div>
              ) : (
                <>
                  <time
                    className="tabular-nums text-on-surface-muted text-sm"
                    dateTime={article.publishedAt || article.createdAt}
                  >
                    {formatArticleDate(article.publishedAt || article.createdAt, currentLocale)}
                  </time>
                  <h3 className="mt-1 text-lg font-semibold text-on-surface group-hover:text-primary transition-colors">
                    {title}
                  </h3>
                </>
              )}
              {(() => {
                const excerpt = articleExcerpt(body);
                if (!excerpt) return null;
                return (
                  <p
                    className={
                      isReading
                        ? "mt-3 text-on-surface-muted leading-relaxed line-clamp-2"
                        : "mt-2 text-on-surface-muted line-clamp-2"
                    }
                  >
                    {excerpt}
                  </p>
                );
              })()}
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
