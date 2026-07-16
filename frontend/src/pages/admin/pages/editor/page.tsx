import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { SectionRenderer } from "@/theme/sections";
import { useSectionRegistry } from "@/plugins/hooks";
import type { SectionData, SectionSettings } from "@/theme/types";
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
import SectionPicker from "./SectionPicker";
import SectionListItem from "./SectionList";
import { VersionHistoryPanel, ConflictDialog } from "./VersionHistoryPanel";
import { useAuth } from "@/contexts/AuthContext";
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
  const { refetch: refetchBootstrap } = useBootstrap();
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("pages:create");
  const canUpdate = hasPermission("pages:update");
  const canPublish = hasPermission("pages:publish");

  // -- page metadata --
  const [slug, setSlug] = useState("");
  const [zhTitle, setZhTitle] = useState("");
  const [enTitle, setEnTitle] = useState("");
  const [mode, setMode] = useState<"template" | "composable">("composable");
  const [showInNav, setShowInNav] = useState(false);
  const [sortOrder, setSortOrder] = useState(0);
  const [status, setStatus] = useState("draft");
  const [metadataDirty, setMetadataDirty] = useState(false);

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
      setZhTitle(meta.zhTitle);
      setEnTitle(meta.enTitle);
      setMode(meta.mode);
      setShowInNav(meta.showInNav);
      setSortOrder(meta.sortOrder);
      setStatus(meta.status);
      setPublishedVersion(meta.publishedVersion);
      setDraftVersion(draft.draftVersion);
      setMetadataDirty(false);

      const config = draft.draftConfig as { sections?: any[] } | null;
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
    } catch {
      setError("加载页面失败");
    } finally {
      setLoading(false);
    }
  }, [pageId]);

  useEffect(() => {
    loadPage();
  }, [loadPage]);

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

  // -- create new page --
  const handleCreate = async () => {
    if (!canCreate) return;
    clearMessages();
    if (!slug.trim()) { setError("请输入 URL 路径"); return; }
    setSaving(true);
    try {
      let sectionsToCreate = sections;
      if (editorMode === "json") {
        const parsed = JSON.parse(sectionJson);
        sectionsToCreate = Array.isArray(parsed) ? parsed : [];
      }
      const result = await createUnifiedPage({
        slug,
        zhTitle,
        enTitle,
        mode,
        showInNav,
        sortOrder,
        draftConfig: { sections: sectionsToCreate },
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
      const result: any = await updateUnifiedPageDraft(pageId, draftVersion, {
        sections: sectionsToSave,
      });
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
    setMetadataSaving(true);
    try {
      await updateUnifiedPage(pageId, {
        slug,
        zhTitle,
        enTitle,
        sortOrder,
        showInNav,
      });
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
    return <div className="text-center py-12 text-gray-500">加载中...</div>;
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* -- top bar -- */}
      <div className="flex items-center justify-between mb-4 flex-shrink-0">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate("/admin/pages")}
            className="text-sm text-gray-500 hover:text-gray-800"
          >
            &larr; 返回
          </button>
          <h2 className="text-xl font-bold text-gray-900">
            {isNew ? "新建页面" : (zhTitle || slug || "编辑页面")}
          </h2>
          <span className={`text-xs px-2 py-0.5 rounded-full ${
            mode === "template"
              ? "bg-purple-100 text-purple-700"
              : "bg-blue-100 text-blue-700"
          }`}>
            {mode === "template" ? "模板" : "自由组合"}
          </span>
          <span className={`text-xs px-2 py-0.5 rounded-full ${
            status === "published"
              ? "bg-green-100 text-green-700"
              : "bg-yellow-100 text-yellow-700"
          }`}>
            {status === "published" ? "已发布" : "草稿"}
          </span>
        </div>

        <div className="flex items-center gap-2">
          {/* mode toggle */}
          <button
            onClick={() => editorMode === "visual" ? switchToJson() : switchToVisual()}
            className="px-3 py-1.5 text-xs border border-gray-300 rounded-md hover:bg-gray-50"
          >
            {editorMode === "visual" ? "JSON 模式" : "可视化模式"}
          </button>

          {!isNew && (
            <button
              onClick={() => setShowHistory(true)}
              className="px-3 py-1.5 text-xs border border-gray-300 rounded-md hover:bg-gray-50"
            >
              版本历史
            </button>
          )}

          {isNew && canCreate ? (
            <button
              onClick={handleCreate}
              disabled={saving || !slug.trim()}
              className="px-4 py-1.5 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 disabled:opacity-50"
            >
              {saving ? "创建中..." : "创建"}
            </button>
          ) : !isNew ? (
            <>
              {canUpdate && (
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="px-4 py-1.5 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 disabled:opacity-50"
                >
                  {saving ? "保存中..." : "保存草稿"}
                </button>
              )}
              {canPublish && (
                status === "published" ? (
                  <button
                    onClick={handleUnpublish}
                    className="px-4 py-1.5 text-sm border border-orange-400 text-orange-600 rounded-md hover:bg-orange-50"
                  >
                    下线
                  </button>
                ) : (
                  <button
                    onClick={handlePublish}
                    disabled={publishing}
                    className="px-4 py-1.5 bg-green-600 text-white text-sm rounded-md hover:bg-green-700 disabled:opacity-50"
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
        <div className="mb-3 p-3 bg-red-50 text-red-700 rounded-md text-sm flex-shrink-0">
          {error}
          <button onClick={() => setError("")} className="ml-2 text-red-500">&times;</button>
        </div>
      )}
      {successMsg && (
        <div className="mb-3 p-3 bg-green-50 text-green-700 rounded-md text-sm flex-shrink-0">
          {successMsg}
        </div>
      )}

      {/* -- route and navigation metadata -- */}
      <div className="bg-white rounded-lg shadow p-5 mb-4 flex-shrink-0">
          <div className="flex items-start justify-between gap-4 mb-3">
            <div>
              <h3 className="text-sm font-semibold text-gray-700">页面信息</h3>
              {!isNew && (
                <p className="text-xs text-gray-500 mt-1">
                  {status === "published"
                    ? "路径与导航信息独立于内容版本，保存后会立即更新线上页面。"
                    : "页面发布前不会出现在公开路由；页面信息与内容草稿分别保存。"}
                </p>
              )}
            </div>
            {!isNew && canUpdate && (
              <button
                onClick={handleSaveMetadata}
                disabled={metadataSaving || !metadataDirty || !slug.trim()}
                className="px-3 py-1.5 text-xs border border-blue-300 text-blue-700 rounded-md hover:bg-blue-50 disabled:opacity-50"
              >
                {metadataSaving ? "保存中..." : "保存页面信息"}
              </button>
            )}
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label htmlFor="page-slug" className="block text-xs font-medium text-gray-600 mb-1">URL 路径 (slug)</label>
              <div className="flex items-center">
                <span className="text-gray-400 text-sm mr-1">/</span>
                <input
                  id="page-slug"
                  type="text"
                  value={slug}
                  onChange={(e) => {
                    setSlug(e.target.value);
                    setMetadataDirty(true);
                  }}
                  placeholder="about-us"
                  className="flex-1 border border-gray-300 rounded-md px-3 py-1.5 text-sm"
                />
              </div>
            </div>
            <div>
              <label htmlFor="page-title-zh" className="block text-xs font-medium text-gray-600 mb-1">标题 (中文)</label>
              <input
                id="page-title-zh"
                type="text"
                value={zhTitle}
                onChange={(e) => {
                  setZhTitle(e.target.value);
                  setMetadataDirty(true);
                }}
                className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
              />
            </div>
            <div>
              <label htmlFor="page-title-en" className="block text-xs font-medium text-gray-600 mb-1">标题 (English)</label>
              <input
                id="page-title-en"
                type="text"
                value={enTitle}
                onChange={(e) => {
                  setEnTitle(e.target.value);
                  setMetadataDirty(true);
                }}
                className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
              />
            </div>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-3">
            <div>
              <label htmlFor="page-mode" className="block text-xs font-medium text-gray-600 mb-1">页面模式</label>
              <select
                id="page-mode"
                value={mode}
                onChange={(e) => setMode(e.target.value as "template" | "composable")}
                disabled={!isNew}
                className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
              >
                <option value="composable">自由组合 (Composable)</option>
                <option value="template">模板 (Template)</option>
              </select>
            </div>
            <div>
              <label htmlFor="page-sort-order" className="block text-xs font-medium text-gray-600 mb-1">排序</label>
              <input
                id="page-sort-order"
                type="number"
                value={sortOrder}
                onChange={(e) => {
                  setSortOrder(Number(e.target.value));
                  setMetadataDirty(true);
                }}
                className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
              />
            </div>
            <div className="flex items-end">
              <label className="flex items-center gap-2 text-sm text-gray-700 pb-1">
                <input
                  type="checkbox"
                  checked={showInNav}
                  onChange={(e) => {
                    setShowInNav(e.target.checked);
                    setMetadataDirty(true);
                  }}
                  className="rounded border-gray-300"
                />
                显示在导航
              </label>
            </div>
          </div>
        </div>

      {/* -- editor body -- */}
      {editorMode === "json" ? (
        /* JSON mode */
        <div className="bg-white rounded-lg shadow p-5 flex-1 flex flex-col min-h-0">
          <div className="flex items-center justify-between mb-2">
            <label htmlFor="page-sections-json" className="text-sm font-medium text-gray-700">
              区块配置 (JSON 数组)
            </label>
            <span className="text-xs text-gray-400">
              可用类型: {sectionMetas.map((m) => m.type).join(", ")}
            </span>
          </div>
          <textarea
            id="page-sections-json"
            value={sectionJson}
            onChange={(e) => setSectionJson(e.target.value)}
            rows={20}
            className="flex-1 w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono resize-none"
            spellCheck={false}
          />
        </div>
      ) : (
        /* Visual mode — three-column layout */
        <div className="flex-1 flex min-h-0 bg-white rounded-lg shadow overflow-hidden">
          {/* Left sidebar: section list */}
          <div className="w-64 flex-shrink-0 border-r border-gray-200 flex flex-col">
            <div className="px-3 py-2 border-b border-gray-200 flex items-center justify-between">
              <span className="text-xs font-semibold text-gray-600 uppercase">区块列表</span>
              <span className="text-xs text-gray-400">{sections.length} 个</span>
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
                <div className="text-xs text-gray-400 text-center py-4">暂无区块</div>
              )}
            </div>
            {isComposable && (
              <div className="p-3 border-t border-gray-200">
                <button
                  onClick={() => setShowPicker(true)}
                  className="w-full flex items-center justify-center gap-1 px-3 py-2 border-2 border-dashed border-gray-300 rounded-md text-sm text-gray-500 hover:border-blue-400 hover:text-blue-600 transition-colors"
                >
                  <span className="text-lg leading-none">+</span> 添加区块
                </button>
              </div>
            )}
          </div>

          {/* Center: preview */}
          <div className="flex-1 overflow-y-auto bg-gray-50">
            {sections.length === 0 ? (
              <div className="flex items-center justify-center h-full text-sm text-gray-400">
                {isComposable ? "点击左侧「+ 添加区块」开始构建页面" : "暂无内容"}
              </div>
            ) : (
              <div className="border-l border-r border-gray-100">
                {sections.map((s, i) => (
                  <div
                    key={s.id}
                    onClick={() => setSelectedIndex(i)}
                    className={`relative cursor-pointer transition-all ${
                      selectedIndex === i
                        ? "ring-2 ring-blue-400 ring-inset"
                        : "hover:ring-1 hover:ring-gray-300 hover:ring-inset"
                    }`}
                  >
                    <SectionRenderer section={s} />
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Right sidebar: section data editor */}
          <div className="w-80 flex-shrink-0 border-l border-gray-200 flex flex-col">
            <div className="px-3 py-2 border-b border-gray-200">
              <span className="text-xs font-semibold text-gray-600 uppercase">
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
                <div className="text-xs text-gray-400 text-center py-8">
                  选择左侧区块以编辑属性
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* -- bottom version info -- */}
      {!isNew && (
        <div className="flex items-center justify-between mt-3 text-xs text-gray-500 flex-shrink-0">
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
