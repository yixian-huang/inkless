import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { getAdminArticles, deleteArticle } from "@/api/articles";
import type { Article } from "@/api/articles";
import {
  AdminBadge,
  AdminButton,
  AdminEmptyState,
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
import { prefetchAdminEditors } from "@/pages/admin/adminRoutePrefetch";

const PAGE_SIZE = 15;

export default function AdminArticlesPage() {
  useDocumentTitle("文章管理");
  const navigate = useNavigate();
  const { confirm, confirmDialog } = useAdminConfirm();

  useEffect(() => {
    prefetchAdminEditors();
  }, []);

  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [deleting, setDeleting] = useState<number | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const { data, error, loading, isFetching, refetch } = useAdminQuery(
    [...adminQueryKeys.articles, page, PAGE_SIZE, statusFilter],
    () => getAdminArticles(page, PAGE_SIZE, statusFilter || undefined),
  );

  const articles = data?.items ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / PAGE_SIZE);
  const displayError = actionError ?? (error ? error.message : null);

  const handleDelete = async (article: Article) => {
    const title = article.zhTitle || article.enTitle || "未命名";
    const ok = await confirm({
      title: "删除文章",
      message: `确定删除「${title}」吗？此操作不可撤销。`,
      confirmLabel: "删除",
      danger: true,
    });
    if (!ok) return;

    setDeleting(article.id);
    setActionError(null);
    try {
      await deleteArticle(article.id);
      invalidateAdminQueryPrefix(adminQueryKeys.articles);
      invalidateAdminQueryPrefix(adminQueryKeys.dashboardStats);
      await refetch({ force: true });
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "删除失败");
    } finally {
      setDeleting(null);
    }
  };

  const statusLabel = (status: string) => {
    switch (status) {
      case "published":
        return <AdminBadge tone="success">已发布</AdminBadge>;
      case "draft":
        return <AdminBadge tone="warning">草稿</AdminBadge>;
      default:
        return <AdminBadge>{status}</AdminBadge>;
    }
  };

  return (
    <div>
      {confirmDialog}
      <AdminPageHeader
        title="文章管理"
        description={
          loading
            ? "加载中…"
            : `共 ${total} 篇文章${isFetching ? " · 刷新中" : ""}`
        }
        actions={
          <>
            <AdminButton
              variant="secondary"
              size="sm"
              onClick={() => navigate("/admin/articles/categories")}
            >
              分类
            </AdminButton>
            <AdminButton
              variant="secondary"
              size="sm"
              onClick={() => navigate("/admin/articles/tags")}
            >
              标签
            </AdminButton>
            <AdminButton size="sm" onClick={() => navigate("/admin/articles/new")}>
              新建文章
            </AdminButton>
          </>
        }
      />

      {displayError && (
        <AdminErrorBanner message={displayError} onDismiss={() => setActionError(null)} />
      )}

      <div className="mb-4 flex flex-wrap items-center gap-2">
        <span className="text-sm text-slate-500">状态：</span>
        {[
          { value: "", label: "全部" },
          { value: "draft", label: "草稿" },
          { value: "published", label: "已发布" },
        ].map((opt) => (
          <button
            key={opt.value}
            type="button"
            onClick={() => {
              setStatusFilter(opt.value);
              setPage(1);
            }}
            className={`rounded-lg border px-3 py-1.5 text-sm transition-colors ${
              statusFilter === opt.value
                ? "border-blue-300 bg-blue-50 text-blue-700"
                : "border-slate-200 text-slate-600 hover:bg-slate-50"
            }`}
          >
            {opt.label}
          </button>
        ))}
      </div>

      {loading ? (
        <AdminLoading />
      ) : articles.length === 0 ? (
        <AdminEmptyState
          title="暂无文章"
          description="创建第一篇文章，或调整筛选条件。"
          action={
            <AdminButton size="sm" onClick={() => navigate("/admin/articles/new")}>
              新建文章
            </AdminButton>
          }
        />
      ) : (
        <>
          <AdminTable>
            <AdminTableHead>
              <tr>
                <AdminTh>标题</AdminTh>
                <AdminTh>状态</AdminTh>
                <AdminTh>分类</AdminTh>
                <AdminTh>创建时间</AdminTh>
                <AdminTh className="text-right">操作</AdminTh>
              </tr>
            </AdminTableHead>
            <AdminTableBody>
              {articles.map((article) => (
                <tr key={article.id} className="hover:bg-slate-50/80">
                  <AdminTd>
                    <div className="max-w-xs truncate text-sm font-medium text-slate-900">
                      {article.zhTitle || article.enTitle || "（未命名）"}
                    </div>
                    <div className="max-w-xs truncate text-xs text-slate-500">/{article.slug}</div>
                  </AdminTd>
                  <AdminTd className="whitespace-nowrap">{statusLabel(article.status)}</AdminTd>
                  <AdminTd className="whitespace-nowrap text-slate-600">
                    {article.category?.zhName || article.category?.enName || "—"}
                  </AdminTd>
                  <AdminTd className="whitespace-nowrap text-slate-500">
                    {new Date(article.createdAt).toLocaleDateString("zh-CN")}
                  </AdminTd>
                  <AdminTd className="whitespace-nowrap text-right">
                    <button
                      type="button"
                      onClick={() => navigate(`/admin/articles/edit/${article.id}`)}
                      onMouseEnter={() => prefetchAdminEditors()}
                      onFocus={() => prefetchAdminEditors()}
                      className="mr-3 text-sm font-medium text-blue-600 hover:text-blue-800"
                    >
                      编辑
                    </button>
                    <button
                      type="button"
                      onClick={() => handleDelete(article)}
                      disabled={deleting === article.id}
                      className="text-sm font-medium text-red-600 hover:text-red-800 disabled:opacity-50"
                    >
                      {deleting === article.id ? "…" : "删除"}
                    </button>
                  </AdminTd>
                </tr>
              ))}
            </AdminTableBody>
          </AdminTable>

          <AdminPagination
            page={page}
            totalPages={totalPages}
            total={total}
            onPageChange={setPage}
          />
        </>
      )}
    </div>
  );
}
