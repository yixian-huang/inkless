import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import type { Article } from "@/api/articles";
import ArticleReadingMeta from "@/components/blog/ArticleReadingMeta";
import { useIsReadingLayout } from "@/plugins/hooks";
import type { Locale } from "@/lib/locale";
import { formatArticleDate } from "@/utils/articleLocale";

interface ArticlePostHeaderProps {
  title: string;
  bodyHtml: string;
  article: Article;
  currentLocale: Locale;
}

/** Article header — must sit inside `ArticleTypographyRoot`. */
export default function ArticlePostHeader({
  title,
  bodyHtml,
  article,
  currentLocale,
}: ArticlePostHeaderProps) {
  const { t } = useTranslation("common");
  const isReading = useIsReadingLayout();
  const showViews = typeof article.viewCount === "number" && article.viewCount > 0;

  return (
    <header className={isReading ? "mb-7 pb-5 border-b border-border/70" : "mb-8"}>
      {isReading && (
        <p className="article-page-ui font-sans mb-3 text-xs tracking-wide">
          <Link
            to="/blog"
            className="text-on-surface-muted/80 hover:text-on-surface transition-colors"
          >
            ← {t("blog.backToArchive")}
          </Link>
        </p>
      )}
      <h1
        className={
          isReading
            ? "article-page-title text-[1.75rem] sm:text-3xl md:text-[2.35rem] font-normal text-on-surface leading-[1.22] tracking-tight"
            : "article-page-title font-heading text-3xl md:text-4xl font-bold text-on-surface leading-tight"
        }
      >
        {title}
      </h1>
      <div
        className={`mt-4 flex flex-wrap items-center gap-x-2.5 gap-y-1.5 text-on-surface-muted article-page-ui ${
          isReading ? "font-sans text-[11px] sm:text-xs tracking-wide" : "text-sm"
        }`}
      >
        <time dateTime={article.publishedAt || article.createdAt}>
          {formatArticleDate(article.publishedAt || article.createdAt, currentLocale)}
        </time>
        {bodyHtml && (
          <>
            <span className="text-on-surface-muted/35" aria-hidden="true">
              ·
            </span>
            <ArticleReadingMeta html={bodyHtml} />
          </>
        )}
        {showViews && (
          <>
            <span className="text-on-surface-muted/35" aria-hidden="true">
              ·
            </span>
            <span className={isReading ? "text-on-surface-muted/75" : undefined}>
              {t("blog.viewCount", { count: article.viewCount })}
            </span>
          </>
        )}
      </div>
      {article.coverImage && (
        <img
          src={article.coverImage}
          alt=""
          className={
            isReading
              ? "mt-7 w-full object-cover max-h-[380px] rounded-sm"
              : "mt-6 w-full rounded-card object-cover max-h-[420px]"
          }
        />
      )}
    </header>
  );
}
