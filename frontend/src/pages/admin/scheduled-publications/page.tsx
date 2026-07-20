import { useCallback, useMemo, useState } from "react";
import {
  cancelScheduledPublication,
  listScheduledPublications,
  retryScheduledPublication,
  type ScheduledPublication,
  type ScheduledPublicationResourceType,
  type ScheduledPublicationStatus,
} from "@/api/scheduledPublications";
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
} from "@/components/admin/ui";
import { useAuth } from "@/contexts/AuthContext";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { invalidateAdminQueryPrefix, useAdminQuery } from "@/lib/adminQuery";
import { adminQueryKeys } from "@/lib/adminQueryKeys";

const statusOptions: Array<{ value: ScheduledPublicationStatus | ""; label: string }> = [
  { value: "pending", label: "等待发布" },
  { value: "failed", label: "发布失败" },
  { value: "running", label: "发布中" },
  { value: "succeeded", label: "已发布" },
  { value: "cancelled", label: "已取消" },
  { value: "", label: "全部" },
];

const resourceOptions: Array<{ value: ScheduledPublicationResourceType | ""; label: string }> = [
  { value: "", label: "全部类型" },
  { value: "page", label: "页面" },
  { value: "article", label: "文章" },
];

const statusBadgeTone: Record<ScheduledPublicationStatus, "info" | "warning" | "success" | "danger" | "neutral"> = {
  pending: "info",
  running: "info",
  succeeded: "success",
  failed: "danger",
  cancelled: "neutral",
};

const statusLabels: Record<ScheduledPublicationStatus, string> = {
  pending: "等待发布",
  running: "发布中",
  succeeded: "已发布",
  failed: "发布失败",
  cancelled: "已取消",
};

function formatDateTime(value: string) {
  try {
    return new Date(value).toLocaleString("zh-CN");
  } catch {
    return value;
  }
}

