import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { getTags, createTag, updateTag, deleteTag } from "@/api/articles";
import type { Tag } from "@/api/articles";
import MetadataEditor from "@/components/admin/MetadataEditor";

export default function TagsPage() {
  const navigate = useNavigate();

  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);
  const [saving, setSaving] = useState(false);

  // New tag form
  const [showNew, setShowNew] = useState(false);
  const [newSlug, setNewSlug] = useState("");
  const [newZhName, setNewZhName] = useState("");
  const [newEnName, setNewEnName] = useState("");
  const [newColor, setNewColor] = useState("#6B7280");
  const [newCoverImage, setNewCoverImage] = useState("");
  const [newMetadata, setNewMetadata] = useState<Record<string, unknown>>({});

  // Edit tag state
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editSlug, setEditSlug] = useState("");
  const [editZhName, setEditZhName] = useState("");
  const [editEnName, setEditEnName] = useState("");
  const [editColor, setEditColor] = useState("#6B7280");
  const [editCoverImage, setEditCoverImage] = useState("");
  const [editMetadata, setEditMetadata] = useState<Record<string, unknown>>({});

  const loadTags = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getTags();
      setTags(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load tags");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTags();
  }, [loadTags]);

  const handleCreate = async () => {
    if (!newSlug.trim() || !newZhName.trim()) {
      setError("Slug and Chinese name are required");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      const created = await createTag({
        slug: newSlug,
        zhName: newZhName,
        enName: newEnName,
        color: newColor,
        coverImage: newCoverImage,
        metadata: newMetadata,
      });
      setTags((prev) => [...prev, created]);
      setShowNew(false);
      setNewSlug("");
      setNewZhName("");
      setNewEnName("");
      setNewColor("#6B7280");
      setNewCoverImage("");
      setNewMetadata({});
    } catch (err) {
      setError(err instanceof Error ? err.message : "Create failed");
    } finally {
      setSaving(false);
    }
  };

  const startEdit = (tag: Tag) => {
    setEditingId(tag.id);
    setEditSlug(tag.slug);
    setEditZhName(tag.zhName);
    setEditEnName(tag.enName);
    setEditColor(tag.color || "#6B7280");
    setEditCoverImage(tag.coverImage || "");
    setEditMetadata(tag.metadata || {});
  };

  const handleUpdate = async () => {
    if (!editingId || !editSlug.trim() || !editZhName.trim()) {
      setError("Slug and Chinese name are required");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      const updated = await updateTag(editingId, {
        slug: editSlug,
        zhName: editZhName,
        enName: editEnName,
        color: editColor,
        coverImage: editCoverImage,
        metadata: editMetadata,
      });
      setTags((prev) => prev.map((t) => (t.id === editingId ? updated : t)));
      setEditingId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Update failed");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (tag: Tag) => {
    if (!confirm(`Delete tag "${tag.zhName || tag.enName}"?`)) return;

    setDeleting(tag.id);
    setError(null);
    try {
      await deleteTag(tag.id);
      setTags((prev) => prev.filter((t) => t.id !== tag.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Delete failed");
    } finally {
      setDeleting(null);
    }
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <button
            onClick={() => navigate("/admin/articles")}
            className="text-blue-600 hover:text-blue-800 mb-2 inline-flex items-center text-sm"
          >
            &larr; Back to Articles
          </button>
          <h1 className="text-2xl font-bold text-gray-900">Tags</h1>
        </div>
        <button
          onClick={() => setShowNew(!showNew)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
        >
          {showNew ? "Cancel" : "New Tag"}
        </button>
      </div>

      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">
          {error}
        </div>
      )}

      {/* New tag form */}
      {showNew && (
        <div className="mb-6 bg-white shadow rounded-lg p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">New Tag</h2>
          <div className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Slug</label>
                <input
                  type="text"
                  value={newSlug}
                  onChange={(e) => setNewSlug(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
                  placeholder="tag-slug"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">中文名称</label>
                <input
                  type="text"
                  value={newZhName}
                  onChange={(e) => setNewZhName(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
                  placeholder="Chinese name"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">English Name</label>
                <input
                  type="text"
                  value={newEnName}
                  onChange={(e) => setNewEnName(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
                  placeholder="English name"
                />
              </div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">颜色</label>
                <div className="flex items-center gap-2">
                  <input
                    type="color"
                    value={newColor}
                    onChange={(e) => setNewColor(e.target.value)}
                    className="w-10 h-10 border border-gray-300 rounded cursor-pointer"
                  />
                  <input
                    type="text"
                    value={newColor}
                    onChange={(e) => setNewColor(e.target.value)}
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm"
                    placeholder="#6B7280"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">封面图片 URL</label>
                <input
                  type="text"
                  value={newCoverImage}
                  onChange={(e) => setNewCoverImage(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
                  placeholder="https://..."
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">元数据</label>
              <MetadataEditor value={newMetadata} onChange={setNewMetadata} />
            </div>
          </div>
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
      ) : tags.length === 0 ? (
        <div className="flex items-center justify-center h-64">
          <p className="text-gray-500">No tags yet. Create one above.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {tags.map((tag) => (
            <div
              key={tag.id}
              className="bg-white shadow rounded-lg overflow-hidden"
            >
              {editingId === tag.id ? (
                <div className="p-4 space-y-3">
                  <div>
                    <label className="block text-xs font-medium text-gray-500 mb-1">Slug</label>
                    <input
                      type="text"
                      value={editSlug}
                      onChange={(e) => setEditSlug(e.target.value)}
                      className="w-full px-2 py-1.5 border border-gray-300 rounded text-sm"
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="block text-xs font-medium text-gray-500 mb-1">中文名称</label>
                      <input
                        type="text"
                        value={editZhName}
                        onChange={(e) => setEditZhName(e.target.value)}
                        className="w-full px-2 py-1.5 border border-gray-300 rounded text-sm"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-500 mb-1">English</label>
                      <input
                        type="text"
                        value={editEnName}
                        onChange={(e) => setEditEnName(e.target.value)}
                        className="w-full px-2 py-1.5 border border-gray-300 rounded text-sm"
                      />
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-500 mb-1">颜色</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="color"
                        value={editColor}
                        onChange={(e) => setEditColor(e.target.value)}
                        className="w-8 h-8 border border-gray-300 rounded cursor-pointer"
                      />
                      <input
                        type="text"
                        value={editColor}
                        onChange={(e) => setEditColor(e.target.value)}
                        className="flex-1 px-2 py-1.5 border border-gray-300 rounded text-sm"
                      />
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-500 mb-1">封面图片 URL</label>
                    <input
                      type="text"
                      value={editCoverImage}
                      onChange={(e) => setEditCoverImage(e.target.value)}
                      className="w-full px-2 py-1.5 border border-gray-300 rounded text-sm"
                      placeholder="https://..."
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-500 mb-1">元数据</label>
                    <MetadataEditor value={editMetadata} onChange={setEditMetadata} />
                  </div>
                  <div className="flex items-center gap-2 pt-1">
                    <button
                      onClick={handleUpdate}
                      disabled={saving}
                      className="px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700 disabled:opacity-50"
                    >
                      {saving ? "Saving..." : "Save"}
                    </button>
                    <button
                      onClick={() => setEditingId(null)}
                      className="px-3 py-1.5 border border-gray-300 rounded text-sm hover:bg-gray-50"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ) : (
                <>
                  {tag.coverImage && (
                    <img
                      src={tag.coverImage}
                      alt=""
                      className="w-full h-32 object-cover"
                      onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
                    />
                  )}
                  <div className="p-4">
                    <div className="flex items-center gap-2 mb-2">
                      <span
                        className="w-4 h-4 rounded-full shrink-0 border border-gray-200"
                        style={{ backgroundColor: tag.color || "#6B7280" }}
                      />
                      <span className="text-sm font-medium text-gray-900 truncate">
                        {tag.zhName || tag.enName}
                      </span>
                    </div>
                    {tag.enName && tag.zhName && (
                      <div className="text-xs text-gray-500 mb-1">{tag.enName}</div>
                    )}
                    <div className="text-xs text-gray-400 mb-3">{tag.slug}</div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => startEdit(tag)}
                        className="text-blue-600 hover:text-blue-800 text-sm"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => handleDelete(tag)}
                        disabled={deleting === tag.id}
                        className="text-red-600 hover:text-red-800 text-sm disabled:opacity-50"
                      >
                        {deleting === tag.id ? "..." : "Delete"}
                      </button>
                    </div>
                  </div>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
