import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  listUnifiedPages,
  deleteUnifiedPage,
  type UnifiedPageItem,
} from "@/api/unifiedPages";
import {
  AdminBadge,
  AdminButton,
  AdminEmptyState,
  AdminLoading,
  AdminPageHeader,
  AdminTable,
  AdminTableBody,
  AdminTableHead,
  AdminTd,
  AdminTh,
  useAdminConfirm,
} from "@/components/admin/ui";
import { useAuth } from "@/contexts/AuthContext";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { invalidateAdminQueryPrefix, useAdminQuery } from "@/lib/adminQuery";
import { adminQueryKeys } from "@/lib/adminQueryKeys";
import { prefetchAdminEditors } from "@/pages/admin/adminRoutePrefetch";

export default function AdminPagesPage() {
  useDocumentTitle("页面管理");
  const [statusFilter, setStatusFilter] = useState("");
  const [modeFilter, setModeFilter] = useState("");
  const { confirm, confirmDialog } = useAdminConfirm();

  const navigate = useNavigate();

  useEffect(() => {
    prefetchAdminEditors();
  }, []);
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("pages:create");
  const canUpdate = hasPermission("pages:update");
  const canDelete = hasPermission("pages:delete");

  const { data, loading, isFetching, refetch } = useAdminQuery(
    [...adminQueryKeys.pages, statusFilter, modeFilter],
    async () => {
      const list = await listUnifiedPages(
        statusFilter || undefined,
        modeFilter || undefined,
      );
      return list ?? [];
    },
  );

  const pages: UnifiedPageItem[] = data ?? [];

  const handleDelete = async (id: number) => {
    const ok = await confirm({
      title: "删除页面",
      message: "确定删除此页面？此操作不可撤销。",
      confirmLabel: "删除",
      danger: true,
    });
    if (!ok) return;
    try {
      await deleteUnifiedPage(id);
      invalidateAdminQueryPrefix(adminQueryKeys.pages);
      invalidateAdminQueryPrefix(adminQueryKeys.dashboardStats);
      await refetch({ force: true });
    } catch {
      // ignore
    }
  };

  return (
    <div>
      {confirmDialog}
      <AdminPageHeader
        title="页面管理"
        description={
          loading
            ? "管理可视化页面与区块组合"
            : `管理可视化页面与区块组合${isFetching ? " · 刷新中" : ""}`
        }
        actions={
          <>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700"
            >
              <option value="">全部状态</option>
              <option value="draft">草稿</option>
              <option value="published">已发布</option>
            </select>
            <select
              value={modeFilter}
              onChange={(e) => setModeFilter(e.target.value)}
              className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700"
            >
              <option value="">全部模式</option>
              <option value="template">模板</option>
              <option value="composable">自由组合</option>
            </select>
            {canCreate && (
              <AdminButton size="sm" onClick={() => navigate("/admin/pages/new")}>
                新建页面
              </AdminButton>
            )}
          </>
        }
      />

      {loading ? (
        <AdminLoading />
      ) : pages.length === 0 ? (
        <AdminEmptyState
          title="暂无页面"
          description="创建页面，或调整筛选条件。"
          action={
            canCreate ? (
              <AdminButton size="sm" onClick={() => navigate("/admin/pages/new")}>
                新建页面
              </AdminButton>
            ) : undefined
          }
        />
      ) : (
        <AdminTable>
          <AdminTableHead>
            <tr>
              <AdminTh>标题</AdminTh>
              <AdminTh>路径</AdminTh>
              <AdminTh>模式</AdminTh>
              <AdminTh>状态</AdminTh>
              <AdminTh>草稿版本</AdminTh>
              <AdminTh>发布版本</AdminTh>
              <AdminTh className="text-right">操作</AdminTh>
            </tr>
          </AdminTableHead>
          <AdminTableBody>
            {pages.map((page) => (
              <tr key={page.id} className="hover:bg-slate-50/80">
                <AdminTd className="font-medium text-slate-900">
                  {page.zhTitle || page.enTitle || "（无标题）"}
                </AdminTd>
                <AdminTd className="font-mono text-slate-500">/{page.slug}</AdminTd>
                <AdminTd>
                  <AdminBadge tone={page.mode === "template" ? "info" : "neutral"}>
                    {page.mode === "template" ? "模板" : "自由组合"}
                  </AdminBadge>
                </AdminTd>
                <AdminTd>
                  <AdminBadge tone={page.status === "published" ? "success" : "warning"}>
                    {page.status === "published" ? "已发布" : "草稿"}
                  </AdminBadge>
                </AdminTd>
                <AdminTd className="text-slate-500">{page.draftVersion}</AdminTd>
                <AdminTd className="text-slate-500">
                  {page.publishedVersion > 0 ? page.publishedVersion : "—"}
                </AdminTd>
                <AdminTd className="space-x-2 text-right">
                  <button
                    type="button"
                    onClick={() => navigate(`/admin/pages/edit/${page.id}`)}
                    onMouseEnter={() => prefetchAdminEditors()}
                    onFocus={() => prefetchAdminEditors()}
                    className="text-sm font-medium text-blue-600 hover:text-blue-800"
                  >
                    {canUpdate ? "编辑" : "查看"}
                  </button>
                  {canDelete && (
                    <button
                      type="button"
                      onClick={() => handleDelete(page.id)}
                      className="text-sm font-medium text-red-600 hover:text-red-800"
                    >
                      删除
                    </button>
                  )}
                </AdminTd>
              </tr>
            ))}
          </AdminTableBody>
        </AdminTable>
      )}
    </div>
  );
}