export default function AdminScheduledPublicationsPage() {
  useDocumentTitle("定时发布");
  const { hasPermission } = useAuth();
  const canPublishPages = hasPermission("pages:publish");
  const canPublishArticles = hasPermission("articles:publish");

  const [statusFilter, setStatusFilter] = useState<ScheduledPublicationStatus | "">("pending");
  const [resourceFilter, setResourceFilter] = useState<ScheduledPublicationResourceType | "">("");
  const [page, setPage] = useState(1);
  const [busyId, setBusyId] = useState<number | null>(null);
  const [actionError, setActionError] = useState("");
  const pageSize = 20;

  const { data, error, loading, refetch } = useAdminQuery(
    [...adminQueryKeys.scheduled, page, pageSize, statusFilter, resourceFilter],
    () =>
      listScheduledPublications({
        page,
        pageSize,
        status: statusFilter,
        resourceType: resourceFilter,
      }),
  );
  const items = data?.items ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const displayError = actionError || (error ? error.message : "");

  const canManageResource = useCallback(
    (resourceType: ScheduledPublicationResourceType) =>
      resourceType === "page" ? canPublishPages : canPublishArticles,
    [canPublishArticles, canPublishPages],
  );

  const loadQueue = useCallback(async () => {
    setActionError("");
    invalidateAdminQueryPrefix(adminQueryKeys.scheduled);
    await refetch({ force: true });
  }, [refetch]);

  const summaryText = useMemo(() => {
    const statusLabel = statusOptions.find((option) => option.value === statusFilter)?.label ?? "全部";
    const resourceLabel = resourceOptions.find((option) => option.value === resourceFilter)?.label ?? "全部类型";
    return `${resourceLabel} / ${statusLabel} / ${total} 条`;
  }, [resourceFilter, statusFilter, total]);

  const handleCancel = async (item: ScheduledPublication) => {
    if (!canManageResource(item.resourceType)) return;
    setBusyId(item.id);
    setActionError("");
    try {
      await cancelScheduledPublication(item.id);
      await loadQueue();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "取消定时发布失败");
    } finally {
      setBusyId(null);
    }
  };

  const handleRetry = async (item: ScheduledPublication) => {
    if (!canManageResource(item.resourceType)) return;
    setBusyId(item.id);
    setActionError("");
    try {
      await retryScheduledPublication(item.id);
      await loadQueue();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "重试定时发布失败");
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div>
      <AdminPageHeader
        title="定时发布"
        description={summaryText}
        actions={
          <AdminButton variant="secondary" size="sm" onClick={loadQueue} disabled={loading}>
            刷新
          </AdminButton>
        }
      />

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <select
          value={resourceFilter}
          onChange={(event) => {
            setResourceFilter(event.target.value as ScheduledPublicationResourceType | "");
            setPage(1);
          }}
          className="rounded-md border border-gray-300 px-3 py-2 text-sm"
        >
          {resourceOptions.map((option) => (
            <option key={option.value || "all"} value={option.value}>{option.label}</option>
          ))}
        </select>
        <div className="flex flex-wrap items-center gap-2">
          {statusOptions.map((option) => (
            <button
              key={option.value || "all"}
              type="button"
              onClick={() => {
                setStatusFilter(option.value);
                setPage(1);
              }}
              className={`rounded-md border px-3 py-1.5 text-sm ${
                statusFilter === option.value
                  ? "border-blue-300 bg-blue-50 text-blue-700"
                  : "border-gray-300 text-gray-600 hover:bg-gray-50"
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      {displayError && (
        <AdminErrorBanner message={displayError} onDismiss={() => setActionError("")} />
      )}

      {loading ? (
        <AdminLoading />
      ) : items.length === 0 ? (
        <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50/80 py-16 text-center text-sm text-slate-500">
          暂无定时发布任务
        </div>
      ) : (
        <AdminTable>
          <AdminTableHead>
            <tr>
              <AdminTh>内容</AdminTh>
              <AdminTh>类型</AdminTh>
              <AdminTh>计划时间</AdminTh>
              <AdminTh>状态</AdminTh>
              <AdminTh>失败信息</AdminTh>
              <AdminTh className="text-right">操作</AdminTh>
            </tr>
          </AdminTableHead>
          <AdminTableBody>
            {items.map((item) => {
              const canManage = canManageResource(item.resourceType);
              return (
                <tr key={item.id} className="hover:bg-slate-50/80">
                  <AdminTd>
                    <div className="max-w-xs truncate text-sm font-medium text-slate-900">
                      {item.title || `#${item.resourceId}`}
                    </div>
                    <div className="max-w-xs truncate font-mono text-xs text-slate-500">
                      {item.slug ? `/${item.slug}` : `ID ${item.resourceId}`}
                    </div>
                  </AdminTd>
                  <AdminTd className="text-slate-600">
                    {item.resourceType === "page" ? "页面" : "文章"}
                  </AdminTd>
                  <AdminTd className="whitespace-nowrap text-slate-700">
                    {formatDateTime(item.scheduledAt)}
                  </AdminTd>
                  <AdminTd>
                    <AdminBadge tone={statusBadgeTone[item.status]}>
                      {statusLabels[item.status]}
                    </AdminBadge>
                  </AdminTd>
                  <AdminTd className="text-red-700">
                    <div className="max-w-sm truncate" title={item.lastError ?? undefined}>
                      {item.lastError || "-"}
                    </div>
                  </AdminTd>
                  <AdminTd className="text-right">
                    {item.status === "pending" && canManage && (
                      <button
                        type="button"
                        onClick={() => handleCancel(item)}
                        disabled={busyId === item.id}
                        className="text-sm font-medium text-red-600 hover:text-red-800 disabled:opacity-50"
                      >
                        {busyId === item.id ? "处理中..." : "取消"}
                      </button>
                    )}
                    {item.status === "failed" && canManage && (
                      <button
                        type="button"
                        onClick={() => handleRetry(item)}
                        disabled={busyId === item.id}
                        className="text-sm font-medium text-orange-700 hover:text-orange-900 disabled:opacity-50"
                      >
                        {busyId === item.id ? "处理中..." : "重试"}
                      </button>
                    )}
                    {!canManage && <span className="text-slate-400">无权限</span>}
                  </AdminTd>
                </tr>
              );
            })}
          </AdminTableBody>
        </AdminTable>
      )}

      <AdminPagination
        page={page}
        totalPages={totalPages}
        total={total}
        onPageChange={setPage}
      />
    </div>
  );
}
