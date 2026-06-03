import { useState, useEffect, useCallback, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { getPublicArticle } from "@/api/articles";
import type { Article } from "@/api/articles";
import SeoHead from "@/components/SeoHead";
import BlogPageShell from "@/components/blog/BlogPageShell";
import CommentSection from "@/components/feature/CommentSection";
import { BlogFeatureGate } from "@/components/feature/BlogFeatureGate";
import { useLocaleMode } from "@/hooks/useLocaleMode";
import {
  articleTitle,
  articleBody,
  articleMetaDescription,
  formatArticleDate,
} from "@/utils/articleLocale";

export default function BlogDetailPage() {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation("common");
  const { localeMode, defaultLocale, currentLocale } = useLocaleMode();

  const [article, setArticle] = useState<Article | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
  const contentRef = useRef<HTMLElement>(null);

  useEffect(() => {
    if (!slug) return;

    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await getPublicArticle(slug);
        setArticle(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load article");
      } finally {
        setLoading(false);
      }
    };

    load();
  }, [slug]);

  const handleContentClick = useCallback((e: React.MouseEvent) => {
    const target = e.target as HTMLElement;
    if (target.tagName === "IMG") {
      const src = (target as HTMLImageElement).src;
      if (src) setLightboxSrc(src);
    }
  }, []);

  const title = article
    ? articleTitle(article, localeMode, defaultLocale, currentLocale)
    : "";
  const body = article
    ? articleBody(article, localeMode, defaultLocale, currentLocale)
    : "";
  const metaDesc = article
    ? articleMetaDescription(article, localeMode, defaultLocale, currentLocale)
    : "";

  return (
    <>
      {article && (
        <SeoHead
          title={title}
          description={metaDesc}
          ogTitle={article.zhSeoTitle || article.enSeoTitle || title}
          ogDescription={metaDesc}
          ogImage={article.ogImage || article.coverImage || ""}
          ogType="article"
          canonicalUrl={`/blog/${article.slug}`}
        />
      )}
      <BlogPageShell>
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-on-surface-muted">{t("status.loading")}</div>
          </div>
        ) : error || !article ? (
          <div className="text-center">
            <p className="text-red-600 mb-4">{error || t("blog.notFound")}</p>
            <button
              type="button"
              onClick={() => navigate("/blog")}
              className="text-primary hover:text-accent transition-colors"
            >
              {t("blog.backToArchive")}
            </button>
          </div>
        ) : (
          <>
            <header className="mb-8">
              <h1 className="text-3xl md:text-4xl font-heading font-bold text-on-surface leading-tight">
                {title}
              </h1>
              {article.coverImage && (
                <img
                  src={article.coverImage}
                  alt={title}
                  className="mt-6 w-full rounded-card object-cover max-h-[420px]"
                />
              )}
              <div className="mt-4 flex items-center gap-3 text-sm text-on-surface-muted flex-wrap">
                <time dateTime={article.publishedAt || article.createdAt}>
                  {formatArticleDate(article.publishedAt || article.createdAt, currentLocale)}
                </time>
                {article.category && (
                  <>
                    <span>&middot;</span>
                    <button
                      type="button"
                      onClick={() => navigate(`/blog?category=${article.category!.slug}`)}
                      className="text-primary hover:text-accent transition-colors"
                    >
                      {article.category.zhName || article.category.enName}
                    </button>
                  </>
                )}
                {article.tags && article.tags.length > 0 && (
                  <>
                    <span>&middot;</span>
                    {article.tags.map((tag) => (
                      <button
                        key={tag.id}
                        type="button"
                        onClick={() => navigate(`/blog?tag=${tag.slug}`)}
                        className="text-xs px-2.5 py-1 bg-surface-alt text-on-surface-muted rounded-full border border-border hover:bg-surface"
                      >
                        {tag.zhName || tag.enName}
                      </button>
                    ))}
                  </>
                )}
              </div>
            </header>

            <article
              ref={contentRef}
              className="tiptap ProseMirror prose prose-gray max-w-none article-public-view"
              dangerouslySetInnerHTML={{ __html: body }}
              onClick={handleContentClick}
            />

            {article.allowComments !== false && (
              <BlogFeatureGate feature="comments">
                <CommentSection contentType="article" contentId={article.id} />
              </BlogFeatureGate>
            )}
          </>
        )}
      </BlogPageShell>

      {lightboxSrc && (
        <div
          className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/70 cursor-zoom-out"
          onClick={() => setLightboxSrc(null)}
          onKeyDown={(e) => e.key === "Escape" && setLightboxSrc(null)}
          role="presentation"
        >
          <img
            src={lightboxSrc}
            alt=""
            className="max-w-[90vw] max-h-[90vh] object-contain rounded-lg shadow-2xl"
          />
        </div>
      )}
    </>
  );
}
