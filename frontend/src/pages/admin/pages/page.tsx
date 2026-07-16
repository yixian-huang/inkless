import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  listUnifiedPages,
  deleteUnifiedPage,
  type UnifiedPageItem,
} from "@/api/unifiedPages";
import { useAuth } from "@/contexts/AuthContext";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

export default function AdminPagesPage() {
  useDocumentTitle("页面管理");
  const [pages, setPages] = useState<UnifiedPageItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState("");
  const [modeFilter, setModeFilter] = useState("");

  const navigate = useNavigate();
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("pages:create");
  const canUpdate = hasPermission("pages:update");
  const canDelete = hasPermission("pages:delete");

  const fetchPages = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listUnifiedPages(
        statusFilter || undefined,
        modeFilter || undefined,
      );
      setPages(data ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [statusFilter, modeFilter]);

  useEffect(() => {
    fetchPages();
  }, [fetchPages]);

  const handleDelete = async (id: number) => {
    if (!confirm("确定删除此页面？此操作不可撤销。")) return;
    try {
      await deleteUnifiedPage(id);
      fetchPages();
    } catch {
      // ignore
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">页面管理</h2>
        <div className="flex items-center gap-3">
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm"
          >
            <option value="">全部状态</option>
            <option value="draft">草稿</option>
            <option value="published">已发布</option>
          </select>
          <select
            value={modeFilter}
            onChange={(e) => setModeFilter(e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm"
          >
            <option value="">全部模式</option>
            <option value="template">模板</option>
            <option value="composable">自由组合</option>
          </select>
          {canCreate && (
            <button
              onClick={() => navigate("/admin/pages/new")}
              className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700"
            >
              新建页面
            </button>
          )}
        </div>
      </div>

      {loading ? (
        <div className="text-center py-12 text-gray-500">加载中...</div>
      ) : pages.length === 0 ? (
        <div className="text-center py-12 text-gray-500">暂无页面</div>
      ) : (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  标题
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  路径
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  模式
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  草稿版本
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  发布版本
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {pages.map((page) => (
                <tr key={page.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 text-sm text-gray-900">
                    {page.zhTitle || page.enTitle || "(无标题)"}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 font-mono">
                    /{page.slug}
                  </td>
                  <td className="px-6 py-4">
                    <span className={`inline-flex px-2 py-0.5 text-xs rounded-full ${
                      page.mode === "template"
                        ? "bg-purple-100 text-purple-700"
                        : "bg-blue-100 text-blue-700"
                    }`}>
                      {page.mode === "template" ? "模板" : "自由组合"}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span className={`inline-flex px-2 py-1 text-xs rounded-full ${
                      page.status === "published"
                        ? "bg-green-100 text-green-800"
                        : "bg-yellow-100 text-yellow-800"
                    }`}>
                      {page.status === "published" ? "已发布" : "草稿"}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {page.draftVersion}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {page.publishedVersion > 0 ? page.publishedVersion : "-"}
                  </td>
                  <td className="px-6 py-4 text-right text-sm space-x-2">
                    <button
                      onClick={() => navigate(`/admin/pages/edit/${page.id}`)}
                      className="text-blue-600 hover:text-blue-800"
                    >
                      {canUpdate ? "编辑" : "查看"}
                    </button>
                    {canDelete && (
                      <button
                        onClick={() => handleDelete(page.id)}
                        className="text-red-600 hover:text-red-800"
                      >
                        删除
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
