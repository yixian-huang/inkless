import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { getPublicTagBySlug } from "@/api/articles";
import type { Article, Tag } from "@/api/articles";
import SeoHead from "@/components/SeoHead";
import BlogPageShell from "@/components/blog/BlogPageShell";
import BlogPageHeader from "@/components/blog/BlogPageHeader";
import ArticleList from "@/components/blog/ArticleList";
import { pickLocalizedName } from "@/components/blog/pickLocalizedName";
import { useLocaleMode } from "@/hooks/useLocaleMode";
import { useSEODefaults } from "@/hooks/useSEODefaults";

const PAGE_SIZE = 12;

export default function TagDetailPage() {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation("common");
  const { buildTitle } = useSEODefaults();
  const { localeMode, defaultLocale, currentLocale } = useLocaleMode();

  const [tag, setTag] = useState<Tag | null>(null);
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    if (!slug) return;
    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await getPublicTagBySlug(slug, page, PAGE_SIZE);
        setTag(data.tag);
        setArticles(data.articles?.items || []);
        setTotal(data.articles?.total || 0);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load tag");
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [slug, page]);

  const totalPages = Math.ceil(total / PAGE_SIZE);

  const tagName = tag ? pickLocalizedName(tag.zhName, tag.enName, currentLocale) : "";
  const displayTitle = tagName ? `#${tagName}` : t("blog.tags");

  return (
    <>
      {tag && (
        <SeoHead
          title={buildTitle(displayTitle)}
          description={t("blog.tagPosts")}
          ogTitle={displayTitle}
          ogDescription={t("blog.tagPosts")}
          ogType="website"
          canonicalUrl={`/tags/${tag.slug}`}
        />
      )}
      <BlogPageShell>
        {error && !tag ? (
          <div className="text-center py-16">
            <p className="text-red-600 mb-4">{error || t("status.error")}</p>
            <button
              type="button"
              onClick={() => navigate("/tags")}
              className="text-primary hover:text-accent transition-colors"
            >
              {t("blog.tagsPageTitle")}
            </button>
          </div>
        ) : (
          <>
            <BlogPageHeader
              title={displayTitle}
              description={t("blog.tagPosts")}
              backTo={{ href: "/tags", label: t("blog.tagsPageTitle") }}
            />

            <ArticleList
              articles={articles}
              localeMode={localeMode}
              defaultLocale={defaultLocale}
              currentLocale={currentLocale}
              onSelect={(articleSlug) => navigate(`/blog/${articleSlug}`)}
              loading={loading}
              loadingLabel={t("status.loading")}
              emptyLabel={t("blog.noPosts")}
            />

            {totalPages > 1 && (
              <div className="mt-8 flex items-center justify-center gap-2 article-page-ui font-sans">
                <button
                  type="button"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="px-4 py-2 text-sm border border-border rounded-button text-on-surface hover:bg-surface-alt disabled:opacity-50"
                >
                  {t("pagination.prev")}
                </button>
                <span className="text-sm text-on-surface-muted px-4">
                  {page} / {totalPages}
                </span>
                <button
                  type="button"
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="px-4 py-2 text-sm border border-border rounded-button text-on-surface hover:bg-surface-alt disabled:opacity-50"
                >
                  {t("pagination.next")}
                </button>
              </div>
            )}
          </>
        )}
      </BlogPageShell>
    </>
  );
}
