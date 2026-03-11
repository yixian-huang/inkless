import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { getPublicTags } from "@/api/articles";
import type { Tag } from "@/api/articles";
import { PublicLayout } from "@/theme/layouts";
import PageHero from "@/components/feature/PageHero";

export default function TagsPage() {
  const { t } = useTranslation();
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const data = await getPublicTags();
        setTags(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load tags");
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  return (
    <PublicLayout>
      <PageHero title={t("tags.title", "标签")} subtitle={t("tags.subtitle", "浏览所有文章标签")} />
    <div className="max-w-6xl mx-auto px-4 py-12">
      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-600">{t("common.loading", "Loading...")}</div>
        </div>
      ) : error ? (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">{error}</div>
      ) : (<>

      {tags.length === 0 ? (
        <div className="flex items-center justify-center h-64">
          <p className="text-gray-500">{t("tags.empty", "暂无标签")}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {tags.map((tag) => (
            <Link
              key={tag.id}
              to={`/tags/${tag.slug}`}
              className="group flex items-center gap-4 bg-white rounded-lg shadow-sm border border-gray-200 p-4 hover:shadow-md transition-shadow"
            >
              {tag.coverImage ? (
                <div className="w-16 h-16 rounded-lg overflow-hidden flex-shrink-0">
                  <img
                    src={tag.coverImage}
                    alt={tag.zhName || tag.enName}
                    className="w-full h-full object-cover"
                    loading="lazy"
                  />
                </div>
              ) : (
                <div
                  className="w-16 h-16 rounded-lg flex-shrink-0 flex items-center justify-center"
                  style={{ backgroundColor: tag.color || "#e5e7eb" }}
                >
                  <span className="text-2xl text-white/80 font-bold">
                    {(tag.zhName || tag.enName || "#").charAt(0)}
                  </span>
                </div>
              )}
              <div className="min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  {tag.color && (
                    <span
                      className="w-3 h-3 rounded-full flex-shrink-0"
                      style={{ backgroundColor: tag.color }}
                    />
                  )}
                  <h2 className="text-base font-semibold text-gray-900 group-hover:text-blue-600 transition-colors truncate">
                    {tag.zhName || tag.enName}
                  </h2>
                </div>
                {tag.enName && tag.zhName && (
                  <p className="text-sm text-gray-400 truncate">{tag.enName}</p>
                )}
              </div>
            </Link>
          ))}
        </div>
      )}
    </>)}
    </div>
    </PublicLayout>
  );
}
