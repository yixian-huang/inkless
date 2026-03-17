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
  updateUnifiedPageDraft,
  publishUnifiedPage,
  unpublishUnifiedPage,
  listUnifiedPageVersions,
  rollbackUnifiedPage,
} from "@/api/unifiedPages";

// ---------------------------------------------------------------------------
// SectionPicker — modal overlay to add a new section type
// ---------------------------------------------------------------------------
function SectionPicker({
  onSelect,
  onClose,
}: {
  onSelect: (type: string) => void;
  onClose: () => void;
}) {
  const { metas: sectionMetas } = useSectionRegistry();

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg mx-4 p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">添加区块</h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 text-xl leading-none"
          >
            &times;
          </button>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          {sectionMetas.map((meta) => (
            <button
              key={meta.type}
              onClick={() => onSelect(meta.type)}
              className="flex flex-col items-start p-3 border border-gray-200 rounded-lg hover:border-blue-400 hover:bg-blue-50 transition-colors text-left"
            >
              <span className="text-sm font-medium text-gray-900">
                {meta.labelZh}
              </span>
              <span className="text-xs text-gray-500">{meta.label}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// SectionListItem — a single row in the section list sidebar
// ---------------------------------------------------------------------------
function SectionListItem({
  section,
  index,
  total,
  isSelected,
  isComposable,
  onSelect,
  onMoveUp,
  onMoveDown,
  onDelete,
  dragHandlers,
}: {
  section: SectionData;
  index: number;
  total: number;
  isSelected: boolean;
  isComposable: boolean;
  onSelect: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onDelete: () => void;
  dragHandlers: {
    onDragStart: (e: React.DragEvent) => void;
    onDragOver: (e: React.DragEvent) => void;
    onDrop: (e: React.DragEvent) => void;
    onDragEnd: () => void;
  };
}) {
  const { metas: sectionMetas } = useSectionRegistry();
  const meta = sectionMetas.find((m) => m.type === section.type);
  const label = meta?.labelZh || section.type;
  const locked = !!section.locked;
  const draggable = isComposable && !locked;

  return (
    <div
      draggable={draggable}
      onDragStart={draggable ? dragHandlers.onDragStart : undefined}
      onDragOver={draggable ? dragHandlers.onDragOver : undefined}
      onDrop={draggable ? dragHandlers.onDrop : undefined}
      onDragEnd={draggable ? dragHandlers.onDragEnd : undefined}
      onClick={onSelect}
      className={`flex items-center gap-2 px-3 py-2 rounded-md cursor-pointer select-none border transition-colors ${
        isSelected
          ? "border-blue-500 bg-blue-50"
          : "border-gray-200 bg-white hover:border-gray-300"
      }`}
    >
      {/* drag grip or lock icon */}
      {locked ? (
        <span className="text-gray-400 text-xs" title="模板锁定">&#128274;</span>
      ) : draggable ? (
        <span className="text-gray-400 cursor-grab text-xs" title="拖拽排序">&#x2630;</span>
      ) : (
        <span className="text-gray-300 text-xs">&#x2630;</span>
      )}

      {/* index + label */}
      <span className="flex-1 text-sm text-gray-800 truncate">
        <span className="text-gray-400 mr-1">{index + 1}.</span>
        {label}
      </span>

      {/* up / down / delete — only for composable & unlocked */}
      {isComposable && !locked && (
        <>
          <button
            onClick={(e) => { e.stopPropagation(); onMoveUp(); }}
            disabled={index === 0}
            className="text-gray-400 hover:text-gray-700 disabled:opacity-30 text-xs px-1"
            title="上移"
          >&#9650;</button>
          <button
            onClick={(e) => { e.stopPropagation(); onMoveDown(); }}
            disabled={index === total - 1}
            className="text-gray-400 hover:text-gray-700 disabled:opacity-30 text-xs px-1"
            title="下移"
          >&#9660;</button>
          <button
            onClick={(e) => { e.stopPropagation(); onDelete(); }}
            className="text-gray-400 hover:text-red-600 text-sm px-1"
            title="删除"
          >&times;</button>
        </>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// VersionHistoryPanel — slide-out panel listing versions
// ---------------------------------------------------------------------------
function VersionHistoryPanel({
  pageId,
  onClose,
  onRollback,
}: {
  pageId: number;
  onClose: () => void;
  onRollback: (version: number) => void;
}) {
  const [versions, setVersions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listUnifiedPageVersions(pageId)
      .then((data: any) => {
        setVersions(Array.isArray(data) ? data : data?.items || []);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [pageId]);

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/30">
      <div className="w-96 bg-white h-full shadow-xl flex flex-col">
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200">
          <h3 className="text-base font-semibold text-gray-900">版本历史</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl">&times;</button>
        </div>
        <div className="flex-1 overflow-y-auto p-4">
          {loading ? (
            <div className="text-center text-gray-500 py-8">加载中...</div>
          ) : versions.length === 0 ? (
            <div className="text-center text-gray-500 py-8">暂无版本记录</div>
          ) : (
            <div className="space-y-2">
              {versions.map((v: any) => (
                <div
                  key={v.version ?? v.id}
                  className="flex items-center justify-between p-3 border border-gray-200 rounded-lg"
                >
                  <div>
                    <div className="text-sm font-medium text-gray-800">
                      版本 {v.version ?? v.id}
                    </div>
                    <div className="text-xs text-gray-500">
                      {v.createdAt ? new Date(v.createdAt).toLocaleString() : ""}
                    </div>
                  </div>
                  <button
                    onClick={() => onRollback(v.version ?? v.id)}
                    className="text-xs px-3 py-1 border border-blue-500 text-blue-600 rounded hover:bg-blue-50"
                  >
                    回滚
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ConflictDialog
// ---------------------------------------------------------------------------
function ConflictDialog({
  currentVersion,
  onReload,
  onDismiss,
}: {
  currentVersion: number;
  onReload: () => void;
  onDismiss: () => void;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-sm mx-4 p-6">
        <h3 className="text-lg font-semibold text-red-700 mb-2">版本冲突</h3>
        <p className="text-sm text-gray-600 mb-4">
          此页面已被他人编辑，当前服务端版本为 <strong>{currentVersion}</strong>。
          请重新加载后再编辑。
        </p>
        <div className="flex justify-end gap-2">
          <button
            onClick={onDismiss}
            className="px-4 py-2 text-sm text-gray-600 border border-gray-300 rounded hover:bg-gray-50"
          >
            关闭
          </button>
          <button
            onClick={onReload}
            className="px-4 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            重新加载
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main page component
// ---------------------------------------------------------------------------
export default function PageEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isNew = !id;
  const pageId = id ? Number(id) : 0;
  const { metas: sectionMetas } = useSectionRegistry();

  // -- page metadata --
  const [slug, setSlug] = useState("");
  const [zhTitle, setZhTitle] = useState("");
  const [enTitle, setEnTitle] = useState("");
  const [mode, setMode] = useState<"template" | "composable">("composable");
  const [showInNav, setShowInNav] = useState(false);
  const [sortOrder, setSortOrder] = useState(0);
  const [status, setStatus] = useState("draft");

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

      const config = draft.draftConfig as { sections?: any[] } | null;
      // Backend stores content in "props"; frontend SectionData uses "data" — normalize
      const loadedSections: SectionData[] = (config?.sections || []).map((s: any) => ({
        ...s,
        data: s.data || s.props || {},
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
      setSections((prev) => [...prev, newSection]);
      setSelectedIndex(sections.length);
      setShowPicker(false);
    },
    [sections.length],
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
    clearMessages();
    if (!slug.trim()) { setError("请输入 URL 路径"); return; }
    setSaving(true);
    try {
      const result = await createUnifiedPage({
        slug,
        zhTitle,
        enTitle,
        mode,
        showInNav,
        sortOrder,
        draftConfig: { sections },
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

  // -- publish --
  const handlePublish = async () => {
    clearMessages();
    setPublishing(true);
    try {
      await publishUnifiedPage(pageId, draftVersion);
      setStatus("published");
      setPublishedVersion(draftVersion);
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
    clearMessages();
    try {
      await unpublishUnifiedPage(pageId);
      setStatus("draft");
      setPublishedVersion(0);
      setSuccessMsg("已下线");
      setTimeout(() => setSuccessMsg(""), 3000);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || "下线失败");
    }
  };

  // -- rollback --
  const handleRollback = async (version: number) => {
    if (!window.confirm(`确定回滚到版本 ${version}？`)) return;
    clearMessages();
    try {
      await rollbackUnifiedPage(pageId, version);
      setShowHistory(false);
      await loadPage();
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

          {isNew ? (
            <button
              onClick={handleCreate}
              disabled={saving || !slug.trim()}
              className="px-4 py-1.5 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 disabled:opacity-50"
            >
              {saving ? "创建中..." : "创建"}
            </button>
          ) : (
            <>
              <button
                onClick={handleSave}
                disabled={saving}
                className="px-4 py-1.5 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 disabled:opacity-50"
              >
                {saving ? "保存中..." : "保存草稿"}
              </button>
              {status === "published" ? (
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
              )}
            </>
          )}
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

      {/* -- metadata section (collapsed for existing, expanded for new) -- */}
      {isNew && (
        <div className="bg-white rounded-lg shadow p-5 mb-4 flex-shrink-0">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">页面信息</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">URL 路径 (slug)</label>
              <div className="flex items-center">
                <span className="text-gray-400 text-sm mr-1">/</span>
                <input
                  type="text"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value)}
                  placeholder="about-us"
                  className="flex-1 border border-gray-300 rounded-md px-3 py-1.5 text-sm"
                />
              </div>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">标题 (中文)</label>
              <input
                type="text"
                value={zhTitle}
                onChange={(e) => setZhTitle(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">标题 (English)</label>
              <input
                type="text"
                value={enTitle}
                onChange={(e) => setEnTitle(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
              />
            </div>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-3">
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">页面模式</label>
              <select
                value={mode}
                onChange={(e) => setMode(e.target.value as "template" | "composable")}
                className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
              >
                <option value="composable">自由组合 (Composable)</option>
                <option value="template">模板 (Template)</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">排序</label>
              <input
                type="number"
                value={sortOrder}
                onChange={(e) => setSortOrder(Number(e.target.value))}
                className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm"
              />
            </div>
            <div className="flex items-end">
              <label className="flex items-center gap-2 text-sm text-gray-700 pb-1">
                <input
                  type="checkbox"
                  checked={showInNav}
                  onChange={(e) => setShowInNav(e.target.checked)}
                  className="rounded border-gray-300"
                />
                显示在导航
              </label>
            </div>
          </div>
        </div>
      )}

      {/* -- editor body -- */}
      {editorMode === "json" ? (
        /* JSON mode */
        <div className="bg-white rounded-lg shadow p-5 flex-1 flex flex-col min-h-0">
          <div className="flex items-center justify-between mb-2">
            <label className="text-sm font-medium text-gray-700">
              区块配置 (JSON 数组)
            </label>
            <span className="text-xs text-gray-400">
              可用类型: {sectionMetas.map((m) => m.type).join(", ")}
            </span>
          </div>
          <textarea
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
