import { useState, useEffect, useCallback } from "react";
import {
  getCategoryTree,
  createCategory,
  updateCategory,
  deleteCategory,
} from "@/api/articles";
import type { Category } from "@/api/articles";
import MetadataEditor from "@/components/admin/MetadataEditor";
import {
  AdminButton,
  AdminErrorBanner,
  AdminPageHeader,
  useAdminConfirm,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

// Flatten tree into a list with depth info for rendering
function flattenTree(cats: Category[], depth = 0): { cat: Category; depth: number }[] {
  const result: { cat: Category; depth: number }[] = [];
  for (const cat of cats) {
    result.push({ cat, depth });
    if (cat.children && cat.children.length > 0) {
      result.push(...flattenTree(cat.children, depth + 1));
    }
  }
  return result;
}

// Collect all categories as flat list for parent selector
function flattenAll(cats: Category[]): Category[] {
  const result: Category[] = [];
  for (const cat of cats) {
    result.push(cat);
    if (cat.children && cat.children.length > 0) {
      result.push(...flattenAll(cat.children));
    }
  }
  return result;
}

export default function CategoriesPage() {
  useDocumentTitle("分类管理");
  const { confirm, confirmDialog } = useAdminConfirm();

  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);

  // Inline edit state
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editSlug, setEditSlug] = useState("");
  const [editZhName, setEditZhName] = useState("");
  const [editEnName, setEditEnName] = useState("");
  const [editParentId, setEditParentId] = useState<number | null>(null);
  const [editCoverImage, setEditCoverImage] = useState("");
  const [editZhDescription, setEditZhDescription] = useState("");
  const [editEnDescription, setEditEnDescription] = useState("");
  const [editHideFromList, setEditHideFromList] = useState(false);
  const [editPreventCascade, setEditPreventCascade] = useState(false);
  const [editSortOrder, setEditSortOrder] = useState(0);
  const [editMetadata, setEditMetadata] = useState<Record<string, unknown>>({});
  const [saving, setSaving] = useState(false);

  // New category form
  const [showNew, setShowNew] = useState(false);
  const [newSlug, setNewSlug] = useState("");
  const [newZhName, setNewZhName] = useState("");
  const [newEnName, setNewEnName] = useState("");
  const [newParentId, setNewParentId] = useState<number | null>(null);
  const [newCoverImage, setNewCoverImage] = useState("");
  const [newZhDescription, setNewZhDescription] = useState("");
  const [newEnDescription, setNewEnDescription] = useState("");
  const [newHideFromList, setNewHideFromList] = useState(false);
  const [newPreventCascade, setNewPreventCascade] = useState(false);
  const [newSortOrder, setNewSortOrder] = useState(0);
  const [newMetadata, setNewMetadata] = useState<Record<string, unknown>>({});

  const loadCategories = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getCategoryTree();
      setCategories(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load categories");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  const allFlat = flattenAll(categories);
  const flatList = flattenTree(categories);

  const handleCreate = async () => {
    if (!newSlug.trim() || !newZhName.trim()) {
      setError("Slug and Chinese name are required");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      await createCategory({
        slug: newSlug,
        zhName: newZhName,
        enName: newEnName,
        parentId: newParentId,
        coverImage: newCoverImage,
        zhDescription: newZhDescription,
        enDescription: newEnDescription,
        hideFromList: newHideFromList,
        preventCascade: newPreventCascade,
        sortOrder: newSortOrder,
        metadata: newMetadata,
      });
      setShowNew(false);
      setNewSlug("");
      setNewZhName("");
      setNewEnName("");
      setNewParentId(null);
      setNewCoverImage("");
      setNewZhDescription("");
      setNewEnDescription("");
      setNewHideFromList(false);
      setNewPreventCascade(false);
      setNewSortOrder(0);
      setNewMetadata({});
      await loadCategories();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Create failed");
    } finally {
      setSaving(false);
    }
  };

  const startEdit = (cat: Category) => {
    setEditingId(cat.id);
    setEditSlug(cat.slug);
    setEditZhName(cat.zhName);
    setEditEnName(cat.enName);
    setEditParentId(cat.parentId ?? null);
    setEditCoverImage(cat.coverImage || "");
    setEditZhDescription(cat.zhDescription || "");
    setEditEnDescription(cat.enDescription || "");
    setEditHideFromList(cat.hideFromList || false);
    setEditPreventCascade(cat.preventCascade || false);
    setEditSortOrder(cat.sortOrder || 0);
    setEditMetadata(cat.metadata || {});
  };

  const handleUpdate = async () => {
    if (!editingId || !editSlug.trim() || !editZhName.trim()) {
      setError("Slug and Chinese name are required");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      await updateCategory(editingId, {
        slug: editSlug,
        zhName: editZhName,
        enName: editEnName,
        parentId: editParentId,
        coverImage: editCoverImage,
        zhDescription: editZhDescription,
        enDescription: editEnDescription,
        hideFromList: editHideFromList,
        preventCascade: editPreventCascade,
        sortOrder: editSortOrder,
        metadata: editMetadata,
      });
      setEditingId(null);
      await loadCategories();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Update failed");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (cat: Category) => {
    const name = cat.zhName || cat.enName;
    const ok = await confirm({
      title: "删除分类",
      message: `确定删除分类「${name}」吗？此操作不可撤销。`,
      confirmLabel: "删除",
      danger: true,
    });
    if (!ok) return;

    setDeleting(cat.id);
    setError(null);
    try {
      await deleteCategory(cat.id);
      await loadCategories();
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除失败");
    } finally {
      setDeleting(null);
    }
  };

  // Category form fields shared between new and edit
  const renderFormFields = (
    mode: "new" | "edit",
    vals: {
      slug: string; zhName: string; enName: string; parentId: number | null;
      coverImage: string; zhDescription: string; enDescription: string;
      hideFromList: boolean; preventCascade: boolean; sortOrder: number;
      metadata: Record<string, unknown>;
    },
    setters: {
      setSlug: (v: string) => void; setZhName: (v: string) => void; setEnName: (v: string) => void;
      setParentId: (v: number | null) => void; setCoverImage: (v: string) => void;
      setZhDescription: (v: string) => void; setEnDescription: (v: string) => void;
      setHideFromList: (v: boolean) => void; setPreventCascade: (v: boolean) => void;
      setSortOrder: (v: number) => void; setMetadata: (v: Record<string, unknown>) => void;
    },
    excludeId?: number,
  ) => (
    <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Slug</label>
          <input
            type="text"
            value={vals.slug}
            onChange={(e) => setters.setSlug(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
            placeholder="category-slug"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">中文名称</label>
          <input
            type="text"
            value={vals.zhName}
            onChange={(e) => setters.setZhName(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
            placeholder="Chinese name"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">English Name</label>
          <input
            type="text"
            value={vals.enName}
            onChange={(e) => setters.setEnName(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
            placeholder="English name"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">父级分类</label>
          <select
            value={vals.parentId ?? ""}
            onChange={(e) => setters.setParentId(e.target.value ? Number(e.target.value) : null)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
          >
            <option value="">无 (顶级分类)</option>
            {allFlat
              .filter((c) => c.id !== excludeId)
              .map((c) => (
                <option key={c.id} value={c.id}>
                  {c.zhName || c.enName}
                </option>
              ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">封面图片 URL</label>
          <input
            type="text"
            value={vals.coverImage}
            onChange={(e) => setters.setCoverImage(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
            placeholder="https://..."
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">排序</label>
          <input
            type="number"
            value={vals.sortOrder}
            onChange={(e) => setters.setSortOrder(Number(e.target.value))}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">中文描述</label>
          <textarea
            value={vals.zhDescription}
            onChange={(e) => setters.setZhDescription(e.target.value)}
            rows={2}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
            placeholder="Chinese description"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">English Description</label>
          <textarea
            value={vals.enDescription}
            onChange={(e) => setters.setEnDescription(e.target.value)}
            rows={2}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
            placeholder="English description"
          />
        </div>
      </div>

      <div className="flex items-center gap-6">
        <label className="flex items-center gap-2 text-sm text-gray-700">
          <input
            type="checkbox"
            checked={vals.hideFromList}
            onChange={(e) => setters.setHideFromList(e.target.checked)}
            className="rounded border-gray-300"
          />
          从列表隐藏
        </label>
        <label className="flex items-center gap-2 text-sm text-gray-700">
          <input
            type="checkbox"
            checked={vals.preventCascade}
            onChange={(e) => setters.setPreventCascade(e.target.checked)}
            className="rounded border-gray-300"
          />
          阻止级联
        </label>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">元数据</label>
        <MetadataEditor value={vals.metadata} onChange={setters.setMetadata} />
      </div>
    </div>
  );

  return (
    <div>
      {confirmDialog}
      <AdminPageHeader
        title="分类管理"
        description="管理文章分类与层级"
        breadcrumbs={[
          { label: "文章管理", to: "/admin/articles" },
          { label: "分类" },
        ]}
        actions={
          <AdminButton size="sm" onClick={() => setShowNew(!showNew)}>
            {showNew ? "取消" : "新建分类"}
          </AdminButton>
        }
      />

      {error && <AdminErrorBanner message={error} onDismiss={() => setError(null)} />}

      {/* New category form */}
      {showNew && (
        <div className="mb-6 rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-base font-semibold text-slate-900">新建分类</h2>
          {renderFormFields(
            "new",
            {
              slug: newSlug, zhName: newZhName, enName: newEnName, parentId: newParentId,
              coverImage: newCoverImage, zhDescription: newZhDescription, enDescription: newEnDescription,
              hideFromList: newHideFromList, preventCascade: newPreventCascade, sortOrder: newSortOrder,
              metadata: newMetadata,
            },
            {
              setSlug: setNewSlug, setZhName: setNewZhName, setEnName: setNewEnName,
              setParentId: setNewParentId, setCoverImage: setNewCoverImage,
              setZhDescription: setNewZhDescription, setEnDescription: setNewEnDescription,
              setHideFromList: setNewHideFromList, setPreventCascade: setNewPreventCascade,
              setSortOrder: setNewSortOrder, setMetadata: setNewMetadata,
            },
          )}
          <button
            onClick={handleCreate}
            disabled={saving}
            className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm disabled:opacity-50"
          >
            {saving ? "Creating..." : "Create"}
          </button>
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-600">Loading...</div>
        </div>
      ) : flatList.length === 0 ? (
        <div className="flex items-center justify-center h-64">
          <p className="text-gray-500">No categories yet. Create one above.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {flatList.map(({ cat, depth }) => (
            <div key={cat.id} className="bg-white shadow rounded-lg overflow-hidden">
              {editingId === cat.id ? (
                <div className="p-6">
                  <h3 className="text-sm font-semibold text-gray-900 mb-4">编辑分类</h3>
                  {renderFormFields(
                    "edit",
                    {
                      slug: editSlug, zhName: editZhName, enName: editEnName, parentId: editParentId,
                      coverImage: editCoverImage, zhDescription: editZhDescription, enDescription: editEnDescription,
                      hideFromList: editHideFromList, preventCascade: editPreventCascade, sortOrder: editSortOrder,
                      metadata: editMetadata,
                    },
                    {
                      setSlug: setEditSlug, setZhName: setEditZhName, setEnName: setEditEnName,
                      setParentId: setEditParentId, setCoverImage: setEditCoverImage,
                      setZhDescription: setEditZhDescription, setEnDescription: setEditEnDescription,
                      setHideFromList: setEditHideFromList, setPreventCascade: setEditPreventCascade,
                      setSortOrder: setEditSortOrder, setMetadata: setEditMetadata,
                    },
                    cat.id,
                  )}
                  <div className="mt-4 flex items-center gap-3">
                    <button
                      onClick={handleUpdate}
                      disabled={saving}
                      className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 text-sm disabled:opacity-50"
                    >
                      {saving ? "Saving..." : "Save"}
                    </button>
                    <button
                      onClick={() => setEditingId(null)}
                      className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ) : (
                <div
                  className="px-6 py-4 flex items-center justify-between hover:bg-gray-50"
                  style={{ paddingLeft: `${1.5 + depth * 1.5}rem` }}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    {depth > 0 && (
                      <span className="text-gray-300 text-sm">{"└─"}</span>
                    )}
                    {cat.coverImage && (
                      <img
                        src={cat.coverImage}
                        alt=""
                        className="w-8 h-8 rounded object-cover shrink-0"
                        onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
                      />
                    )}
                    <div className="min-w-0">
                      <div className="text-sm font-medium text-gray-900 truncate">
                        {cat.zhName || cat.enName}
                        {cat.enName && cat.zhName && (
                          <span className="text-gray-400 ml-2 font-normal">{cat.enName}</span>
                        )}
                      </div>
                      <div className="text-xs text-gray-500">{cat.slug}</div>
                    </div>
                    {cat.hideFromList && (
                      <span className="text-xs bg-gray-100 text-gray-500 px-1.5 py-0.5 rounded">隐藏</span>
                    )}
                    {cat.sortOrder > 0 && (
                      <span className="text-xs text-gray-400">#{cat.sortOrder}</span>
                    )}
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <button
                      onClick={() => startEdit(cat)}
                      className="text-blue-600 hover:text-blue-800 text-sm"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => handleDelete(cat)}
                      disabled={deleting === cat.id}
                      className="text-red-600 hover:text-red-800 text-sm disabled:opacity-50"
                    >
                      {deleting === cat.id ? "..." : "删除"}
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
