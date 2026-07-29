import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { ExternalLink, RefreshCw, Store } from "lucide-react";
import {
  fetchThemeCatalog,
  installOrUpdateThemeFromCatalog,
  type ThemeCatalogItem,
  type ThemeCatalogResponse,
  type ThemeInstallState,
} from "@/api/extensionsThemes";
import {
  AdminButton,
  AdminCard,
  AdminErrorBanner,
  AdminLoading,
  AdminPageHeader,
  AdminSuccessBanner,
  AdminToolbar,
} from "@/components/admin/ui";
import { useBootstrap } from "@/contexts/BootstrapContext";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

const STATE_LABEL: Record<ThemeInstallState, string> = {
  not_installed: "未安装",
  installed: "已安装",
  active: "使用中",
  builtin: "内置",
  update_available: "可更新",
  incompatible: "不兼容",
};

const STATE_BADGE: Record<ThemeInstallState, string> = {
  not_installed: "bg-slate-100 text-slate-700",
  installed: "bg-blue-50 text-blue-700",
  active: "bg-emerald-50 text-emerald-800",
  builtin: "bg-indigo-50 text-indigo-700",
  update_available: "bg-amber-50 text-amber-800",
  incompatible: "bg-rose-50 text-rose-700",
};

function displayName(item: ThemeCatalogItem): string {
  return item.nameZh || item.name || item.slug;
}

function displayDescription(item: ThemeCatalogItem): string {
  return item.descriptionZh || item.description || "";
}

