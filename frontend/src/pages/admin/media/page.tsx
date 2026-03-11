import { useState, useEffect, useCallback } from "react";
import { listMedia, deleteMedia, uploadMedia, getMediaUsages, renameMedia } from "@/api/media";
import type { MediaItem, MediaUsage } from "@/api/media";
import ImageCropUpload from "@/components/admin/ImageCropUpload";
import RecropModal from "@/components/admin/RecropModal";

export default function MediaPage() {
  const [items, setItems] = useState<MediaItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [deleting, setDeleting] = useState<number | null>(null);
  const [showUpload, setShowUpload] = useState(false);
  const [cropItem, setCropItem] = useState<MediaItem | null>(null);
  const [usageItem, setUsageItem] = useState<MediaItem | null>(null);
  const [usages, setUsages] = useState<MediaUsage[]>([]);
  const [loadingUsages, setLoadingUsages] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editName, setEditName] = useState("");
  const [isDragging, setIsDragging] = useState(false);
  const [uploadingPaste, setUploadingPaste] = useState(false);
  const [uploadSuccess, setUploadSuccess] = useState(false);
  const pageSize = 20;

  const loadMedia = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listMedia(page, pageSize);
      setItems(data.items || []);
      setTotal(data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载图片列表失败");
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    loadMedia();
  }, [loadMedia]);

  const handleFilesUpload = useCallback(async (files: File[]) => {
    if (files.length === 0) return;
    setUploadingPaste(true);
    setUploadSuccess(false);
    setError(null);
    try {
      for (const file of files) {
        await uploadMedia(file);
      }
      setUploadSuccess(true);
      loadMedia();
      setTimeout(() => setUploadSuccess(false), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "上传失败");
    } finally {
      setUploadingPaste(false);
    }
  }, [loadMedia]);

  // Paste listener
  useEffect(() => {
    const handlePaste = (e: ClipboardEvent) => {
      const files: File[] = [];
      if (e.clipboardData?.items) {
        for (const item of e.clipboardData.items) {
          if (item.type.startsWith("image/")) {
            const file = item.getAsFile();
            if (file) files.push(file);
          }
        }
      }
      if (files.length > 0) {
        e.preventDefault();
        handleFilesUpload(files);
      }
    };
    document.addEventListener("paste", handlePaste);
    return () => document.removeEventListener("paste", handlePaste);
  }, [handleFilesUpload]);

  // Drag event handlers
  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.currentTarget === e.target || !e.currentTarget.contains(e.relatedTarget as Node)) {
      setIsDragging(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
    const files = Array.from(e.dataTransfer.files);
    if (files.length > 0) {
      handleFilesUpload(files);
    }
  };

  const handleDelete = async (item: MediaItem) => {
    if (!confirm(`确定要删除 ${item.filename} 吗？`)) return;

    setDeleting(item.id);
    setError(null);
    try {
      await deleteMedia(item.id);
      setItems((prev) => prev.filter((i) => i.id !== item.id));
      setTotal((prev) => prev - 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除失败");
    } finally {
      setDeleting(null);
    }
  };

  const handleCopyUrl = (url: string) => {
    navigator.clipboard.writeText(url).catch(() => {
      const input = document.createElement("input");
      input.value = url;
      document.body.appendChild(input);
      input.select();
      document.execCommand("copy");
      document.body.removeChild(input);
    });
  };

  const handleUpload = () => {
    setShowUpload(false);
    loadMedia();
  };

  const handleDirectUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setError(null);
    try {
      await uploadMedia(file);
      loadMedia();
    } catch (err) {
      setError(err instanceof Error ? err.message : "上传失败");
    }
    e.target.value = "";
  };

  const handleShowUsages = async (item: MediaItem) => {
    setUsageItem(item);
    setLoadingUsages(true);
    try {
      const data = await getMediaUsages(item.id);
      setUsages(data);
    } catch {
      setUsages([]);
    } finally {
      setLoadingUsages(false);
    }
  };

  const handleRename = async (item: MediaItem, newName: string) => {
    const trimmed = newName.trim();
    if (!trimmed || trimmed === item.filename) {
      setEditingId(null);
      return;
    }
    try {
      const updated = await renameMedia(item.id, trimmed);
      setItems((prev) => prev.map((i) => (i.id === updated.id ? updated : i)));
    } catch (err) {
      setError(err instanceof Error ? err.message : "重命名失败");
    }
    setEditingId(null);
  };

  const totalPages = Math.ceil(total / pageSize);

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const usageTypeLabel: Record<string, string> = {
    article: "文章",
    page: "页面",
    content_document: "内容",
  };

  return (
    <div
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      className="relative"
    >
      {/* Drag overlay */}
      {isDragging && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-blue-500/10 backdrop-blur-sm">
          <div className="border-4 border-dashed border-blue-500 rounded-2xl p-16 bg-white/80">
            <p className="text-xl font-semibold text-blue-600">松开鼠标上传文件</p>
          </div>
        </div>
      )}

      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">媒体管理</h1>
          <p className="text-sm text-gray-500 mt-1">共 {total} 个文件</p>
        </div>
        <div className="flex items-center gap-3">
          <label className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 cursor-pointer text-sm">
            直接上传
            <input
              type="file"
              accept="image/*,video/*,audio/*"
              onChange={handleDirectUpload}
              className="hidden"
            />
          </label>
          <button
            onClick={() => setShowUpload(!showUpload)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
          >
            裁剪上传
          </button>
        </div>
      </div>

      {/* Hint text */}
      <p className="text-xs text-gray-400 mb-4">
        提示: 可直接粘贴 (Ctrl+V) 或拖放图片到页面上传
      </p>

      {/* Upload in progress */}
      {uploadingPaste && (
        <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg text-blue-700 text-sm flex items-center gap-2">
          <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          正在上传...
        </div>
      )}

      {/* Upload success */}
      {uploadSuccess && (
        <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm">
          上传成功
        </div>
      )}

      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">
          {error}
        </div>
      )}

      {showUpload && (
        <div className="mb-6 p-6 bg-white shadow rounded-lg">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">裁剪上传</h2>
          <ImageCropUpload onUpload={handleUpload} />
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-600">加载中...</div>
        </div>
      ) : items.length === 0 ? (
        <div
          className="flex flex-col items-center justify-center h-64 border-2 border-dashed border-gray-300 rounded-lg hover:border-gray-400 transition-colors"
          onDragOver={handleDragOver}
          onDrop={handleDrop}
        >
          <svg className="w-12 h-12 text-gray-300 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <p className="text-gray-500 mb-1">暂无图片</p>
          <p className="text-sm text-gray-400">点击上传按钮、粘贴 (Ctrl+V) 或拖放文件到此处</p>
        </div>
      ) : (
        <>
          {/* Image grid */}
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
            {items.map((item) => (
              <div
                key={item.id}
                className="group relative bg-white rounded-lg shadow overflow-hidden"
              >
                {/* Thumbnail */}
                <div className="aspect-square bg-gray-100 flex items-center justify-center overflow-hidden">
                  {item.mimeType.startsWith("image/") ? (
                    <img
                      src={item.url}
                      alt={item.filename}
                      className="w-full h-full object-cover"
                      loading="lazy"
                    />
                  ) : (
                    <div className="flex flex-col items-center gap-2 text-gray-400">
                      <span className="text-4xl">
                        {item.mimeType.startsWith("video/") ? "🎬" : item.mimeType.startsWith("audio/") ? "🎵" : "📄"}
                      </span>
                      <span className="text-xs text-gray-500 uppercase">{item.mimeType.split("/")[1]}</span>
                    </div>
                  )}
                </div>

                {/* Info */}
                <div className="p-3">
                  {editingId === item.id ? (
                    <input
                      type="text"
                      value={editName}
                      onChange={(e) => setEditName(e.target.value)}
                      onBlur={() => handleRename(item, editName)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") handleRename(item, editName);
                        if (e.key === "Escape") setEditingId(null);
                      }}
                      className="text-xs font-medium text-gray-700 w-full border border-blue-400 rounded px-1.5 py-0.5 focus:outline-none focus:ring-1 focus:ring-blue-500"
                      autoFocus
                    />
                  ) : (
                    <p
                      className="text-xs font-medium text-gray-700 truncate cursor-pointer hover:text-blue-600"
                      title="点击重命名"
                      onClick={() => {
                        setEditingId(item.id);
                        setEditName(item.filename);
                      }}
                    >
                      {item.filename}
                    </p>
                  )}
                  <p className="text-xs text-gray-500 mt-1">
                    {formatFileSize(item.size)}
                    {item.width && item.height && ` · ${item.width}×${item.height}`}
                  </p>
                  <p className="text-xs text-gray-400 mt-0.5">
                    {new Date(item.createdAt).toLocaleDateString("zh-CN")}
                  </p>
                </div>

                {/* Hover actions */}
                <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex flex-col items-center justify-center gap-2">
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleCopyUrl(item.url)}
                      className="px-3 py-1.5 text-xs bg-white text-gray-800 rounded-md hover:bg-gray-100"
                    >
                      复制 URL
                    </button>
                    <button
                      onClick={() => setCropItem(item)}
                      className="px-3 py-1.5 text-xs bg-white text-gray-800 rounded-md hover:bg-gray-100"
                    >
                      裁剪
                    </button>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleShowUsages(item)}
                      className="px-3 py-1.5 text-xs bg-white text-gray-800 rounded-md hover:bg-gray-100"
                    >
                      查看引用
                    </button>
                    <button
                      onClick={() => handleDelete(item)}
                      disabled={deleting === item.id}
                      className="px-3 py-1.5 text-xs bg-red-600 text-white rounded-md hover:bg-red-700 disabled:opacity-50"
                    >
                      {deleting === item.id ? "删除中..." : "删除"}
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="mt-6 flex items-center justify-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="px-3 py-1.5 text-sm border rounded hover:bg-gray-50 disabled:opacity-50"
              >
                上一页
              </button>
              <span className="text-sm text-gray-600">
                {page} / {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="px-3 py-1.5 text-sm border rounded hover:bg-gray-50 disabled:opacity-50"
              >
                下一页
              </button>
            </div>
          )}
        </>
      )}

      {/* Recrop modal */}
      {cropItem && (
        <RecropModal
          item={cropItem}
          onClose={() => setCropItem(null)}
          onSuccess={(updated) => {
            setItems((prev) => prev.map((i) => (i.id === updated.id ? updated : i)));
            setCropItem(null);
          }}
        />
      )}

      {/* Usages modal */}
      {usageItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-xl shadow-xl w-[90vw] max-w-lg">
            <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-gray-900">图片引用 - {usageItem.filename}</h3>
              <button
                onClick={() => setUsageItem(null)}
                className="text-gray-400 hover:text-gray-600 text-xl"
              >
                &times;
              </button>
            </div>
            <div className="p-6 max-h-96 overflow-y-auto">
              {loadingUsages ? (
                <p className="text-gray-500">加载中...</p>
              ) : usages.length === 0 ? (
                <p className="text-gray-500">该图片未被任何页面或文章引用</p>
              ) : (
                <ul className="space-y-3">
                  {usages.map((u, i) => (
                    <li key={i} className="p-3 bg-gray-50 rounded-lg flex items-center gap-2">
                      <span className="text-xs font-medium px-2 py-0.5 rounded bg-blue-100 text-blue-800 whitespace-nowrap">
                        {usageTypeLabel[u.type] || u.type}
                      </span>
                      <span className="text-sm font-medium text-gray-900 truncate">{u.title}</span>
                      <span className="text-xs text-gray-500 whitespace-nowrap">({u.field})</span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
