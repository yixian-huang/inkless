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

  return (
    <header className={isReading ? "mb-8 pb-6 border-b border-border/80" : "mb-8"}>
      {isReading && (
        <p className="article-page-ui font-sans mb-4 text-sm">
          <Link
            to="/blog"
            className="text-on-surface-muted hover:text-primary transition-colors"
          >
            ← {t("blog.backToArchive")}
          </Link>
        </p>
      )}
      <h1
        className={
          isReading
            ? "article-page-title text-3xl md:text-[2.5rem] font-normal text-on-surface leading-[1.2] tracking-tight"
            : "article-page-title font-heading text-3xl md:text-4xl font-bold text-on-surface leading-tight"
        }
      >
        {title}
      </h1>
      {article.coverImage && (
        <img
          src={article.coverImage}
          alt={title}
          className={
            isReading
              ? "mt-8 w-full object-cover max-h-[400px] rounded-sm"
              : "mt-6 w-full rounded-card object-cover max-h-[420px]"
          }
        />
      )}
      <div
        className={`mt-5 flex flex-wrap items-center gap-x-3 gap-y-2 text-on-surface-muted article-page-ui ${
          isReading ? "font-sans text-xs" : "text-sm"
        }`}
      >
        <time dateTime={article.publishedAt || article.createdAt}>
          {formatArticleDate(article.publishedAt || article.createdAt, currentLocale)}
        </time>
        {bodyHtml && (
          <>
            <span className="text-on-surface-muted/40" aria-hidden="true">
              ·
            </span>
            <ArticleReadingMeta html={bodyHtml} />
          </>
        )}
        {typeof article.viewCount === "number" && article.viewCount >= 0 && (
          <>
            <span className="text-on-surface-muted/40" aria-hidden="true">
              ·
            </span>
            <span>{t("blog.viewCount", { count: article.viewCount })}</span>
          </>
        )}
      </div>
    </header>
  );
}
