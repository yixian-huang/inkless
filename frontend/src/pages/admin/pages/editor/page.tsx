import { useState, useEffect, useCallback, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useSectionRegistry, useThemeManager } from "@/plugins/hooks";
import type { DynamicPageLayout, SectionData, SectionSettings } from "@/theme/types";
import DynamicPagePreview from "@/theme/DynamicPagePreview";
import {
  HOST_PAGE_PRESET_METAS,
  buildHostPagePreset,
  isHostPagePresetId,
  type HostPagePresetId,
} from "@/theme/pagePresets";
import {
  checkPageSlug,
  themePageSlugsFromManifest,
} from "@/lib/pageSlugConflict";
import PropertiesPanel from "./components/PropertiesPanel";
import { useDragSort } from "./hooks/useDragSort";
import {
  getUnifiedPage,
  getUnifiedPageDraft,
  createUnifiedPage,
  updateUnifiedPage,
  updateUnifiedPageDraft,
  publishUnifiedPage,
  unpublishUnifiedPage,
  rollbackUnifiedPage,
} from "@/api/unifiedPages";
import {
  cancelScheduledPublication,
  createScheduledPublication,
  getResourceScheduledPublication,
  retryScheduledPublication,
  updateScheduledPublication,
  type ScheduledPublication,
} from "@/api/scheduledPublications";
import { ScheduledPublicationPanel } from "@/components/admin/ScheduledPublicationPanel";
import SectionPicker from "./SectionPicker";
import SectionListItem from "./SectionList";
import { VersionHistoryPanel, ConflictDialog } from "./VersionHistoryPanel";
import { useAuth } from "@/contexts/AuthContext";
import {
  AdminButton,
  AdminLoading,
} from "@/components/admin/ui";
import { useBootstrap } from "@/contexts/BootstrapContext";

