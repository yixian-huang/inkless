import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { getPublicCategories } from "@/api/articles";
import type { Category } from "@/api/articles";
import PageHero from "@/components/feature/PageHero";
import BlogPageShell from "@/components/blog/BlogPageShell";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

export default function CategoriesPage() {
  useDocumentTitle("分类");
  const { t } = useTranslation();
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const data = await getPublicCategories();
        setCategories(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load categories");
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  return (
    <>
      <PageHero title={t("categories.title", "分类")} subtitle={t("categories.subtitle", "浏览所有文章分类")} />
      <BlogPageShell>
      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-600">{t("common.loading", "Loading...")}</div>
        </div>
      ) : error ? (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">{error}</div>
      ) : (<>

      {categories.length === 0 ? (
        <div className="flex items-center justify-center h-64">
          <p className="text-gray-500">{t("categories.empty", "暂无分类")}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {categories.map((category) => (
            <Link
              key={category.id}
              to={`/categories/${category.slug}`}
              className="group bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden hover:shadow-md transition-shadow"
            >
              {category.coverImage ? (
                <div className="aspect-video bg-gray-100 overflow-hidden">
                  <img
                    src={category.coverImage}
                    alt={category.zhName || category.enName}
                    className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                    loading="lazy"
                  />
                </div>
              ) : (
                <div className="aspect-video bg-gradient-to-br from-blue-100 to-indigo-200 flex items-center justify-center">
                  <span className="text-4xl text-blue-400/60">
                    {(category.zhName || category.enName || "").charAt(0)}
                  </span>
                </div>
              )}
              <div className="p-5">
                <h2 className="text-lg font-semibold text-gray-900 mb-1 group-hover:text-blue-600 transition-colors">
                  {category.zhName || category.enName}
                </h2>
                {category.enName && category.zhName && (
                  <p className="text-sm text-gray-400 mb-2">{category.enName}</p>
                )}
                {category.zhDescription && (
                  <p className="text-sm text-gray-600 line-clamp-2">{category.zhDescription}</p>
                )}
              </div>
            </Link>
          ))}
        </div>
      )}
      </>)}
      </BlogPageShell>
    </>
  );
}
