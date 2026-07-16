import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { getAdminArticles, deleteArticle } from "@/api/articles";
import type { Article } from "@/api/articles";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

export default function AdminArticlesPage() {
  useDocumentTitle("文章管理");
  const navigate = useNavigate();

  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [deleting, setDeleting] = useState<number | null>(null);
  const pageSize = 15;

  const loadArticles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getAdminArticles(page, pageSize, statusFilter || undefined);
      setArticles(data.items || []);
      setTotal(data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load articles");
    } finally {
      setLoading(false);
    }
  }, [page, statusFilter]);

  useEffect(() => {
    loadArticles();
  }, [loadArticles]);

  const handleDelete = async (article: Article) => {
    if (!confirm(`Are you sure you want to delete "${article.zhTitle || article.enTitle}"?`)) return;

    setDeleting(article.id);
    setError(null);
    try {
      await deleteArticle(article.id);
      setArticles((prev) => prev.filter((a) => a.id !== article.id));
      setTotal((prev) => prev - 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Delete failed");
    } finally {
      setDeleting(null);
    }
  };

  const totalPages = Math.ceil(total / pageSize);

  const statusLabel = (status: string) => {
    switch (status) {
      case "published":
        return <span className="px-2 py-0.5 text-xs bg-green-100 text-green-800 rounded-full">Published</span>;
      case "draft":
        return <span className="px-2 py-0.5 text-xs bg-yellow-100 text-yellow-800 rounded-full">Draft</span>;
      default:
        return <span className="px-2 py-0.5 text-xs bg-gray-100 text-gray-600 rounded-full">{status}</span>;
    }
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">文章管理</h1>
          <p className="text-sm text-gray-500 mt-1">Total: {total}</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate("/admin/articles/categories")}
            className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm"
          >
            Categories
          </button>
          <button
            onClick={() => navigate("/admin/articles/tags")}
            className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm"
          >
            Tags
          </button>
          <button
            onClick={() => navigate("/admin/articles/new")}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
          >
            New Article
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">
          {error}
        </div>
      )}

      {/* Status filter */}
      <div className="mb-4 flex items-center gap-2">
        <span className="text-sm text-gray-600">Status:</span>
        {[
          { value: "", label: "All" },
          { value: "draft", label: "Draft" },
          { value: "published", label: "Published" },
        ].map((opt) => (
          <button
            key={opt.value}
            onClick={() => {
              setStatusFilter(opt.value);
              setPage(1);
            }}
            className={`px-3 py-1.5 text-sm rounded-lg border ${
              statusFilter === opt.value
                ? "bg-blue-50 border-blue-300 text-blue-700"
                : "border-gray-300 text-gray-600 hover:bg-gray-50"
            }`}
          >
            {opt.label}
          </button>
        ))}
      </div>

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
          <div className="bg-white shadow rounded-lg overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Title
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Category
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Created
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {articles.map((article) => (
                  <tr key={article.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <div className="text-sm font-medium text-gray-900 truncate max-w-xs">
                        {article.zhTitle || article.enTitle || "(Untitled)"}
                      </div>
                      <div className="text-xs text-gray-500 truncate max-w-xs">
                        /{article.slug}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {statusLabel(article.status)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                      {article.category?.zhName || article.category?.enName || "-"}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(article.createdAt).toLocaleDateString("zh-CN")}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                      <button
                        onClick={() => navigate(`/admin/articles/edit/${article.id}`)}
                        className="text-blue-600 hover:text-blue-800 mr-3"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => handleDelete(article)}
                        disabled={deleting === article.id}
                        className="text-red-600 hover:text-red-800 disabled:opacity-50"
                      >
                        {deleting === article.id ? "..." : "Delete"}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="mt-6 flex items-center justify-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="px-3 py-1.5 text-sm border rounded hover:bg-gray-50 disabled:opacity-50"
              >
                Previous
              </button>
              <span className="text-sm text-gray-600">
                {page} / {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="px-3 py-1.5 text-sm border rounded hover:bg-gray-50 disabled:opacity-50"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
