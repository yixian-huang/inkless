import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { ExternalLink, RefreshCw, Store, TimerReset } from "lucide-react";
import {
  fetchThemeAutoUpdateSettings,
  fetchThemeCatalog,
  installOrUpdateThemeFromCatalog,
  runThemeAutoUpdate,
  updateThemeAutoUpdateSettings,
  type ThemeAutoUpdateReport,
  type ThemeAutoUpdateSettings,
  type ThemeCatalogItem,
  type ThemeCatalogResponse,
  type ThemeInstallState,
} from "@/api/extensionsThemes";
import {
  AdminButton,
  AdminCard,
  AdminErrorBanner,
  AdminField,
  AdminHint,
  AdminInput,
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

const INTERVAL_PRESETS = [15, 30, 60, 180, 360, 1440] as const;

function displayName(item: ThemeCatalogItem): string {
  return item.nameZh || item.name || item.slug;
}

function displayDescription(item: ThemeCatalogItem): string {
  return item.descriptionZh || item.description || "";
}

function formatReportSummary(report: ThemeAutoUpdateReport | null | undefined): string {
  if (!report) return "尚无检查记录";
  const u = report.updated?.length ?? 0;
  const e = report.errors?.length ?? 0;
  const s = report.skipped?.length ?? 0;
  return `检查 ${report.checked} · 更新 ${u} · 跳过 ${s} · 失败 ${e}` +
    (report.catalogSource ? ` · 目录 ${report.catalogSource}` : "");
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

  const [autoSettings, setAutoSettings] = useState<ThemeAutoUpdateSettings | null>(null);
  const [autoLoading, setAutoLoading] = useState(true);
  const [autoSaving, setAutoSaving] = useState(false);
  const [autoRunning, setAutoRunning] = useState(false);
  const [lastRunReport, setLastRunReport] = useState<ThemeAutoUpdateReport | null>(null);

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

  const loadAutoSettings = useCallback(async () => {
    setAutoLoading(true);
    try {
      const data = await fetchThemeAutoUpdateSettings();
      setAutoSettings(data);
      if (data.lastReport) setLastRunReport(data.lastReport);
    } catch (e: any) {
      // Non-fatal for market browse; surface inline in auto panel.
      setAutoSettings(null);
      setError((prev) => prev || e?.response?.data?.error || e?.message || "加载自动更新配置失败");
    } finally {
      setAutoLoading(false);
    }
  }, []);

  useEffect(() => {
    load(false);
    loadAutoSettings();
  }, [load, loadAutoSettings]);

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

  async function saveAutoPatch(
    patch: Partial<ThemeAutoUpdateSettings>,
    successMsg?: string,
  ) {
    if (!autoSettings) return;
    setAutoSaving(true);
    setError("");
    setSuccess("");
    try {
      const saved = await updateThemeAutoUpdateSettings({
        enabled: patch.enabled ?? autoSettings.enabled,
        intervalMinutes: patch.intervalMinutes ?? autoSettings.intervalMinutes,
        onlyMarketplace: patch.onlyMarketplace ?? autoSettings.onlyMarketplace,
        includeActive: patch.includeActive ?? autoSettings.includeActive,
        onlyActive: patch.onlyActive ?? autoSettings.onlyActive,
      });
      setAutoSettings(saved);
      if (saved.lastReport) setLastRunReport(saved.lastReport);
      if (successMsg) setSuccess(successMsg);
    } catch (e: any) {
      setError(e?.response?.data?.error || e?.message || "保存自动更新配置失败");
    } finally {
      setAutoSaving(false);
    }
  }

  async function handleAutoRun(dryRun: boolean) {
    setAutoRunning(true);
    setError("");
    setSuccess("");
    try {
      const report = await runThemeAutoUpdate({ dryRun });
      setLastRunReport(report);
      const summary = formatReportSummary(report);
      setSuccess(
        dryRun
          ? `检查完成（未写入）：${summary}`
          : `已执行自动更新：${summary}`,
      );
      await loadAutoSettings();
      await load(false);
      if (!dryRun && (report.updated?.length ?? 0) > 0) {
        await refetchBootstrap();
      }
    } catch (e: any) {
      setError(e?.response?.data?.error || e?.message || "运行自动更新失败");
    } finally {
      setAutoRunning(false);
    }
  }

  const reportForUi = lastRunReport ?? autoSettings?.lastReport ?? null;

  return (
    <div className="mx-auto max-w-6xl">
      <AdminPageHeader
        title="主题市场"
        description="浏览并一键安装官方主题；可选自动从 catalog 同步小版本，无需重部署站点"
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

      <AdminCard className="mb-4 p-4">
        <div className="mb-3 flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 className="flex items-center gap-2 text-sm font-semibold text-slate-900">
              <TimerReset className="h-4 w-4 text-slate-500" />
              可选自动更新
            </h2>
            <p className="mt-1 max-w-2xl text-xs leading-relaxed text-slate-500">
              默认关闭。开启后定时拉取官方 catalog，对已装 marketplace 主题升级 UMD 指针（补丁/小版本）。
              不会切换「当前激活」的主题，也不适合替代大版本换皮——大版本请在下方卡片手动确认。
            </p>
          </div>
          {autoSettings && (
            <label className="inline-flex cursor-pointer items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm">
              <input
                type="checkbox"
                className="h-4 w-4 rounded border-slate-300 text-neutral-900 focus:ring-neutral-500"
                checked={autoSettings.enabled}
                disabled={autoSaving || autoLoading}
                onChange={(e) =>
                  saveAutoPatch(
                    { enabled: e.target.checked },
                    e.target.checked ? "已开启主题自动更新" : "已关闭主题自动更新",
                  )
                }
              />
              <span className="font-medium text-slate-800">
                {autoSettings.enabled ? "已开启" : "已关闭"}
              </span>
            </label>
          )}
        </div>

        {autoLoading && !autoSettings ? (
          <AdminLoading label="加载自动更新配置…" />
        ) : !autoSettings ? (
          <p className="text-xs text-slate-500">无法读取自动更新配置（后端未部署或权限不足）。</p>
        ) : (
          <div className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <AdminField
                label="检查间隔（分钟）"
                hint="最小 15，最大 1440（24 小时）"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <AdminInput
                    type="number"
                    min={15}
                    max={1440}
                    step={15}
                    className="w-28"
                    value={autoSettings.intervalMinutes}
                    disabled={autoSaving}
                    onChange={(e) => {
                      const n = Number(e.target.value);
                      setAutoSettings({
                        ...autoSettings,
                        intervalMinutes: Number.isFinite(n) ? n : autoSettings.intervalMinutes,
                      });
                    }}
                    onBlur={() =>
                      saveAutoPatch(
                        { intervalMinutes: autoSettings.intervalMinutes },
                        "已保存检查间隔",
                      )
                    }
                  />
                  <select
                    className="h-9 rounded-lg border border-slate-200 bg-white px-2 text-xs text-slate-700"
                    value={
                      INTERVAL_PRESETS.includes(
                        autoSettings.intervalMinutes as (typeof INTERVAL_PRESETS)[number],
                      )
                        ? String(autoSettings.intervalMinutes)
                        : ""
                    }
                    disabled={autoSaving}
                    onChange={(e) => {
                      const n = Number(e.target.value);
                      if (!n) return;
                      void saveAutoPatch({ intervalMinutes: n }, `间隔已设为 ${n} 分钟`);
                    }}
                  >
                    <option value="">预设…</option>
                    {INTERVAL_PRESETS.map((m) => (
                      <option key={m} value={m}>
                        {m < 60 ? `${m} 分钟` : m === 60 ? "1 小时" : m === 1440 ? "24 小时" : `${m / 60} 小时`}
                      </option>
                    ))}
                  </select>
                </div>
              </AdminField>

              <label className="flex cursor-pointer items-start gap-2 rounded-lg border border-slate-100 p-3 text-xs text-slate-700">
                <input
                  type="checkbox"
                  className="mt-0.5 h-4 w-4 rounded border-slate-300"
                  checked={autoSettings.onlyMarketplace}
                  disabled={autoSaving}
                  onChange={(e) =>
                    saveAutoPatch({ onlyMarketplace: e.target.checked }, "已保存范围设置")
                  }
                />
                <span>
                  <span className="font-medium text-slate-900">仅 marketplace / external</span>
                  <AdminHint>跳过内置主题，只更新有 UMD URL 的安装记录。</AdminHint>
                </span>
              </label>

              <label className="flex cursor-pointer items-start gap-2 rounded-lg border border-slate-100 p-3 text-xs text-slate-700">
                <input
                  type="checkbox"
                  className="mt-0.5 h-4 w-4 rounded border-slate-300"
                  checked={autoSettings.includeActive}
                  disabled={autoSaving}
                  onChange={(e) =>
                    saveAutoPatch({ includeActive: e.target.checked }, "已保存范围设置")
                  }
                />
                <span>
                  <span className="font-medium text-slate-900">包含当前激活主题</span>
                  <AdminHint>只升级包指针，不改变「哪个主题在用」。</AdminHint>
                </span>
              </label>

              <label className="flex cursor-pointer items-start gap-2 rounded-lg border border-slate-100 p-3 text-xs text-slate-700">
                <input
                  type="checkbox"
                  className="mt-0.5 h-4 w-4 rounded border-slate-300"
                  checked={autoSettings.onlyActive}
                  disabled={autoSaving}
                  onChange={(e) =>
                    saveAutoPatch({ onlyActive: e.target.checked }, "已保存范围设置")
                  }
                />
                <span>
                  <span className="font-medium text-slate-900">只检查激活主题</span>
                  <AdminHint>其它已装主题留在旧版本，直到手动更新。</AdminHint>
                </span>
              </label>
            </div>

            <div className="flex flex-wrap items-center gap-2 border-t border-slate-100 pt-3">
              <AdminButton
                size="sm"
                variant="secondary"
                disabled={autoRunning || autoSaving}
                onClick={() => handleAutoRun(true)}
              >
                {autoRunning ? "检查中…" : "立即检查（不写入）"}
              </AdminButton>
              <AdminButton
                size="sm"
                variant="primary"
                disabled={autoRunning || autoSaving}
                onClick={() => handleAutoRun(false)}
              >
                {autoRunning ? "应用中…" : "立即应用更新"}
              </AdminButton>
              <span className="text-[11px] text-slate-400">
                手动运行不依赖「已开启」开关。
              </span>
            </div>

            <div className="rounded-lg bg-slate-50 px-3 py-2 text-xs text-slate-600">
              <div className="flex flex-wrap gap-x-4 gap-y-1">
                <span>
                  上次检查：{autoSettings.lastCheckAt || "—"}
                </span>
                <span>
                  上次应用：{autoSettings.lastApplyAt || "—"}
                </span>
                {autoSettings.lastError ? (
                  <span className="text-rose-600">错误：{autoSettings.lastError}</span>
                ) : null}
              </div>
              <p className="mt-1 text-slate-500">{formatReportSummary(reportForUi)}</p>
              {reportForUi && (reportForUi.updated?.length || reportForUi.errors?.length) ? (
                <ul className="mt-2 max-h-28 space-y-0.5 overflow-y-auto font-mono text-[11px] text-slate-500">
                  {(reportForUi.updated || []).map((it) => (
                    <li key={`u-${it.themeId}-${it.to}`}>
                      ↑ {it.slug || it.themeId}: {it.from || "?"} → {it.to || "?"}
                      {it.reason ? ` (${it.reason})` : ""}
                    </li>
                  ))}
                  {(reportForUi.errors || []).map((it) => (
                    <li key={`e-${it.themeId}`} className="text-rose-600">
                      ✗ {it.slug || it.themeId}: {it.reason || "error"}
                    </li>
                  ))}
                </ul>
              ) : null}
            </div>
          </div>
        )}
      </AdminCard>

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
