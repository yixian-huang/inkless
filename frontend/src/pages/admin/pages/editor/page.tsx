import { useState, useEffect, useCallback, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  getPage,
  createPage,
  updatePage,
  type CreatePageRequest,
} from "@/api/pages";
import { SectionRenderer } from "@/theme/sections";
import { getSectionSchema } from "@/theme/schemas";
import { useSectionRegistry } from "@/plugins/hooks";
import FieldRenderer from "@/components/admin/form-fields/FieldRenderer";
import MetadataEditor from "@/components/admin/MetadataEditor";
import type { SectionData } from "@/theme/types";

// ---------------------------------------------------------------------------
// SectionPicker -- modal overlay to add a new section type
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
          <h3 className="text-lg font-semibold text-gray-900">
            添加区块
          </h3>
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
// SectionListItem -- a single row in the section list
// ---------------------------------------------------------------------------
function SectionListItem({
  section,
  index,
  total,
  isSelected,
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

  return (
    <div
      draggable
      onDragStart={dragHandlers.onDragStart}
      onDragOver={dragHandlers.onDragOver}
      onDrop={dragHandlers.onDrop}
      onDragEnd={dragHandlers.onDragEnd}
      onClick={onSelect}
      className={`flex items-center gap-2 px-3 py-2 rounded-md cursor-pointer select-none border transition-colors ${
        isSelected
          ? "border-blue-500 bg-blue-50"
          : "border-gray-200 bg-white hover:border-gray-300"
      }`}
    >
      {/* drag grip */}
      <span className="text-gray-400 cursor-grab text-xs" title="拖拽排序">
        &#x2630;
      </span>

      {/* index + label */}
      <span className="flex-1 text-sm text-gray-800 truncate">
        <span className="text-gray-400 mr-1">{index + 1}.</span>
        {label}
      </span>

      {/* up / down / delete */}
      <button
        onClick={(e) => {
          e.stopPropagation();
          onMoveUp();
        }}
        disabled={index === 0}
        className="text-gray-400 hover:text-gray-700 disabled:opacity-30 text-xs px-1"
        title="上移"
      >
        &#9650;
      </button>
      <button
        onClick={(e) => {
          e.stopPropagation();
          onMoveDown();
        }}
        disabled={index === total - 1}
        className="text-gray-400 hover:text-gray-700 disabled:opacity-30 text-xs px-1"
        title="下移"
      >
        &#9660;
      </button>
      <button
        onClick={(e) => {
          e.stopPropagation();
          onDelete();
        }}
        className="text-gray-400 hover:text-red-600 text-sm px-1"
        title="删除"
      >
        &times;
      </button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// SectionList -- ordered list with drag-and-drop reordering
// ---------------------------------------------------------------------------
function SectionList({
  sections,
  selectedIndex,
  onSelect,
  onMove,
  onDelete,
  onAdd,
}: {
  sections: SectionData[];
  selectedIndex: number | null;
  onSelect: (i: number) => void;
  onMove: (from: number, to: number) => void;
  onDelete: (i: number) => void;
  onAdd: () => void;
}) {
  const dragIndexRef = useRef<number | null>(null);

  const makeDragHandlers = (index: number) => ({
    onDragStart: (e: React.DragEvent) => {
      dragIndexRef.current = index;
      e.dataTransfer.effectAllowed = "move";
    },
    onDragOver: (e: React.DragEvent) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = "move";
    },
    onDrop: (e: React.DragEvent) => {
      e.preventDefault();
      const from = dragIndexRef.current;
      if (from !== null && from !== index) {
        onMove(from, index);
      }
      dragIndexRef.current = null;
    },
    onDragEnd: () => {
      dragIndexRef.current = null;
    },
  });

  return (
    <div className="flex flex-col gap-2">
      {sections.map((section, i) => (
        <SectionListItem
          key={section.id}
          section={section}
          index={i}
          total={sections.length}
          isSelected={selectedIndex === i}
          onSelect={() => onSelect(i)}
          onMoveUp={() => {
            if (i > 0) onMove(i, i - 1);
          }}
          onMoveDown={() => {
            if (i < sections.length - 1) onMove(i, i + 1);
          }}
          onDelete={() => onDelete(i)}
          dragHandlers={makeDragHandlers(i)}
        />
      ))}

      <button
        onClick={onAdd}
        className="mt-1 flex items-center justify-center gap-1 px-3 py-2 border-2 border-dashed border-gray-300 rounded-md text-sm text-gray-500 hover:border-blue-400 hover:text-blue-600 transition-colors"
      >
        <span className="text-lg leading-none">+</span> 添加区块
      </button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// SectionEditorPanel -- form fields driven by section schema
// ---------------------------------------------------------------------------
function SectionEditorPanel({
  section,
  onChange,
}: {
  section: SectionData;
  onChange: (data: Record<string, unknown>) => void;
}) {
  const { metas } = useSectionRegistry();
  const schema = getSectionSchema(section.type, metas);
  const entries = Object.entries(schema);

  if (entries.length === 0) {
    return (
      <div className="text-sm text-gray-500 italic py-4">
        此区块类型尚无可编辑字段。
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {entries.map(([key, descriptor]) => (
        <FieldRenderer
          key={key}
          descriptor={descriptor}
          value={section.data[key]}
          onChange={(val) => {
            onChange({ ...section.data, [key]: val });
          }}
          path={key}
        />
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// PreviewPanel -- renders sections through SectionRenderer
// ---------------------------------------------------------------------------
function PreviewPanel({ sections }: { sections: SectionData[] }) {
  if (sections.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-sm text-gray-400 py-12">
        暂无区块，请在左侧添加。
      </div>
    );
  }

  return (
    <div className="border border-gray-200 rounded-md overflow-hidden bg-white">
      <div
        className="origin-top-left"
        style={{ transform: "scale(0.5)", transformOrigin: "top left", width: "200%" }}
      >
        {sections.map((s) => (
          <SectionRenderer key={s.id} section={s} />
        ))}
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
  const { metas: sectionMetas } = useSectionRegistry();

  // -- basic page fields --
  const [slug, setSlug] = useState("");
  const [titleZh, setTitleZh] = useState("");
  const [titleEn, setTitleEn] = useState("");
  const [template, setTemplate] = useState("default");
  const [seoTitleZh, setSeoTitleZh] = useState("");
  const [seoDescZh, setSeoDescZh] = useState("");

  // -- new page fields --
  const [coverImage, setCoverImage] = useState("");
  const [pageVisibility, setPageVisibility] = useState("public");
  const [pagePinned, setPagePinned] = useState(false);
  const [pageAllowComments, setPageAllowComments] = useState(false);
  const [pageAutoSummary, setPageAutoSummary] = useState(false);
  const [pageMetadata, setPageMetadata] = useState<Record<string, unknown>>({});

  // -- section visual editor state --
  const [sections, setSections] = useState<SectionData[]>([]);
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [showPicker, setShowPicker] = useState(false);
  const [activeTab, setActiveTab] = useState<"edit" | "preview">("edit");

  // -- mode toggle: visual vs JSON --
  const [editorMode, setEditorMode] = useState<"visual" | "json">("visual");
  const [configJson, setConfigJson] = useState("{\n  \"sections\": []\n}");

  // -- save state --
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  // -- load existing page --
  useEffect(() => {
    if (!id) return;
    getPage(Number(id)).then((page) => {
      setSlug(page.slug);
      setTitleZh(page.title?.zh || "");
      setTitleEn(page.title?.en || "");
      setTemplate(page.template || "default");
      setSeoTitleZh(page.seoTitle?.zh || "");
      setSeoDescZh(page.seoDescription?.zh || "");
      setCoverImage(page.coverImage || "");
      setPageVisibility(page.visibility || "public");
      setPagePinned(page.pinned || false);
      setPageAllowComments(page.allowComments || false);
      setPageAutoSummary(page.autoSummary || false);
      setPageMetadata(page.metadata || {});

      const config = page.config as { sections?: SectionData[] } | null;
      const loadedSections = config?.sections || [];
      setSections(loadedSections);
      setConfigJson(JSON.stringify(page.config, null, 2) || "{}");
    });
  }, [id]);

  // -- keep configJson in sync when sections change (visual -> json) --
  useEffect(() => {
    if (editorMode === "visual") {
      setConfigJson(JSON.stringify({ sections }, null, 2));
    }
  }, [sections, editorMode]);

  // -- section helpers --
  const addSection = useCallback(
    (type: string) => {
      const newSection: SectionData = {
        id: crypto.randomUUID(),
        type,
        data: {},
        settings: {},
      };
      setSections((prev) => [...prev, newSection]);
      setSelectedIndex(sections.length); // select the newly added one
      setShowPicker(false);
      setActiveTab("edit");
    },
    [sections.length],
  );

  const moveSection = useCallback(
    (from: number, to: number) => {
      setSections((prev) => {
        const next = [...prev];
        const [item] = next.splice(from, 1);
        next.splice(to, 0, item);
        return next;
      });
      setSelectedIndex(to);
    },
    [],
  );

  const deleteSection = useCallback(
    (index: number) => {
      if (!window.confirm("确定要删除此区块吗？")) return;
      setSections((prev) => prev.filter((_, i) => i !== index));
      setSelectedIndex((prev) => {
        if (prev === null) return null;
        if (prev === index) return null;
        if (prev > index) return prev - 1;
        return prev;
      });
    },
    [],
  );

  const updateSectionData = useCallback(
    (index: number, data: Record<string, unknown>) => {
      setSections((prev) =>
        prev.map((s, i) => (i === index ? { ...s, data } : s)),
      );
    },
    [],
  );

  // -- switch to visual mode: parse JSON into sections --
  const switchToVisual = useCallback(() => {
    try {
      const parsed = JSON.parse(configJson);
      const parsedSections: SectionData[] = parsed?.sections || [];
      setSections(parsedSections);
      setSelectedIndex(null);
      setEditorMode("visual");
    } catch {
      setError("JSON 格式错误，无法切换到可视化模式");
    }
  }, [configJson]);

  const switchToJson = useCallback(() => {
    setConfigJson(JSON.stringify({ sections }, null, 2));
    setEditorMode("json");
  }, [sections]);

  // -- save handler --
  const handleSave = async () => {
    setError("");
    setSaving(true);
    try {
      let config: unknown;
      if (editorMode === "json") {
        try {
          config = JSON.parse(configJson);
        } catch {
          setError("JSON 配置格式错误");
          setSaving(false);
          return;
        }
      } else {
        config = { sections };
      }

      const data: CreatePageRequest = {
        slug,
        title: { zh: titleZh, en: titleEn },
        template,
        config,
        seoTitle: { zh: seoTitleZh },
        seoDescription: { zh: seoDescZh },
        coverImage,
        visibility: pageVisibility,
        pinned: pagePinned,
        allowComments: pageAllowComments,
        autoSummary: pageAutoSummary,
        metadata: pageMetadata,
      };

      if (isNew) {
        await createPage(data);
      } else {
        await updatePage(Number(id), data);
      }
      navigate("/admin/pages");
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "保存失败";
      setError(msg);
    } finally {
      setSaving(false);
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

  return (
    <div>
      {/* -- header -- */}
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">
          {isNew ? "新建页面" : "编辑页面"}
        </h2>
        <button
          onClick={() => navigate("/admin/pages")}
          className="text-sm text-gray-600 hover:text-gray-900"
        >
          返回列表
        </button>
      </div>

      {/* -- error banner -- */}
      {error && (
        <div className="mb-4 p-3 bg-red-50 text-red-700 rounded-md text-sm">
          {error}
        </div>
      )}

      {/* -- basic fields card -- */}
      <div className="bg-white rounded-lg shadow p-6 space-y-5 mb-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              URL 路径 (slug)
            </label>
            <div className="flex items-center">
              <span className="text-gray-400 text-sm mr-1">/</span>
              <input
                type="text"
                value={slug}
                onChange={(e) => setSlug(e.target.value)}
                placeholder="about-us"
                className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              布局模板
            </label>
            <select
              value={template}
              onChange={(e) => setTemplate(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            >
              <option value="default">默认 (Default)</option>
              <option value="fullwidth">全宽 (Fullwidth)</option>
              <option value="sidebar">侧边栏 (Sidebar)</option>
              <option value="landing">落地页 (Landing)</option>
              <option value="blank">空白 (Blank)</option>
            </select>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              页面标题 (中文)
            </label>
            <input
              type="text"
              value={titleZh}
              onChange={(e) => setTitleZh(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              页面标题 (English)
            </label>
            <input
              type="text"
              value={titleEn}
              onChange={(e) => setTitleEn(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              SEO 标题
            </label>
            <input
              type="text"
              value={seoTitleZh}
              onChange={(e) => setSeoTitleZh(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              SEO 描述
            </label>
            <input
              type="text"
              value={seoDescZh}
              onChange={(e) => setSeoDescZh(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              封面图片 URL
            </label>
            <input
              type="text"
              value={coverImage}
              onChange={(e) => setCoverImage(e.target.value)}
              placeholder="https://..."
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
            {coverImage && (
              <img
                src={coverImage}
                alt="Cover preview"
                className="mt-2 max-h-32 rounded border border-gray-200"
                onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
              />
            )}
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              可见性 (Visibility)
            </label>
            <select
              value={pageVisibility}
              onChange={(e) => setPageVisibility(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            >
              <option value="public">公开 (Public)</option>
              <option value="private">私密 (Private)</option>
              <option value="password_protected">密码保护 (Password Protected)</option>
            </select>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-6">
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={pagePinned}
              onChange={(e) => setPagePinned(e.target.checked)}
              className="rounded border-gray-300"
            />
            置顶 (Pinned)
          </label>
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={pageAllowComments}
              onChange={(e) => setPageAllowComments(e.target.checked)}
              className="rounded border-gray-300"
            />
            允许评论 (Allow Comments)
          </label>
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={pageAutoSummary}
              onChange={(e) => setPageAutoSummary(e.target.checked)}
              className="rounded border-gray-300"
            />
            自动摘要 (Auto Summary)
          </label>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            元数据 (Metadata)
          </label>
          <MetadataEditor value={pageMetadata} onChange={setPageMetadata} />
        </div>
      </div>

      {/* -- mode toggle bar -- */}
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-gray-700">页面内容</h3>
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-500">编辑器模式:</span>
          <button
            onClick={() =>
              editorMode === "visual" ? switchToJson() : switchToVisual()
            }
            className={`px-3 py-1 text-xs rounded-md border transition-colors ${
              editorMode === "json"
                ? "bg-gray-800 text-white border-gray-800"
                : "bg-white text-gray-600 border-gray-300 hover:border-gray-400"
            }`}
          >
            JSON
          </button>
          <button
            onClick={() =>
              editorMode === "json" ? switchToVisual() : undefined
            }
            className={`px-3 py-1 text-xs rounded-md border transition-colors ${
              editorMode === "visual"
                ? "bg-blue-600 text-white border-blue-600"
                : "bg-white text-gray-600 border-gray-300 hover:border-gray-400"
            }`}
          >
            可视化
          </button>
        </div>
      </div>

      {/* -- JSON mode -- */}
      {editorMode === "json" && (
        <div className="bg-white rounded-lg shadow p-6 space-y-5">
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="block text-sm font-medium text-gray-700">
                页面配置 (JSON)
              </label>
              <span className="text-xs text-gray-400">
                可用 Section 类型：
                {sectionMetas.map((m) => m.type).join(", ")}
              </span>
            </div>
            <textarea
              value={configJson}
              onChange={(e) => setConfigJson(e.target.value)}
              rows={16}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono"
              spellCheck={false}
            />
          </div>

          <div className="flex justify-end">
            <button
              onClick={handleSave}
              disabled={saving || !slug}
              className="px-6 py-2 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700 disabled:opacity-50"
            >
              {saving ? "保存中..." : "保存"}
            </button>
          </div>
        </div>
      )}

      {/* -- Visual mode -- */}
      {editorMode === "visual" && (
        <div className="bg-white rounded-lg shadow">
          <div className="grid grid-cols-1 lg:grid-cols-3 min-h-[400px]">
            {/* Left panel: section list */}
            <div className="lg:col-span-1 border-r border-gray-200 p-4">
              <SectionList
                sections={sections}
                selectedIndex={selectedIndex}
                onSelect={setSelectedIndex}
                onMove={moveSection}
                onDelete={deleteSection}
                onAdd={() => setShowPicker(true)}
              />
            </div>

            {/* Right panel: editor / preview */}
            <div className="lg:col-span-2 p-4 flex flex-col">
              {/* Tab bar */}
              <div className="flex items-center border-b border-gray-200 mb-4">
                <button
                  onClick={() => setActiveTab("edit")}
                  className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors -mb-px ${
                    activeTab === "edit"
                      ? "border-blue-600 text-blue-600"
                      : "border-transparent text-gray-500 hover:text-gray-700"
                  }`}
                >
                  编辑
                </button>
                <button
                  onClick={() => setActiveTab("preview")}
                  className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors -mb-px ${
                    activeTab === "preview"
                      ? "border-blue-600 text-blue-600"
                      : "border-transparent text-gray-500 hover:text-gray-700"
                  }`}
                >
                  预览
                </button>
              </div>

              {/* Tab content */}
              {activeTab === "edit" && (
                <div className="flex-1 overflow-y-auto">
                  {selectedSection ? (
                    <div>
                      <h4 className="text-sm font-semibold text-gray-800 mb-3">
                        {selectedMeta?.labelZh || selectedSection.type}
                        <span className="text-xs text-gray-400 ml-2">
                          {selectedMeta?.label}
                        </span>
                      </h4>
                      <SectionEditorPanel
                        section={selectedSection}
                        onChange={(data) =>
                          updateSectionData(selectedIndex!, data)
                        }
                      />
                    </div>
                  ) : (
                    <div className="flex items-center justify-center h-full text-sm text-gray-400 py-12">
                      {sections.length === 0
                        ? "点击「+ 添加区块」开始构建页面。"
                        : "请在左侧选择一个区块进行编辑。"}
                    </div>
                  )}
                </div>
              )}

              {activeTab === "preview" && (
                <div className="flex-1 overflow-y-auto">
                  <PreviewPanel sections={sections} />
                </div>
              )}
            </div>
          </div>

          {/* Save button */}
          <div className="border-t border-gray-200 px-6 py-4 flex justify-end">
            <button
              onClick={handleSave}
              disabled={saving || !slug}
              className="px-6 py-2 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700 disabled:opacity-50"
            >
              {saving ? "保存中..." : "保存"}
            </button>
          </div>
        </div>
      )}

      {/* -- Section picker modal -- */}
      {showPicker && (
        <SectionPicker onSelect={addSection} onClose={() => setShowPicker(false)} />
      )}
    </div>
  );
}
