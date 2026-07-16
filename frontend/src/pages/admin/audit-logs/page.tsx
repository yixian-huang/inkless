import { useState, useEffect, useCallback } from "react";
import { getAuditLogs, type AuditEvent, type AuditLogListResponse, type AuditLogFilters } from "@/api/auditLogs";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

const ACTION_OPTIONS = [
  { value: "", label: "全部操作" },
  { value: "content.create", label: "创建内容" },
  { value: "content.update", label: "更新内容" },
  { value: "content.save_draft", label: "保存草稿" },
  { value: "content.publish", label: "发布内容" },
  { value: "content.unpublish", label: "下线内容" },
  { value: "content.rollback", label: "回滚内容" },
  { value: "content.delete", label: "删除内容" },
  { value: "content.validate", label: "验证内容" },
  { value: "auth.login", label: "登录" },
  { value: "permissions.assign", label: "分配权限" },
  { value: "permissions.unassign", label: "取消权限" },
  { value: "migration.import", label: "数据迁移" },
  { value: "media.upload", label: "上传媒体" },
  { value: "media.delete", label: "删除媒体" },
  { value: "backup.create", label: "创建备份" },
  { value: "backup.restore", label: "恢复备份" },
];

const PAGE_SIZE = 20;

function parseDetails(details: string): Record<string, unknown> | null {
  try {
    return JSON.parse(details);
  } catch {
    return null;
  }
}

function formatDetailsSummary(details: string): string {
  const parsed = parseDetails(details);
  if (!parsed) return details || "-";

  const parts: string[] = [];
  if (parsed.pageKey) parts.push(`页面: ${parsed.pageKey}`);
  if (parsed.version) parts.push(`版本: ${parsed.version}`);
  if (parsed.filename) parts.push(`文件: ${parsed.filename}`);
  if (parsed.reason) parts.push(`原因: ${parsed.reason}`);
  if (parsed.error) parts.push(`错误: ${parsed.error}`);
  if (parsed.status) parts.push(`状态码: ${parsed.status}`);
  if (parsed.request_id) parts.push(`请求: ${parsed.request_id}`);

  return parts.length > 0 ? parts.join(", ") : JSON.stringify(parsed).slice(0, 80);
}

export default function AdminAuditLogsPage() {
  useDocumentTitle("审计日志");
  const [data, setData] = useState<AuditLogListResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState<AuditLogFilters>({});

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getAuditLogs(page, PAGE_SIZE, filters);
      setData(result);
    } catch {
      setError("获取审计日志失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  }, [page, filters]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0;

  const handleFilterChange = (key: keyof AuditLogFilters, value: string) => {
    setPage(1);
    setFilters((prev) => {
      const next = { ...prev };
      if (value) {
        next[key] = value;
      } else {
        delete next[key];
      }
      return next;
    });
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-gray-900">审计日志</h2>
        <button
          onClick={fetchData}
          disabled={loading}
          className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 disabled:opacity-50"
        >
          {loading ? "加载中..." : "刷新"}
        </button>
      </div>

      {/* Filters */}
      <div className="mb-6 flex flex-wrap gap-4">
        <select
          value={filters.action || ""}
          onChange={(e) => handleFilterChange("action", e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {ACTION_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>

        <input
          type="text"
          placeholder="操作人"
          value={filters.actor || ""}
          onChange={(e) => handleFilterChange("actor", e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />

        <input
          type="date"
          value={filters.from || ""}
          onChange={(e) => handleFilterChange("from", e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />

        <input
          type="date"
          value={filters.to || ""}
          onChange={(e) => handleFilterChange("to", e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-md text-red-700 text-sm">
          {error}
        </div>
      )}

      {loading && !data ? (
        <div className="text-center py-12 text-gray-500">加载中...</div>
      ) : data ? (
        <>
          <div className="bg-white shadow rounded-lg overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    时间
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    操作
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    操作人
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    资源
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    结果
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    详情
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {data.items.map((event: AuditEvent) => (
                  <tr key={event.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">
                      {new Date(event.createdAt).toLocaleString("zh-CN")}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 font-medium">
                      {event.action}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">
                      {event.actor}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">
                      {event.resource}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span
                        className={
                          event.result === "success"
                            ? "inline-flex px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800"
                            : "inline-flex px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800"
                        }
                      >
                        {event.result}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate" title={event.details}>
                      {formatDetailsSummary(event.details)}
                    </td>
                  </tr>
                ))}
                {data.items.length === 0 && (
                  <tr>
                    <td colSpan={6} className="px-6 py-8 text-center text-sm text-gray-500">
                      暂无审计日志
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="mt-4 flex items-center justify-between">
              <p className="text-sm text-gray-500">
                共 {data.total} 条，第 {page}/{totalPages} 页
              </p>
              <div className="flex gap-2">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="px-3 py-1 text-sm border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  上一页
                </button>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="px-3 py-1 text-sm border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  下一页
                </button>
              </div>
            </div>
          )}
        </>
      ) : null}
    </div>
  );
}