export default function AdminThemeMarketPage() {
  useDocumentTitle("主题市场");
  const { refetch: refetchBootstrap } = useBootstrap();

  const [catalog, setCatalog] = useState<ThemeCatalogResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [busySlug, setBusySlug] = useState<string | null>(null);

  const load = useCallback(async (refresh = false) => {
    if (refresh) setRefreshing(true);
    else setLoading(true);
    setError("");
    try {
      const data = await fetchThemeCatalog({ refresh });
      setCatalog(data);
    } catch (e: any) {
      setError(e?.response?.data?.error || e?.message || "加载主题目录失败");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    load(false);
  }, [load]);

  const items = useMemo(() => catalog?.items ?? [], [catalog]);

  const counts = useMemo(() => {
    const c: Record<string, number> = {};
    for (const it of items) {
      c[it.installState] = (c[it.installState] || 0) + 1;
    }
    return c;
  }, [items]);

  async function runInstall(item: ThemeCatalogItem, activate: boolean) {
    setBusySlug(item.slug);
    setError("");
    setSuccess("");
    try {
      const res = await installOrUpdateThemeFromCatalog(item.slug, { activate });
      const label = displayName(item);
      if (activate && res.activated) {
        setSuccess(`已安装并激活「${label}」`);
        await refetchBootstrap();
      } else if (res.created) {
        setSuccess(`已安装「${label}」`);
      } else {
        setSuccess(`已更新「${label}」到 v${res.theme.version || item.latest?.version || ""}`);
        if (res.activated) await refetchBootstrap();
      }
      if (res.warning) {
        setError(String(res.warning));
      }
      await load(false);
    } catch (e: any) {
      setError(e?.response?.data?.error || e?.message || "安装失败");
    } finally {
      setBusySlug(null);
    }
  }

  return (
    <div className="mx-auto max-w-6xl">
      <AdminPageHeader
        title="主题市场"
        description="浏览并一键安装官方主题（Phase A：仅 official catalog）"
        breadcrumbs={[
          { label: "外观", to: "/admin/theme" },
          { label: "主题市场" },
        ]}
        actions={
          <div className="flex flex-wrap gap-2">
            <Link
              to="/admin/theme"
              className="inline-flex items-center rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50"
            >
              已安装主题
            </Link>
            <AdminButton
              variant="secondary"
              disabled={refreshing || loading}
              onClick={() => load(true)}
            >
              <RefreshCw className={`mr-1.5 h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
              刷新目录
            </AdminButton>
          </div>
        }
      />

      {error && (
        <div className="mb-4">
          <AdminErrorBanner message={error} />
        </div>
      )}
      {success && (
        <div className="mb-4">
          <AdminSuccessBanner message={success} />
        </div>
      )}

      <AdminToolbar className="mb-4 flex flex-wrap items-center gap-3 text-xs text-slate-500">
        <span className="inline-flex items-center gap-1.5 font-medium text-slate-700">
          <Store className="h-4 w-4" />
          目录来源：{catalog?.source || "—"}
        </span>
        {catalog?.updatedAt && <span>更新于 {catalog.updatedAt}</span>}
        {catalog?.warning && (
          <span className="text-amber-700">注意：{catalog.warning}</span>
        )}
        <span className="text-slate-400">
          {counts.active ? `使用中 ${counts.active}` : null}
          {counts.builtin ? ` · 内置 ${counts.builtin}` : null}
          {counts.not_installed ? ` · 未安装 ${counts.not_installed}` : null}
          {counts.update_available ? ` · 可更新 ${counts.update_available}` : null}
          {counts.incompatible ? ` · 不兼容 ${counts.incompatible}` : null}
        </span>
      </AdminToolbar>

      {loading ? (
        <AdminLoading label="加载官方主题目录…" />
      ) : items.length === 0 ? (
        <AdminCard className="p-8 text-center text-sm text-slate-500">
          目录为空。可检查内嵌 official_themes.json 或远程 INKLESS_THEME_CATALOG_URL。
        </AdminCard>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {items.map((item) => {
            const busy = busySlug === item.slug;
            const state = item.installState;
            const version = item.latest?.version || "—";
            const incompatible = state === "incompatible";
            const isActive = state === "active";
            const isBuiltin = state === "builtin";
            const canInstall =
              !incompatible &&
              (state === "not_installed" ||
                state === "installed" ||
                state === "update_available" ||
                (isBuiltin && !item.builtinOnly && Boolean(item.latest?.umdUrl)));
            const canUpdate = state === "update_available" || Boolean(item.updateAvailable);
            const canActivateOnly = isBuiltin || state === "installed";

            return (
              <AdminCard key={item.slug} className="flex flex-col overflow-hidden p-0">
                <div
                  className="h-24 w-full border-b border-slate-100"
                  style={{
                    background:
                      item.previewUrl ||
                      item.iconUrl ||
                      "linear-gradient(135deg, #0f172a 0%, #14b8a6 100%)",
                  }}
                />
                <div className="flex flex-1 flex-col p-4">
                  <div className="mb-2 flex items-start justify-between gap-2">
                    <div>
                      <h3 className="text-sm font-semibold text-slate-900">
                        {displayName(item)}
                      </h3>
                      <p className="text-xs text-slate-400">
                        {item.author || "Inkless"} · v{version}
                        {item.installedVersion
                          ? ` · 已装 ${item.installedVersion}`
                          : null}
                      </p>
                    </div>
                    <span
                      className={`shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium ${STATE_BADGE[state]}`}
                    >
                      {STATE_LABEL[state] || state}
                    </span>
                  </div>
                  <p className="mb-3 line-clamp-3 flex-1 text-xs leading-relaxed text-slate-600">
                    {displayDescription(item) || "官方主题"}
                  </p>
                  {item.tags && item.tags.length > 0 && (
                    <div className="mb-3 flex flex-wrap gap-1">
                      {item.tags.slice(0, 4).map((tag) => (
                        <span
                          key={tag}
                          className="rounded bg-slate-100 px-1.5 py-0.5 text-[10px] text-slate-600"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  )}
                  {incompatible && item.incompatibleReason && (
                    <p className="mb-3 rounded-md bg-rose-50 px-2 py-1.5 text-[11px] text-rose-700">
                      {item.incompatibleReason}
                    </p>
                  )}
                  <div className="mt-auto flex flex-wrap items-center gap-2">
                    {state === "not_installed" && !incompatible && (
                      <>
                        <AdminButton
                          size="sm"
                          variant="primary"
                          disabled={busy}
                          onClick={() => runInstall(item, true)}
                        >
                          {busy ? "安装中…" : "安装并激活"}
                        </AdminButton>
                        <AdminButton
                          size="sm"
                          variant="secondary"
                          disabled={busy}
                          onClick={() => runInstall(item, false)}
                        >
                          仅安装
                        </AdminButton>
                      </>
                    )}
                    {canUpdate && !incompatible && (
                      <AdminButton
                        size="sm"
                        variant="primary"
                        disabled={busy}
                        onClick={() => runInstall(item, isActive)}
                      >
                        {busy ? "更新中…" : isActive ? "更新并保持激活" : "更新"}
                      </AdminButton>
                    )}
                    {canActivateOnly && !isActive && !incompatible && (
                      <AdminButton
                        size="sm"
                        variant="primary"
                        disabled={busy}
                        onClick={() => runInstall(item, true)}
                      >
                        {busy ? "处理中…" : isBuiltin ? "激活" : "激活"}
                      </AdminButton>
                    )}
                    {isActive && !canUpdate && (
                      <AdminButton size="sm" variant="secondary" disabled>
                        当前主题
                      </AdminButton>
                    )}
                    {item.repoUrl && (
                      <a
                        href={item.repoUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-xs text-slate-500 hover:text-slate-800"
                      >
                        仓库 <ExternalLink className="h-3 w-3" />
                      </a>
                    )}
                    {!canInstall && !canActivateOnly && !isActive && !canUpdate && !incompatible && (
                      <span className="text-xs text-slate-400">无可用操作</span>
                    )}
                  </div>
                </div>
              </AdminCard>
            );
          })}
        </div>
      )}
    </div>
  );
}
