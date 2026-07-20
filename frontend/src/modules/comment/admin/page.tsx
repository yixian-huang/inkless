import { useState, Fragment } from "react";
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
import AdminCommentReplyPanel from "./AdminCommentReplyPanel";
import {
  adminCommentAuthorName,
  adminCommentCreatedAt,
  approveComment,
  deleteComment,
  getAdminComments,
  rejectComment,
  type AdminComment,
} from "./api";

type StatusFilter = "" | "pending" | "approved" | "rejected";

const STATUS_TABS: { label: string; value: StatusFilter }[] = [
  { label: "全部", value: "" },
  { label: "待审核", value: "pending" },
  { label: "已通过", value: "approved" },
  { label: "已拒绝", value: "rejected" },
];

const STATUS_BADGE_TONE: Record<string, "warning" | "success" | "danger" | "neutral"> = {
  pending: "warning",
  approved: "success",
  rejected: "danger",
};

const STATUS_BADGE_LABEL: Record<string, string> = {
  pending: "待审核",
  approved: "已通过",
  rejected: "已拒绝",
};

const PAGE_SIZE = 20;

export default function AdminCommentsPage() {
  useDocumentTitle("评论管理");
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("");
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [actionLoading, setActionLoading] = useState(false);
  const [replyTarget, setReplyTarget] = useState<AdminComment | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const { confirm, confirmDialog } = useAdminConfirm();

  const { data, error, loading, isFetching, refetch } = useAdminQuery(
    [...adminQueryKeys.comments, page, PAGE_SIZE, statusFilter],
    () => getAdminComments(page, PAGE_SIZE, statusFilter || undefined),
  );

  const fetchData = async () => {
    setActionError(null);
    invalidateAdminQueryPrefix(adminQueryKeys.comments);
    await refetch({ force: true });
  };

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0;

  const handleTabChange = (value: StatusFilter) => {
    setStatusFilter(value);
    setPage(1);
    setSelectedIds(new Set());
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
    if (selectedIds.size === data.comments.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(data.comments.map((item) => item.id)));
    }
  };

  const handleApprove = async (id: number) => {
    setActionLoading(true);
    try {
      await approveComment(id);
      await fetchData();
    } catch {
      setActionError("操作失败，请稍后重试");
    } finally {
      setActionLoading(false);
    }
  };

  const handleReject = async (id: number) => {
    setActionLoading(true);
    try {
      await rejectComment(id);
      await fetchData();
    } catch {
      setActionError("操作失败，请稍后重试");
    } finally {
      setActionLoading(false);
    }
  };

  const handleDelete = async (id: number) => {
    const ok = await confirm({
      title: "删除评论",
      message: "确认删除此评论？此操作不可撤销。",
      confirmLabel: "删除",
      danger: true,
    });
    if (!ok) return;
    setActionLoading(true);
    try {
      await deleteComment(id);
      setSelectedIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
      await fetchData();
    } catch {
      setActionError("删除失败，请稍后重试");
    } finally {
      setActionLoading(false);
    }
  };

  const handleBulkApprove = async () => {
    if (selectedIds.size === 0) return;
    setActionLoading(true);
    try {
      await Promise.all(Array.from(selectedIds).map((id) => approveComment(id)));
      setSelectedIds(new Set());
      await fetchData();
    } catch {
      setActionError("批量操作失败，请稍后重试");
    } finally {
      setActionLoading(false);
    }
  };

  const handleBulkReject = async () => {
    if (selectedIds.size === 0) return;
    setActionLoading(true);
    try {
      await Promise.all(Array.from(selectedIds).map((id) => rejectComment(id)));
      setSelectedIds(new Set());
      await fetchData();
    } catch {
      setActionError("批量操作失败，请稍后重试");
    } finally {
      setActionLoading(false);
    }
  };

  const handleBulkDelete = async () => {
    if (selectedIds.size === 0) return;
    const ok = await confirm({
      title: "批量删除评论",
      message: `确认删除选中的 ${selectedIds.size} 条评论？此操作不可撤销。`,
      confirmLabel: "删除",
      danger: true,
    });
    if (!ok) return;
    setActionLoading(true);
    try {
      await Promise.all(Array.from(selectedIds).map((id) => deleteComment(id)));
      setSelectedIds(new Set());
      await fetchData();
    } catch {
      setActionError("批量删除失败，请稍后重试");
    } finally {
      setActionLoading(false);
    }
  };

  const formatTime = (dateStr: string) => {
    return new Date(dateStr).toLocaleString("zh-CN");
  };

  const truncate = (text: string, maxLen = 80) => {
    if (text.length <= maxLen) return text;
    return text.slice(0, maxLen) + "…";
  };

  return (
    <div>
      {confirmDialog}
      <AdminPageHeader
        title="评论管理"
        description="审核与回复访客评论"
        actions={
          <AdminButton size="sm" onClick={fetchData} disabled={isFetching}>
            {isFetching ? "加载中…" : "刷新"}
          </AdminButton>
        }
      />

      <div className="mb-6 flex gap-1 border-b border-gray-200">
        {STATUS_TABS.map((tab) => {
          const isActive = statusFilter === tab.value;
          return (
            <button
              key={tab.value}
              type="button"
              onClick={() => handleTabChange(tab.value)}
              className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                isActive
                  ? "border-blue-600 text-blue-600"
                  : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
              }`}
            >
              {tab.label}
            </button>
          );
        })}
      </div>

      {selectedIds.size > 0 && (
        <div className="mb-4 flex items-center gap-3 p-3 bg-blue-50 border border-blue-200 rounded-md">
          <span className="text-sm text-blue-700">已选 {selectedIds.size} 项</span>
          <button
            type="button"
            onClick={handleBulkApprove}
            disabled={actionLoading}
            className="px-3 py-1 text-xs font-medium text-green-700 bg-white border border-green-300 rounded-md hover:bg-green-50 disabled:opacity-50"
          >
            批量通过
          </button>
          <button
            type="button"
            onClick={handleBulkReject}
            disabled={actionLoading}
            className="px-3 py-1 text-xs font-medium text-yellow-700 bg-white border border-yellow-300 rounded-md hover:bg-yellow-50 disabled:opacity-50"
          >
            批量拒绝
          </button>
          <button
            type="button"
            onClick={handleBulkDelete}
            disabled={actionLoading}
            className="px-3 py-1 text-xs font-medium text-red-700 bg-white border border-red-300 rounded-md hover:bg-red-50 disabled:opacity-50"
          >
            批量删除
          </button>
        </div>
      )}

      {(actionError || error) && (
        <AdminErrorBanner
          message={actionError || error?.message || "获取评论列表失败，请稍后重试"}
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
                    checked={data.comments.length > 0 && selectedIds.size === data.comments.length}
                    onChange={handleSelectAll}
                    className="rounded border-gray-300"
                  />
                </AdminTh>
                <AdminTh>作者</AdminTh>
                <AdminTh>内容</AdminTh>
                <AdminTh>文章</AdminTh>
                <AdminTh>状态</AdminTh>
                <AdminTh>时间</AdminTh>
                <AdminTh>操作</AdminTh>
              </tr>
            </AdminTableHead>
            <AdminTableBody>
              {data.comments.map((item: AdminComment) => (
                <Fragment key={item.id}>
                  <tr className="hover:bg-slate-50/80">
                    <AdminTd>
                      <input
                        type="checkbox"
                        checked={selectedIds.has(item.id)}
                        onChange={() => handleToggleSelect(item.id)}
                        className="rounded border-gray-300"
                      />
                    </AdminTd>
                    <AdminTd className="text-slate-900">
                      <div className="font-medium">
                        {adminCommentAuthorName(item)}
                        {item.authorRole === "author" && (
                          <span className="ml-1 text-xs text-blue-600">作者</span>
                        )}
                      </div>
                      {(item.authorEmail || item.author_email) && (
                        <div className="text-xs text-slate-500 mt-0.5">
                          {item.authorEmail || item.author_email}
                        </div>
                      )}
                    </AdminTd>
                    <AdminTd className="max-w-xs text-slate-700">
                      {truncate(item.content)}
                    </AdminTd>
                    <AdminTd className="text-slate-500">
                      {item.article_title ? (
                        <span className="text-xs">{truncate(item.article_title, 40)}</span>
                      ) : item.article_id ? (
                        <span className="text-xs text-slate-400">文章 #{item.article_id}</span>
                      ) : (
                        <span className="text-xs text-slate-400">—</span>
                      )}
                    </AdminTd>
                    <AdminTd>
                      <AdminBadge tone={STATUS_BADGE_TONE[item.status] || "neutral"}>
                        {STATUS_BADGE_LABEL[item.status] || item.status}
                      </AdminBadge>
                    </AdminTd>
                    <AdminTd className="whitespace-nowrap text-slate-500">
                      {formatTime(adminCommentCreatedAt(item))}
                    </AdminTd>
                    <AdminTd>
                      <div className="flex items-center gap-2 flex-wrap">
                        <button
                          type="button"
                          onClick={() => setReplyTarget(item)}
                          disabled={actionLoading}
                          className="text-blue-600 hover:text-blue-800 text-xs font-medium disabled:opacity-50"
                        >
                          回复
                        </button>
                        {item.status !== "approved" && (
                          <button
                            type="button"
                            onClick={() => handleApprove(item.id)}
                            disabled={actionLoading}
                            className="text-green-600 hover:text-green-800 text-xs font-medium disabled:opacity-50"
                          >
                            通过
                          </button>
                        )}
                        {item.status !== "rejected" && (
                          <button
                            type="button"
                            onClick={() => handleReject(item.id)}
                            disabled={actionLoading}
                            className="text-yellow-600 hover:text-yellow-800 text-xs font-medium disabled:opacity-50"
                          >
                            拒绝
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={() => handleDelete(item.id)}
                          disabled={actionLoading}
                          className="text-red-600 hover:text-red-800 text-xs font-medium disabled:opacity-50"
                        >
                          删除
                        </button>
                      </div>
                    </AdminTd>
                  </tr>
                </Fragment>
              ))}
              {data.comments.length === 0 && (
                <tr>
                  <AdminTd colSpan={7} className="py-8 text-center text-slate-500">
                    暂无评论记录
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

      {replyTarget && (
        <AdminCommentReplyPanel
          comment={replyTarget}
          onClose={() => setReplyTarget(null)}
          onSent={fetchData}
        />
      )}
    </div>
  );
}
