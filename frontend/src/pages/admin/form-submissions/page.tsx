import { useState, useCallback, Fragment } from "react";
import {
  getFormSubmissions,
  getSubmissionCounts,
  updateSubmissionStatus,
  bulkUpdateStatus,
  deleteFormSubmission,
} from "@/api/formSubmissions";
import type { FormSubmission } from "@/api/formSubmissions";
import {
  AdminBadge,
  AdminButton,
  AdminErrorBanner,
  AdminLoading,
  AdminPageHeader,
  AdminPagination,
  AdminTable,
  AdminTableBody,
  AdminTableHead,
  AdminTd,
  AdminTh,
  useAdminConfirm,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { invalidateAdminQueryPrefix, useAdminQuery } from "@/lib/adminQuery";
import { adminQueryKeys } from "@/lib/adminQueryKeys";

type StatusFilter = "" | "unread" | "read" | "archived";

const STATUS_TABS: { label: string; value: StatusFilter }[] = [
  { label: "全部", value: "" },
  { label: "未读", value: "unread" },
  { label: "已读", value: "read" },
  { label: "已归档", value: "archived" },
];

const STATUS_BADGE_TONE: Record<string, "info" | "neutral" | "warning"> = {
  unread: "info",
  read: "neutral",
  archived: "warning",
};

const STATUS_BADGE_LABEL: Record<string, string> = {
  unread: "未读",
  read: "已读",
  archived: "已归档",
};

const PAGE_SIZE = 20;

export default function AdminFormSubmissionsPage() {
  useDocumentTitle("表单提交");
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("");
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const { confirm, confirmDialog } = useAdminConfirm();

  const { data, error, loading, refetch } = useAdminQuery(
    [...adminQueryKeys.formSubmissions, "list", page, PAGE_SIZE, statusFilter],
    () => getFormSubmissions(page, PAGE_SIZE, undefined, statusFilter || undefined),
  );

  const { data: countsData, refetch: refetchCounts } = useAdminQuery(
    [...adminQueryKeys.formSubmissions, "counts"],
    async () => {
      const result = await getSubmissionCounts();
      return result.counts as Record<string, number>;
    },
    { staleTime: 15_000 },
  );
  const counts = countsData ?? {};

  const fetchData = useCallback(async () => {
    setActionError(null);
    invalidateAdminQueryPrefix(adminQueryKeys.formSubmissions);
    await Promise.all([refetch({ force: true }), refetchCounts({ force: true })]);
  }, [refetch, refetchCounts]);

  const fetchCounts = useCallback(async () => {
    await refetchCounts({ force: true });
  }, [refetchCounts]);

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0;

  const handleTabChange = (value: StatusFilter) => {
    setStatusFilter(value);
    setPage(1);
    setSelectedIds(new Set());
    setExpandedId(null);
  };

  const handleToggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleSelectAll = () => {
    if (!data) return;
    if (selectedIds.size === data.items.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(data.items.map((item) => item.id)));
    }
  };

  const handleStatusChange = async (id: number, status: "unread" | "read" | "archived") => {
    try {
      await updateSubmissionStatus(id, status);
      await fetchData();
      await fetchCounts();
    } catch {
      setActionError("更新状态失败");
    }
  };

  const handleBulkStatus = async (status: "unread" | "read" | "archived") => {
    if (selectedIds.size === 0) return;
    try {
      await bulkUpdateStatus(Array.from(selectedIds), status);
      setSelectedIds(new Set());
      await fetchData();
      await fetchCounts();
    } catch {
      setActionError("批量更新状态失败");
    }
  };

  const handleDelete = async (id: number) => {
    const ok = await confirm({
      title: "删除提交记录",
      message: "确认删除此提交记录？此操作不可撤销。",
      confirmLabel: "删除",
      danger: true,
    });
    if (!ok) return;
    try {
      await deleteFormSubmission(id);
      await fetchData();
      await fetchCounts();
    } catch {
      setActionError("删除失败");
    }
  };

  const handleToggleExpand = (id: number) => {
    setExpandedId((prev) => (prev === id ? null : id));
  };

  const formatTime = (dateStr: string) => {
    return new Date(dateStr).toLocaleString("zh-CN");
  };

  return (
    <div>
      {confirmDialog}
      <AdminPageHeader
        title="表单提交"
        description="查看与处理站点表单线索"
        actions={
          <AdminButton
            size="sm"
            onClick={() => {
              fetchData();
              fetchCounts();
            }}
            disabled={loading}
          >
            {loading ? "加载中…" : "刷新"}
          </AdminButton>
        }
      />

      {/* Status Tabs */}
      <div className="mb-6 flex gap-1 border-b border-gray-200">
        {STATUS_TABS.map((tab) => {
          const isActive = statusFilter === tab.value;
          const unreadCount = tab.value === "unread" ? (counts.unread || 0) : 0;
          return (
            <button
              key={tab.value}
              onClick={() => handleTabChange(tab.value)}
              className={`relative px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                isActive
                  ? "border-blue-600 text-blue-600"
                  : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
              }`}
            >
              {tab.label}
              {tab.value === "unread" && unreadCount > 0 && (
                <span className="ml-1.5 inline-flex items-center justify-center px-1.5 py-0.5 text-xs font-bold leading-none text-white bg-red-500 rounded-full">
                  {unreadCount}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Bulk Actions */}
      {selectedIds.size > 0 && (
        <div className="mb-4 flex items-center gap-3 p-3 bg-blue-50 border border-blue-200 rounded-md">
          <span className="text-sm text-blue-700">
            已选 {selectedIds.size} 项
          </span>
          <button
            onClick={() => handleBulkStatus("read")}
            className="px-3 py-1 text-xs font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            标为已读
          </button>
          <button
            onClick={() => handleBulkStatus("unread")}
            className="px-3 py-1 text-xs font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            标为未读
          </button>
          <button
            onClick={() => handleBulkStatus("archived")}
            className="px-3 py-1 text-xs font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            归档
          </button>
        </div>
      )}

      {(actionError || error) && (
        <AdminErrorBanner
          message={actionError || error?.message || "获取表单提交列表失败，请稍后重试"}
          onDismiss={() => setActionError(null)}
        />
      )}

      {loading && !data ? (
        <AdminLoading />
      ) : data ? (
        <>
          <AdminTable>
            <AdminTableHead>
              <tr>
                <AdminTh>
                  <input
                    type="checkbox"
                    checked={data.items.length > 0 && selectedIds.size === data.items.length}
                    onChange={handleSelectAll}
                    className="rounded border-gray-300"
                  />
                </AdminTh>
                <AdminTh>姓名</AdminTh>
                <AdminTh>邮箱</AdminTh>
                <AdminTh>类型</AdminTh>
                <AdminTh>状态</AdminTh>
                <AdminTh>提交时间</AdminTh>
                <AdminTh>操作</AdminTh>
              </tr>
            </AdminTableHead>
            <AdminTableBody>
              {data.items.map((item: FormSubmission) => {
                const isExpanded = expandedId === item.id;
                return (
                  <Fragment key={item.id}>
                    <tr
                      className={`hover:bg-slate-50/80 cursor-pointer ${
                        item.status === "unread" ? "font-medium" : ""
                      }`}
                      onClick={() => handleToggleExpand(item.id)}
                    >
                      <AdminTd>
                        <span onClick={(e) => e.stopPropagation()}>
                          <input
                            type="checkbox"
                            checked={selectedIds.has(item.id)}
                            onChange={() => handleToggleSelect(item.id)}
                            className="rounded border-gray-300"
                          />
                        </span>
                      </AdminTd>
                      <AdminTd className="text-slate-900">{item.name}</AdminTd>
                      <AdminTd className="text-slate-700">{item.email}</AdminTd>
                      <AdminTd className="text-slate-700">{item.formType}</AdminTd>
                      <AdminTd>
                        <AdminBadge tone={STATUS_BADGE_TONE[item.status] || "neutral"}>
                          {STATUS_BADGE_LABEL[item.status] || item.status}
                        </AdminBadge>
                      </AdminTd>
                      <AdminTd className="whitespace-nowrap text-slate-500">
                        {formatTime(item.createdAt)}
                      </AdminTd>
                      <AdminTd>
                        <div
                          className="flex items-center gap-2"
                          onClick={(e) => e.stopPropagation()}
                        >
                          {item.status === "unread" ? (
                            <button
                              onClick={() => handleStatusChange(item.id, "read")}
                              className="text-blue-600 hover:text-blue-800 text-xs"
                              title="标为已读"
                            >
                              已读
                            </button>
                          ) : item.status === "read" ? (
                            <button
                              onClick={() => handleStatusChange(item.id, "unread")}
                              className="text-blue-600 hover:text-blue-800 text-xs"
                              title="标为未读"
                            >
                              未读
                            </button>
                          ) : null}
                          {item.status !== "archived" && (
                            <button
                              onClick={() => handleStatusChange(item.id, "archived")}
                              className="text-yellow-600 hover:text-yellow-800 text-xs"
                              title="归档"
                            >
                              归档
                            </button>
                          )}
                          <button
                            onClick={() => handleDelete(item.id)}
                            className="text-red-600 hover:text-red-800 text-xs"
                            title="删除"
                          >
                            删除
                          </button>
                        </div>
                      </AdminTd>
                    </tr>
                    {isExpanded && (
                      <tr>
                        <AdminTd colSpan={7} className="bg-slate-50">
                          <div className="space-y-2 text-sm">
                            {item.phone && (
                              <div>
                                <span className="font-medium text-slate-700">电话：</span>
                                <span className="text-slate-600">{item.phone}</span>
                              </div>
                            )}
                            {item.company && (
                              <div>
                                <span className="font-medium text-slate-700">公司：</span>
                                <span className="text-slate-600">{item.company}</span>
                              </div>
                            )}
                            {item.message && (
                              <div>
                                <span className="font-medium text-slate-700">留言：</span>
                                <p className="text-slate-600 mt-1 whitespace-pre-wrap">{item.message}</p>
                              </div>
                            )}
                            {item.sourceUrl && (
                              <div>
                                <span className="font-medium text-slate-700">来源页面：</span>
                                <span className="text-slate-600">{item.sourceUrl}</span>
                              </div>
                            )}
                            {item.locale && (
                              <div>
                                <span className="font-medium text-slate-700">语言：</span>
                                <span className="text-slate-600">{item.locale}</span>
                              </div>
                            )}
                            {item.ipAddress && (
                              <div>
                                <span className="font-medium text-slate-700">IP：</span>
                                <span className="text-slate-600">{item.ipAddress}</span>
                              </div>
                            )}
                            {item.metadata && Object.keys(item.metadata).length > 0 && (
                              <div>
                                <span className="font-medium text-slate-700">元数据：</span>
                                <pre className="mt-1 p-2 bg-white border border-slate-200 rounded text-xs text-slate-600 overflow-x-auto">
                                  {JSON.stringify(item.metadata, null, 2)}
                                </pre>
                              </div>
                            )}
                          </div>
                        </AdminTd>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
              {data.items.length === 0 && (
                <tr>
                  <AdminTd colSpan={7} className="py-8 text-center text-slate-500">
                    暂无表单提交记录
                  </AdminTd>
                </tr>
              )}
            </AdminTableBody>
          </AdminTable>

          <AdminPagination
            page={page}
            totalPages={totalPages}
            total={data.total}
            onPageChange={setPage}
          />
        </>
      ) : null}
    </div>
  );
}
