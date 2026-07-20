import { useState, useEffect, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { getPublicArticles } from "@/api/articles";
import SeoHead from "@/components/SeoHead";
import BlogPageShell from "@/components/blog/BlogPageShell";
import BlogPageHeader from "@/components/blog/BlogPageHeader";
import BlogArchiveNav from "@/components/blog/BlogArchiveNav";
import ArticleList from "@/components/blog/ArticleList";
import { useSEODefaults } from "@/hooks/useSEODefaults";
import { useLocaleMode } from "@/hooks/useLocaleMode";

const PAGE_SIZE = 12;

export default function BlogPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation("common");
  const { buildTitle, defaultDescription } = useSEODefaults();
  const { localeMode, defaultLocale, currentLocale } = useLocaleMode();

  const [articles, setArticles] = useState<Awaited<ReturnType<typeof getPublicArticles>>["items"]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(() => {
    const p = Number(searchParams.get("page"));
    return Number.isFinite(p) && p > 0 ? p : 1;
  });
  const [total, setTotal] = useState(0);

  const categoryFilter = searchParams.get("category") || undefined;
  const tagFilter = searchParams.get("tag") || undefined;

  const loadArticles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getPublicArticles(page, PAGE_SIZE, categoryFilter, tagFilter);
      setArticles(data.items || []);
      setTotal(data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load articles");
    } finally {
      setLoading(false);
    }
  }, [page, categoryFilter, tagFilter]);

  useEffect(() => {
    loadArticles();
  }, [loadArticles]);

  const totalPages = Math.ceil(total / PAGE_SIZE);

  const updateFilter = (key: "category" | "tag", slug: string) => {
    const params = new URLSearchParams(searchParams);
    if (params.get(key) === slug) {
      params.delete(key);
    } else {
      params.set(key, slug);
    }
    params.delete("page");
    setSearchParams(params);
    setPage(1);
  };

  const goToPage = (next: number) => {
    const params = new URLSearchParams(searchParams);
    if (next <= 1) {
      params.delete("page");
    } else {
      params.set("page", String(next));
    }
    setSearchParams(params);
    setPage(next);
  };

  const listTitle = t("blog.archiveTitle");
  const listDesc = t("blog.archiveDescription");

  return (
    <>
      <SeoHead
        title={buildTitle(listTitle)}
        description={listDesc || defaultDescription}
        ogTitle={listTitle}
        ogDescription={listDesc || defaultDescription}
        ogType="website"
        canonicalUrl="/blog"
      />
      <BlogPageShell>
        <BlogPageHeader title={listTitle} description={listDesc} />

        <BlogArchiveNav
          activeCategory={categoryFilter}
          activeTag={tagFilter}
          onFilterCategory={(slug) => updateFilter("category", slug)}
          onFilterTag={(slug) => updateFilter("tag", slug)}
        />

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">
            {error}
          </div>
        )}

        <ArticleList
          articles={articles}
          localeMode={localeMode}
          defaultLocale={defaultLocale}
          currentLocale={currentLocale}
          loading={loading}
          loadingLabel={t("status.loading")}
          emptyLabel={t("blog.noPosts")}
        />

        {totalPages > 1 && (
          <div className="mt-8 flex items-center justify-center gap-2 article-page-ui font-sans">
            <button
              type="button"
              onClick={() => goToPage(Math.max(1, page - 1))}
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
              onClick={() => goToPage(Math.min(totalPages, page + 1))}
              disabled={page >= totalPages}
              className="px-4 py-2 text-sm border border-border rounded-button text-on-surface hover:bg-surface-alt disabled:opacity-50"
            >
              {t("pagination.next")}
            </button>
          </div>
        )}
      </BlogPageShell>
    </>
  );
}
