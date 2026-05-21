import { useState, useEffect, useCallback } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { getPublicArticles } from "@/api/articles";
import type { Article } from "@/api/articles";
import { PublicLayout } from "@/theme/layouts";
import PageHero from "@/components/feature/PageHero";
import SeoHead from "@/components/SeoHead";

export default function BlogPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 9;

  const categoryFilter = searchParams.get("category") || undefined;
  const tagFilter = searchParams.get("tag") || undefined;

  const loadArticles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getPublicArticles(page, pageSize, categoryFilter, tagFilter);
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

  const totalPages = Math.ceil(total / pageSize);

  const handleCategoryClick = (slug: string) => {
    const params = new URLSearchParams(searchParams);
    if (params.get("category") === slug) {
      params.delete("category");
    } else {
      params.set("category", slug);
    }
    params.delete("page");
    setSearchParams(params);
    setPage(1);
  };

  const handleTagClick = (slug: string) => {
    const params = new URLSearchParams(searchParams);
    if (params.get("tag") === slug) {
      params.delete("tag");
    } else {
      params.set("tag", slug);
    }
    params.delete("page");
    setSearchParams(params);
    setPage(1);
  };

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return "";
    return new Date(dateStr).toLocaleDateString("zh-CN", {
      year: "numeric",
      month: "long",
      day: "numeric",
    });
  };

  const getExcerpt = (body: string, maxLen: number = 120) => {
    if (!body) return "";
    const text = body.replace(/<[^>]*>/g, "").trim();
    return text.length > maxLen ? text.slice(0, maxLen) + "..." : text;
  };

  return (
    <PublicLayout>
      <SeoHead
        title="博客"
        description="博客 - 行业洞察、最新动态与专家观点"
        ogTitle="博客"
        ogDescription="博客 - 行业洞察、最新动态与专家观点"
        ogType="website"
        canonicalUrl="/blog"
      />
      <PageHero title="Blog" subtitle="Insights, updates, and expert perspectives" />
    <div className="max-w-6xl mx-auto px-4 py-12">

      {/* Active filters */}
      {(categoryFilter || tagFilter) && (
        <div className="mb-6 flex items-center gap-2 flex-wrap">
          <span className="text-sm text-gray-500">Filters:</span>
          {categoryFilter && (
            <button
              onClick={() => handleCategoryClick(categoryFilter)}
              className="inline-flex items-center gap-1 px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm hover:bg-blue-200"
            >
              Category: {categoryFilter}
              <span className="ml-1">&times;</span>
            </button>
          )}
          {tagFilter && (
            <button
              onClick={() => handleTagClick(tagFilter)}
              className="inline-flex items-center gap-1 px-3 py-1 bg-green-100 text-green-800 rounded-full text-sm hover:bg-green-200"
            >
              Tag: {tagFilter}
              <span className="ml-1">&times;</span>
            </button>
          )}
        </div>
      )}

      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">
          {error}
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-600">Loading...</div>
        </div>
      ) : articles.length === 0 ? (
        <div className="flex items-center justify-center h-64">
          <p className="text-gray-500">No articles found.</p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {articles.map((article) => (
              <article
                key={article.id}
                className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden hover:shadow-md transition-shadow cursor-pointer"
                onClick={() => navigate(`/blog/${article.slug}`)}
              >
                {article.coverImage && (
                  <div className="aspect-video bg-gray-100 overflow-hidden">
                    <img
                      src={article.coverImage}
                      alt={article.zhTitle || article.enTitle}
                      className="w-full h-full object-cover"
                      loading="lazy"
                    />
                  </div>
                )}
                <div className="p-5">
                  <div className="flex items-center gap-2 mb-3 flex-wrap">
                    {article.category && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleCategoryClick(article.category!.slug);
                        }}
                        className="text-xs px-2 py-0.5 bg-blue-50 text-blue-700 rounded hover:bg-blue-100"
                      >
                        {article.category.zhName || article.category.enName}
                      </button>
                    )}
                    {article.tags?.map((tag) => (
                      <button
                        key={tag.id}
                        onClick={(e) => {
                          e.stopPropagation();
                          handleTagClick(tag.slug);
                        }}
                        className="text-xs px-2 py-0.5 bg-gray-100 text-gray-600 rounded hover:bg-gray-200"
                      >
                        {tag.zhName || tag.enName}
                      </button>
                    ))}
                  </div>
                  <h2 className="text-lg font-semibold text-gray-900 mb-2 line-clamp-2">
                    {article.zhTitle || article.enTitle}
                  </h2>
                  <p className="text-sm text-gray-600 mb-3 line-clamp-3">
                    {getExcerpt(article.zhBody || article.enBody)}
                  </p>
                  <div className="text-xs text-gray-400">
                    {formatDate(article.publishedAt || article.createdAt)}
                  </div>
                </div>
              </article>
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="mt-8 flex items-center justify-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="px-4 py-2 text-sm border rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                Previous
              </button>
              <span className="text-sm text-gray-600 px-4">
                {page} / {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="px-4 py-2 text-sm border rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
    </PublicLayout>
  );
}