// ---------------------------------------------------------------------------
// Main page component
// ---------------------------------------------------------------------------
export default function PageEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isNew = !id;
  const pageId = id ? Number(id) : 0;
  const { metas: sectionMetas } = useSectionRegistry();
  const { activeTheme } = useThemeManager();
  const { refetch: refetchBootstrap } = useBootstrap();
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("pages:create");
  const canUpdate = hasPermission("pages:update");
  const canPublish = hasPermission("pages:publish");

  // -- page metadata --
  const [slug, setSlug] = useState("");
  /** Slug loaded from server — used so theme conflicts don't block keeping existing path */
  const [savedSlug, setSavedSlug] = useState("");
  const [zhTitle, setZhTitle] = useState("");
  const [enTitle, setEnTitle] = useState("");
  const [mode, setMode] = useState<"template" | "composable">("composable");
  const [showInNav, setShowInNav] = useState(false);
  const [sortOrder, setSortOrder] = useState(0);
  const [status, setStatus] = useState("draft");
  const [metadataDirty, setMetadataDirty] = useState(false);

  const themeSlugs = useMemo(
    () => themePageSlugsFromManifest(activeTheme?.pages),
    [activeTheme],
  );

  const slugCheck = useMemo(
    () =>
      checkPageSlug(slug, {
        themeSlugs,
        allowSlug: isNew ? undefined : savedSlug || undefined,
      }),
    [slug, themeSlugs, isNew, savedSlug],
  );
  /** Host composable preset for new pages */
  const [presetId, setPresetId] = useState<HostPagePresetId | "">("");
  /** Page-level config fields stored alongside sections in draftConfig */
  const [pageLayout, setPageLayout] = useState<DynamicPageLayout | string>("auto");
  /** auto = omit showPageHeader; true/false force */
  const [showPageHeaderMode, setShowPageHeaderMode] = useState<"auto" | "on" | "off">("auto");

  // -- section editor state --
  const [sections, setSections] = useState<SectionData[]>([]);
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [showPicker, setShowPicker] = useState(false);
  const [draftVersion, setDraftVersion] = useState(0);
  const [publishedVersion, setPublishedVersion] = useState(0);
  const [editorMode, setEditorMode] = useState<"visual" | "json">("visual");
  const [sectionJson, setSectionJson] = useState("[]");

  // -- UI state --
  const [saving, setSaving] = useState(false);
  const [metadataSaving, setMetadataSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [error, setError] = useState("");
  const [successMsg, setSuccessMsg] = useState("");
  const [showHistory, setShowHistory] = useState(false);
  const [conflictVersion, setConflictVersion] = useState<number | null>(null);
  const [loading, setLoading] = useState(!!id);
  const [scheduledPublication, setScheduledPublication] = useState<ScheduledPublication | null>(null);
  const [scheduleLoading, setScheduleLoading] = useState(!!id);
  const [scheduleBusy, setScheduleBusy] = useState(false);

  // -- load existing page --
  const loadPage = useCallback(async () => {
    if (!pageId) return;
    setLoading(true);
    try {
      const [meta, draft] = await Promise.all([
        getUnifiedPage(pageId),
        getUnifiedPageDraft(pageId),
      ]);
      setSlug(meta.slug);
      setSavedSlug(meta.slug);
      setZhTitle(meta.zhTitle);
      setEnTitle(meta.enTitle);
      setMode(meta.mode);
      setShowInNav(meta.showInNav);
      setSortOrder(meta.sortOrder);
      setStatus(meta.status);
      setPublishedVersion(meta.publishedVersion);
      setDraftVersion(draft.draftVersion);
      setMetadataDirty(false);

      const config = draft.draftConfig as {
        sections?: any[];
        layout?: string;
        showPageHeader?: boolean;
      } | null;
      // Backend stores content in "props"; frontend SectionData uses "data" — normalize.
      // Note: plain `s.data || s.props` is broken because `{}` is truthy in JS, so an
      // empty data object won't fall back to props. Check for meaningful content.
      const hasContent = (v: unknown): boolean =>
        !!v && typeof v === "object" && Object.keys(v as object).length > 0;
      const loadedSections: SectionData[] = (config?.sections || []).map((s: any) => ({
        ...s,
        data: hasContent(s.data) ? s.data : (s.props ?? {}),
      }));
      setSections(loadedSections);
      setSectionJson(JSON.stringify(loadedSections, null, 2));
      setPageLayout(
        typeof config?.layout === "string" && config.layout
          ? config.layout
          : "auto",
      );
      if (config?.showPageHeader === true) setShowPageHeaderMode("on");
      else if (config?.showPageHeader === false) setShowPageHeaderMode("off");
      else setShowPageHeaderMode("auto");

    } catch {
      setError("加载页面失败");
    } finally {
      setLoading(false);
    }
  }, [pageId]);

  const loadSchedule = useCallback(async () => {
    if (!pageId) {
      setScheduleLoading(false);
      return;
    }
    setScheduleLoading(true);
    try {
      const schedule = await getResourceScheduledPublication("page", pageId);
      setScheduledPublication(schedule);
    } catch {
      setScheduledPublication(null);
    } finally {
      setScheduleLoading(false);
    }
  }, [pageId]);

  useEffect(() => {
    loadPage();
    loadSchedule();
  }, [loadPage, loadSchedule]);

  // -- keep JSON in sync --
  useEffect(() => {
    if (editorMode === "visual") {
      setSectionJson(JSON.stringify(sections, null, 2));
    }
  }, [sections, editorMode]);

  // -- section helpers --
  const isComposable = mode === "composable";

  const addSection = useCallback(
    (type: string) => {
      const newSection: SectionData = {
        id: crypto.randomUUID(),
        type,
        variant: "default",
        locked: false,
        data: {},
        settings: {},
      };
      setSections((prev) => {
        setSelectedIndex(prev.length);
        return [...prev, newSection];
      });
      setShowPicker(false);
    },
    [],
  );

  const moveSection = useCallback((from: number, to: number) => {
    setSections((prev) => {
      const next = [...prev];
      const [item] = next.splice(from, 1);
      next.splice(to, 0, item);
      return next;
    });
    setSelectedIndex(to);
  }, []);

  const deleteSection = useCallback((index: number) => {
    if (!window.confirm("确定要删除此区块吗？")) return;
    setSections((prev) => prev.filter((_, i) => i !== index));
    setSelectedIndex((prev) => {
      if (prev === null) return null;
      if (prev === index) return null;
      if (prev > index) return prev - 1;
      return prev;
    });
  }, []);

  const updateSectionData = useCallback((index: number, data: Record<string, unknown>) => {
    setSections((prev) =>
      prev.map((s, i) => (i === index ? { ...s, data } : s)),
    );
  }, []);

  const updateSectionSettings = useCallback((index: number, settings: SectionSettings) => {
    setSections((prev) =>
      prev.map((s, i) => (i === index ? { ...s, settings } : s)),
    );
  }, []);

  const { makeDragHandlers } = useDragSort(moveSection);

  // -- mode toggle --
  const switchToJson = useCallback(() => {
    setSectionJson(JSON.stringify(sections, null, 2));
    setEditorMode("json");
  }, [sections]);

  const switchToVisual = useCallback(() => {
    try {
      const parsed = JSON.parse(sectionJson);
      const parsedSections: SectionData[] = Array.isArray(parsed) ? parsed : [];
      setSections(parsedSections);
      setSelectedIndex(null);
      setEditorMode("visual");
    } catch {
      setError("JSON 格式错误，无法切换到可视化模式");
    }
  }, [sectionJson]);

  // -- clear messages --
  const clearMessages = () => { setError(""); setSuccessMsg(""); };

  const resolvedShowPageHeader = useMemo((): boolean | undefined => {
    if (showPageHeaderMode === "on") return true;
    if (showPageHeaderMode === "off") return false;
    return undefined;
  }, [showPageHeaderMode]);

  const buildDraftConfig = useCallback(
    (sectionsToSave: SectionData[]) => {
      const cfg: Record<string, unknown> = { sections: sectionsToSave };
      if (pageLayout && pageLayout !== "auto") cfg.layout = pageLayout;
      else cfg.layout = "auto";
      if (resolvedShowPageHeader !== undefined) {
        cfg.showPageHeader = resolvedShowPageHeader;
      }
      return cfg;
    },
    [pageLayout, resolvedShowPageHeader],
  );

  const applyPreset = useCallback(
    (id: HostPagePresetId) => {
      const built = buildHostPagePreset(id, {
        zhTitle: zhTitle || slug || "未命名页面",
        enTitle: enTitle || slug || "Untitled page",
      });
      setSections(built.sections);
      setSectionJson(JSON.stringify(built.sections, null, 2));
      setSelectedIndex(built.sections.length ? 0 : null);
      setPageLayout(built.layout || "auto");
      if (built.showPageHeader === true) setShowPageHeaderMode("on");
      else if (built.showPageHeader === false) setShowPageHeaderMode("off");
      else setShowPageHeaderMode("auto");
      setPresetId(id);
      setSuccessMsg(`已套用配方：${HOST_PAGE_PRESET_METAS.find((m) => m.id === id)?.labelZh ?? id}`);
      setTimeout(() => setSuccessMsg(""), 2500);
    },
    [zhTitle, enTitle, slug],
  );

  // -- create new page --
  const handleCreate = async () => {
    if (!canCreate) return;
    clearMessages();
    const check = checkPageSlug(slug, { themeSlugs });
    if (check.blocking) {
      setError(check.messageZh);
      return;
    }
    setSaving(true);
    try {
      let sectionsToCreate = sections;
      let layoutForCreate: string | undefined =
        pageLayout && pageLayout !== "auto" ? String(pageLayout) : "auto";
      let headerForCreate = resolvedShowPageHeader;
      if (editorMode === "json") {
        const parsed = JSON.parse(sectionJson);
        sectionsToCreate = Array.isArray(parsed) ? parsed : [];
      }
      if (sectionsToCreate.length === 0 && presetId && isHostPagePresetId(presetId)) {
        const built = buildHostPagePreset(presetId, {
          zhTitle: zhTitle || slug,
          enTitle: enTitle || slug,
        });
        sectionsToCreate = built.sections;
        layoutForCreate = built.layout || "auto";
        headerForCreate = built.showPageHeader;
      }
      const draftConfig: Record<string, unknown> = {
        sections: sectionsToCreate,
        layout: layoutForCreate || "auto",
      };
      if (headerForCreate !== undefined) draftConfig.showPageHeader = headerForCreate;
      const result = await createUnifiedPage({
        slug: check.slug,
        zhTitle,
        enTitle,
        mode,
        showInNav,
        sortOrder,
        draftConfig,
      });
      navigate(`/admin/pages/edit/${result.id}`, { replace: true });
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || "创建失败");
    } finally {
      setSaving(false);
    }
  };

  // -- save draft --
  const handleSave = async () => {
    if (!canUpdate) return;
    clearMessages();
    setSaving(true);
    try {
      let sectionsToSave = sections;
      if (editorMode === "json") {
        try {
          const parsed = JSON.parse(sectionJson);
          sectionsToSave = Array.isArray(parsed) ? parsed : [];
        } catch {
          setError("JSON 格式错误");
          setSaving(false);
          return;
        }
      }
      const result: any = await updateUnifiedPageDraft(
        pageId,
        draftVersion,
        buildDraftConfig(sectionsToSave),
      );
      setDraftVersion(result.draftVersion ?? draftVersion + 1);
      setSuccessMsg("草稿已保存");
      setTimeout(() => setSuccessMsg(""), 3000);
    } catch (err: any) {
      if (err.response?.status === 409) {
        const serverVersion = err.response?.data?.currentVersion ?? err.response?.data?.version;
        setConflictVersion(serverVersion ?? 0);
      } else {
        setError(err?.response?.data?.error || err?.message || "保存失败");
      }
    } finally {
      setSaving(false);
    }
  };

  // -- save live route/navigation metadata --
  const handleSaveMetadata = async () => {
    if (!canUpdate) return;
    clearMessages();
    const check = checkPageSlug(slug, {
      themeSlugs,
      allowSlug: savedSlug || undefined,
    });
    if (check.blocking) {
      setError(check.messageZh);
      return;
    }
    setMetadataSaving(true);
    try {
      await updateUnifiedPage(pageId, {
        slug: check.slug,
        zhTitle,
        enTitle,
        sortOrder,
        showInNav,
      });
      setSlug(check.slug);
      setSavedSlug(check.slug);
      setMetadataDirty(false);
      if (status === "published") {
        await refetchBootstrap();
      }
      setSuccessMsg(status === "published" ? "页面信息已保存并立即生效" : "页面信息已保存");
      setTimeout(() => setSuccessMsg(""), 3000);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || "页面信息保存失败");
    } finally {
      setMetadataSaving(false);
    }
  };

  // -- publish --
  const handlePublish = async () => {
    if (!canPublish) return;
    clearMessages();
    if (metadataDirty) {
      setError("页面信息尚未保存；请先保存页面信息，再发布内容");
      return;
    }
    setPublishing(true);
    try {
      let sectionsToPublish = sections;
      if (editorMode === "json") {
        const parsed = JSON.parse(sectionJson);
        sectionsToPublish = Array.isArray(parsed) ? parsed : [];
      }
      const saved = await updateUnifiedPageDraft(pageId, draftVersion, {
        sections: sectionsToPublish,
      });
      const publishedDraftVersion = saved.draftVersion ?? draftVersion + 1;
      await publishUnifiedPage(pageId, publishedDraftVersion);
      setDraftVersion(publishedDraftVersion);
      setStatus("published");
      setPublishedVersion(publishedDraftVersion);
      await refetchBootstrap();
      setSuccessMsg("已发布");
      setTimeout(() => setSuccessMsg(""), 3000);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || "发布失败");
    } finally {
      setPublishing(false);
    }
  };

  const currentSectionsForSave = (): SectionData[] | null => {
    if (editorMode !== "json") return sections;
    try {
      const parsed = JSON.parse(sectionJson);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      setError("JSON 格式错误");
      return null;
    }
  };

  const handleSchedulePublish = async (scheduledAt: string) => {
    if (!canPublish || isNew) return;
    clearMessages();
    if (metadataDirty) {
      setError("页面信息尚未保存；请先保存页面信息，再安排定时发布");
      return;
    }

    const sectionsToPublish = currentSectionsForSave();
    if (!sectionsToPublish) return;

    setScheduleBusy(true);
    try {
      const saved = await updateUnifiedPageDraft(pageId, draftVersion, {
        sections: sectionsToPublish,
      });
      const scheduledDraftVersion = saved.draftVersion ?? draftVersion + 1;
      const next = scheduledPublication?.status === "pending"
        ? await updateScheduledPublication(scheduledPublication.id, {
            scheduledAt,
            expectedVersion: scheduledDraftVersion,
          })
        : await createScheduledPublication({
            resourceType: "page",
            resourceId: pageId,
            scheduledAt,
            expectedVersion: scheduledDraftVersion,
          });
      setDraftVersion(scheduledDraftVersion);
      setScheduledPublication(next);
      setSuccessMsg("定时发布已安排");
      setTimeout(() => setSuccessMsg(""), 3000);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || "定时发布失败");
    } finally {
      setScheduleBusy(false);
    }
  };

  const handleCancelSchedule = async () => {
    if (!canPublish || !scheduledPublication) return;
    clearMessages();
    setScheduleBusy(true);
    try {
      await cancelScheduledPublication(scheduledPublication.id);
      setScheduledPublication(null);
      setSuccessMsg("定时发布已取消");
      setTimeout(() => setSuccessMsg(""), 3000);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || "取消定时发布失败");
    } finally {
      setScheduleBusy(false);
    }
  };

  const handleRetrySchedule = async () => {
    if (!canPublish || !scheduledPublication) return;
    clearMessages();
    setScheduleBusy(true);
    try {
      const retried = await retryScheduledPublication(scheduledPublication.id);
      setScheduledPublication(retried);
      setSuccessMsg("定时发布已重新入队");
      setTimeout(() => setSuccessMsg(""), 3000);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || "重试定时发布失败");
    } finally {
      setScheduleBusy(false);
    }
  };

  // -- unpublish --
  const handleUnpublish = async () => {
    if (!canPublish) return;
    clearMessages();
    try {
      await unpublishUnifiedPage(pageId);
      setStatus("draft");
      setPublishedVersion(0);
      await refetchBootstrap();
      setSuccessMsg("已下线");
      setTimeout(() => setSuccessMsg(""), 3000);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || "下线失败");
    }
  };

  // -- rollback --
  const handleRollback = async (version: number) => {
    if (!canPublish) return;
    if (!window.confirm(`确定回滚到版本 ${version}？`)) return;
    clearMessages();
    try {
      await rollbackUnifiedPage(pageId, version);
      setShowHistory(false);
      await loadPage();
      await refetchBootstrap();
      setSuccessMsg(`已回滚到版本 ${version}`);
      setTimeout(() => setSuccessMsg(""), 3000);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || "回滚失败");
    }
  };

  // -- selected section --
  const selectedSection =
    selectedIndex !== null && selectedIndex < sections.length
      ? sections[selectedIndex]
      : null;

  const selectedMeta = selectedSection
    ? sectionMetas.find((m) => m.type === selectedSection.type)
    : null;

  if (loading) {
    return <AdminLoading />;
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* -- top bar -- */}
      <div className="flex items-center justify-between mb-4 flex-shrink-0">
        <div className="flex items-center gap-3">
          <AdminButton variant="ghost" size="sm" onClick={() => navigate("/admin/pages")}>
            ← 返回
          </AdminButton>
          <h2 className="text-xl font-semibold tracking-tight text-slate-900">
            {isNew ? "新建页面" : (zhTitle || slug || "编辑页面")}
          </h2>
          <span className={`text-xs px-2 py-0.5 rounded-full ${
            mode === "template"
              ? "bg-violet-50 text-violet-700 ring-1 ring-inset ring-violet-600/15"
              : "bg-blue-50 text-blue-700 ring-1 ring-inset ring-blue-600/15"
          }`}>
            {mode === "template" ? "模板" : "自由组合"}
          </span>
          <span className={`text-xs px-2 py-0.5 rounded-full ${
            status === "published"
              ? "bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-600/15"
              : "bg-amber-50 text-amber-800 ring-1 ring-inset ring-amber-600/15"
          }`}>
            {status === "published" ? "已发布" : "草稿"}
          </span>
        </div>

        <div className="flex items-center gap-2">
          {/* mode toggle */}
          <AdminButton
            size="sm"
            variant="secondary"
            onClick={() => (editorMode === "visual" ? switchToJson() : switchToVisual())}
          >
            {editorMode === "visual" ? "JSON 模式" : "可视化模式"}
          </AdminButton>

          {!isNew && (
            <AdminButton size="sm" variant="secondary" onClick={() => setShowHistory(true)}>
              版本历史
            </AdminButton>
          )}

          {isNew && canCreate ? (
            <button
              onClick={handleCreate}
              disabled={saving || slugCheck.blocking}
              className="inline-flex h-8 items-center justify-center rounded-lg bg-blue-600 px-3 text-xs font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50"
            >
              {saving ? "创建中..." : "创建"}
            </button>
          ) : !isNew ? (
            <>
              {canUpdate && (
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="inline-flex h-8 items-center justify-center rounded-lg bg-blue-600 px-3 text-xs font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50"
                >
                  {saving ? "保存中..." : "保存草稿"}
                </button>
              )}
              {canPublish && (
                status === "published" ? (
                  <button
                    onClick={handleUnpublish}
                    className="inline-flex h-8 items-center justify-center rounded-lg border border-orange-300 px-3 text-xs font-medium text-orange-700 hover:bg-orange-50"
                  >
                    下线
                  </button>
                ) : (
                  <button
                    onClick={handlePublish}
                    disabled={publishing}
                    className="inline-flex h-8 items-center justify-center rounded-lg bg-emerald-600 px-3 text-xs font-medium text-white shadow-sm hover:bg-emerald-700 disabled:opacity-50"
                  >
                    {publishing ? "发布中..." : "发布"}
                  </button>
                )
              )}
            </>
          ) : null}
        </div>
      </div>

      {/* -- messages -- */}
      {error && (
        <div className="mb-3 p-3 bg-red-50 text-red-700 rounded-xl text-sm flex-shrink-0">
          {error}
          <button onClick={() => setError("")} className="ml-2 text-red-500">&times;</button>
        </div>
      )}
      {successMsg && (
        <div className="mb-3 p-3 bg-green-50 text-green-700 rounded-xl text-sm flex-shrink-0">
          {successMsg}
        </div>
      )}

      {!isNew && (
        <div className="mb-4 flex-shrink-0">
          <ScheduledPublicationPanel
            item={scheduledPublication}
            loading={scheduleLoading}
            busy={scheduleBusy}
            canPublish={canPublish}
            disabledReason="需要 pages:publish 权限才能安排页面定时发布。"
            onSchedule={handleSchedulePublish}
            onCancel={handleCancelSchedule}
            onRetry={handleRetrySchedule}
            onRefresh={loadSchedule}
          />
        </div>
      )}

      {/* -- route and navigation metadata -- */}
      <div className="bg-white rounded-lg shadow p-5 mb-4 flex-shrink-0">
          <div className="flex items-start justify-between gap-4 mb-3">
            <div>
              <h3 className="text-sm font-semibold text-slate-700">页面信息</h3>
              {!isNew && (
                <p className="text-xs text-slate-500 mt-1">
                  {status === "published"
                    ? "路径与导航信息独立于内容版本，保存后会立即更新线上页面。"
                    : "页面发布前不会出现在公开路由；页面信息与内容草稿分别保存。"}
                </p>
              )}
              {isNew && (
                <p className="text-xs text-slate-500 mt-1">
                  可选用 Host 页面配方预填多区块结构（文档 / 指南 / 落地页）。
                </p>
              )}
            </div>
            {!isNew && canUpdate && (
              <button
                onClick={handleSaveMetadata}
                disabled={
                  metadataSaving ||
                  !metadataDirty ||
                  slugCheck.blocking
                }
                className="px-3 py-1.5 text-xs border border-blue-300 text-blue-700 rounded-xl hover:bg-blue-50 disabled:opacity-50"
              >
                {metadataSaving ? "保存中..." : "保存页面信息"}
              </button>
            )}
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label htmlFor="page-slug" className="block text-xs font-medium text-slate-600 mb-1">URL 路径 (slug)</label>
              <div className="flex items-center">
                <span className="text-slate-400 text-sm mr-1">/p/</span>
                <input
                  id="page-slug"
                  type="text"
                  value={slug}
                  onChange={(e) => {
                    setSlug(e.target.value);
                    setMetadataDirty(true);
                  }}
                  placeholder="about-us"
                  aria-invalid={slugCheck.blocking && slug.trim() !== ""}
                  className={`flex-1 rounded-lg border bg-white px-3 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-2 ${
                    slugCheck.blocking && slug.trim() !== ""
                      ? "border-red-300 focus:border-red-500 focus:ring-red-500/20"
                      : "border-slate-200 focus:border-blue-500 focus:ring-blue-500/20"
                  }`}
                />
              </div>
              {slugCheck.blocking && slug.trim() !== "" ? (
                <p className="mt-1 text-xs text-red-600" role="alert">
                  {slugCheck.messageZh}
                </p>
              ) : (
                <p className="mt-1 text-xs text-slate-500">
                  公开地址为 /p/&#123;slug&#125;。勿使用系统路径或当前主题已声明的页面（如 features）。
                </p>
              )}
            </div>
            <div>
              <label htmlFor="page-title-zh" className="block text-xs font-medium text-slate-600 mb-1">标题 (中文)</label>
              <input
                id="page-title-zh"
                type="text"
                value={zhTitle}
                onChange={(e) => {
                  setZhTitle(e.target.value);
                  setMetadataDirty(true);
                }}
                className="w-full rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
            </div>
            <div>
              <label htmlFor="page-title-en" className="block text-xs font-medium text-slate-600 mb-1">标题 (English)</label>
              <input
                id="page-title-en"
                type="text"
                value={enTitle}
                onChange={(e) => {
                  setEnTitle(e.target.value);
                  setMetadataDirty(true);
                }}
                className="w-full rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
            </div>
          </div>
          {isNew && (
            <div className="mt-3 flex flex-col sm:flex-row sm:items-end gap-3 p-3 rounded-lg bg-slate-50 border border-slate-100">
              <div className="flex-1 min-w-0">
                <label htmlFor="page-preset" className="block text-xs font-medium text-slate-600 mb-1">
                  页面配方（Host preset）
                </label>
                <select
                  id="page-preset"
                  value={presetId}
                  onChange={(e) => {
                    const v = e.target.value;
                    setPresetId(v && isHostPagePresetId(v) ? v : "");
                  }}
                  className="w-full rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                >
                  <option value="">空白（自行添加区块）</option>
                  {HOST_PAGE_PRESET_METAS.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.labelZh} — {m.descriptionZh}
                    </option>
                  ))}
                </select>
              </div>
              <button
                type="button"
                disabled={!presetId}
                onClick={() => {
                  if (presetId && isHostPagePresetId(presetId)) applyPreset(presetId);
                }}
                className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-3 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
              >
                套用到编辑器
              </button>
            </div>
          )}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-3">
            <div>
              <label htmlFor="page-mode" className="block text-xs font-medium text-slate-600 mb-1">页面模式</label>
              <select
                id="page-mode"
                value={mode}
                onChange={(e) => setMode(e.target.value as "template" | "composable")}
                disabled={!isNew}
                className="w-full rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              >
                <option value="composable">自由组合 (Composable)</option>
                <option value="template">模板 (Template)</option>
              </select>
            </div>
            <div>
              <label htmlFor="page-sort-order" className="block text-xs font-medium text-slate-600 mb-1">排序</label>
              <input
                id="page-sort-order"
                type="number"
                value={sortOrder}
                onChange={(e) => {
                  setSortOrder(Number(e.target.value));
                  setMetadataDirty(true);
                }}
                className="w-full rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
            </div>
            <div className="flex items-end">
              <label className="flex items-center gap-2 text-sm text-slate-700 pb-1">
                <input
                  type="checkbox"
                  checked={showInNav}
                  onChange={(e) => {
                    setShowInNav(e.target.checked);
                    setMetadataDirty(true);
                  }}
                  className="rounded border-slate-200"
                />
                显示在导航
              </label>
            </div>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-3 p-3 rounded-lg bg-slate-50 border border-slate-100">
            <div>
              <label htmlFor="page-layout" className="block text-xs font-medium text-slate-600 mb-1">
                呈现布局（写入草稿 config）
              </label>
              <select
                id="page-layout"
                value={pageLayout || "auto"}
                onChange={(e) => setPageLayout(e.target.value)}
                className="w-full rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              >
                <option value="auto">自动（按区块推断）</option>
                <option value="reading">阅读（页眉 + 栏宽）</option>
                <option value="landing">落地（全宽 section 栈）</option>
              </select>
            </div>
            <div>
              <label htmlFor="page-header-mode" className="block text-xs font-medium text-slate-600 mb-1">
                页级标题头
              </label>
              <select
                id="page-header-mode"
                value={showPageHeaderMode}
                onChange={(e) =>
                  setShowPageHeaderMode(e.target.value as "auto" | "on" | "off")
                }
                className="w-full rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              >
                <option value="auto">自动</option>
                <option value="on">始终显示</option>
                <option value="off">隐藏</option>
              </select>
            </div>
            <p className="md:col-span-2 text-xs text-slate-500">
              布局与标题头随「保存草稿」写入 published 前的 draftConfig；预览区与公开 DynamicPage 壳一致。
            </p>
          </div>
        </div>

      {/* -- editor body -- */}
      {editorMode === "json" ? (
        /* JSON mode */
        <div className="bg-white rounded-lg shadow p-5 flex-1 flex flex-col min-h-0">
          <div className="flex items-center justify-between mb-2">
            <label htmlFor="page-sections-json" className="text-sm font-medium text-slate-700">
              区块配置 (JSON 数组)
            </label>
            <span className="text-xs text-slate-400">
              可用类型: {sectionMetas.map((m) => m.type).join(", ")}
            </span>
          </div>
          <textarea
            id="page-sections-json"
            value={sectionJson}
            onChange={(e) => setSectionJson(e.target.value)}
            rows={20}
            className="w-full flex-1 resize-none rounded-xl border border-slate-200 bg-white px-3 py-2 font-mono text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
            spellCheck={false}
          />
        </div>
      ) : (
        /* Visual mode — three-column layout */
        <div className="flex-1 flex min-h-0 bg-white rounded-lg shadow overflow-hidden">
          {/* Left sidebar: section list */}
          <div className="w-64 flex-shrink-0 border-r border-slate-200/90 flex flex-col">
            <div className="px-3 py-2 border-b border-slate-200 flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-600 uppercase">区块列表</span>
              <span className="text-xs text-slate-400">{sections.length} 个</span>
            </div>
            <div className="flex-1 overflow-y-auto p-3 space-y-2">
              {sections.map((section, i) => (
                <SectionListItem
                  key={section.id}
                  section={section}
                  index={i}
                  total={sections.length}
                  isSelected={selectedIndex === i}
                  isComposable={isComposable}
                  onSelect={() => setSelectedIndex(i)}
                  onMoveUp={() => { if (i > 0) moveSection(i, i - 1); }}
                  onMoveDown={() => { if (i < sections.length - 1) moveSection(i, i + 1); }}
                  onDelete={() => deleteSection(i)}
                  dragHandlers={makeDragHandlers(i)}
                />
              ))}
              {sections.length === 0 && (
                <div className="text-xs text-slate-400 text-center py-4">暂无区块</div>
              )}
            </div>
            {isComposable && (
              <div className="p-3 border-t border-slate-200">
                <button
                  onClick={() => setShowPicker(true)}
                  className="flex w-full items-center justify-center gap-1 rounded-xl border-2 border-dashed border-slate-200 px-3 py-2 text-sm text-slate-500 transition-colors hover:border-blue-400 hover:bg-blue-50/50 hover:text-blue-600"
                >
                  <span className="text-lg leading-none">+</span> 添加区块
                </button>
              </div>
            )}
          </div>

          {/* Center: preview — same shell as public DynamicPage */}
          <div className="flex-1 overflow-y-auto bg-slate-100/80">
            {sections.length === 0 ? (
              <div className="flex items-center justify-center h-full text-sm text-slate-400">
                {isComposable ? "点击左侧「+ 添加区块」开始构建页面" : "暂无内容"}
              </div>
            ) : (
              <div className="min-h-full border-x border-slate-200/80 bg-white shadow-sm">
                <div className="sticky top-0 z-10 px-3 py-1.5 bg-slate-50/95 border-b border-slate-200 text-[11px] text-slate-500 flex items-center justify-between">
                  <span>
                    预览 · layout={pageLayout || "auto"}
                    {showPageHeaderMode !== "auto"
                      ? ` · header=${showPageHeaderMode}`
                      : ""}
                  </span>
                  <span className="text-slate-400">与公开页同壳</span>
                </div>
                <DynamicPagePreview
                  title={zhTitle || enTitle || slug || "页面标题"}
                  description=""
                  layout={pageLayout || "auto"}
                  showPageHeader={resolvedShowPageHeader}
                  sections={sections}
                  selectedIndex={selectedIndex}
                  onSelectSection={setSelectedIndex}
                />
              </div>
            )}
          </div>

          {/* Right sidebar: section data editor */}
          <div className="w-80 flex-shrink-0 border-l border-slate-200 flex flex-col">
            <div className="px-3 py-2 border-b border-slate-200">
              <span className="text-xs font-semibold text-slate-600 uppercase">
                {selectedSection ? (selectedMeta?.labelZh || selectedSection.type) : "属性编辑"}
              </span>
            </div>
            <div className="flex-1 overflow-y-auto p-3">
              {selectedSection ? (
                <PropertiesPanel
                  section={selectedSection}
                  onDataChange={(data) => updateSectionData(selectedIndex!, data)}
                  onSettingsChange={(settings) => updateSectionSettings(selectedIndex!, settings)}
                />
              ) : (
                <div className="text-xs text-slate-400 text-center py-8">
                  选择左侧区块以编辑属性
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* -- bottom version info -- */}
      {!isNew && (
        <div className="flex items-center justify-between mt-3 text-xs text-slate-500 flex-shrink-0">
          <div>
            草稿版本: <strong>{draftVersion}</strong>
            {publishedVersion > 0 && (
              <span className="ml-3">已发布版本: <strong>{publishedVersion}</strong></span>
            )}
          </div>
          <div>/{slug}</div>
        </div>
      )}

      {/* -- modals -- */}
      {showPicker && (
        <SectionPicker onSelect={addSection} onClose={() => setShowPicker(false)} />
      )}
      {showHistory && !isNew && (
        <VersionHistoryPanel
          pageId={pageId}
          onClose={() => setShowHistory(false)}
          onRollback={handleRollback}
          canRollback={canPublish}
        />
      )}
      {conflictVersion !== null && (
        <ConflictDialog
          currentVersion={conflictVersion}
          onReload={() => { setConflictVersion(null); loadPage(); }}
          onDismiss={() => setConflictVersion(null)}
        />
      )}
    </div>
  );
}
