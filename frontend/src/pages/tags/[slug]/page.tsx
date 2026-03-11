import { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { getPublicTagBySlug } from "@/api/articles";
import type { Article, Tag } from "@/api/articles";
import { PublicLayout } from "@/theme/layouts";
import PageHero from "@/components/feature/PageHero";

export default function TagDetailPage() {
  const { slug } = useParams<{ slug: string }>();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [tag, setTag] = useState<Tag | null>(null);
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 9;

  useEffect(() => {
    if (!slug) return;
    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await getPublicTagBySlug(slug, page, pageSize);
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

  const totalPages = Math.ceil(total / pageSize);

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
      <PageHero
        title={tag?.zhName || tag?.enName || t("tags.title", "标签")}
        imageSrc={tag?.coverImage || undefined}
      />
    <div className="max-w-6xl mx-auto px-4 py-12">
      {loading && !tag ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-600">{t("common.loading", "Loading...")}</div>
        </div>
      ) : error || !tag ? (
        <div className="max-w-3xl mx-auto text-center">
          <p className="text-red-600 mb-4">{error || "Tag not found"}</p>
          <button onClick={() => navigate("/tags")} className="text-blue-600 hover:text-blue-800">
            {t("tags.backToList", "返回标签列表")}
          </button>
        </div>
      ) : (<>
      {/* Articles */}
      {articles.length === 0 ? (
        <div className="flex items-center justify-center h-40">
          <p className="text-gray-500">{t("tags.noArticles", "该标签下暂无文章")}</p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {articles.map((article) => (
              <Link
                key={article.id}
                to={`/blog/${article.slug}`}
                className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden hover:shadow-md transition-shadow"
              >
                {article.coverImage ? (
                  <div className="aspect-video bg-gray-100 overflow-hidden">
                    <img
                      src={article.coverImage}
                      alt={article.zhTitle || article.enTitle}
                      className="w-full h-full object-cover"
                      loading="lazy"
                    />
                  </div>
                ) : (
                  <div className="aspect-video bg-gradient-to-br from-gray-50 to-gray-100" />
                )}
                <div className="p-5">
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
              </Link>
            ))}
          </div>

          {totalPages > 1 && (
            <div className="mt-8 flex items-center justify-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="px-4 py-2 text-sm border rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                {t("common.prev", "上一页")}
              </button>
              <span className="text-sm text-gray-600 px-4">
                {page} / {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="px-4 py-2 text-sm border rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                {t("common.next", "下一页")}
              </button>
            </div>
          )}
        </>
      )}
      </>)}
    </div>
    </PublicLayout>
  );
}
