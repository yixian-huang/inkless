import { useState, useEffect, useCallback } from "react";
import { listMedia, uploadMedia } from "@/api/media";
import type { MediaItem } from "@/api/media";

interface MediaPickerModalProps {
  open: boolean;
  onClose: () => void;
  onSelect: (item: MediaItem) => void;
  accept?: string;
  type?: string;
  title?: string;
}

export default function MediaPickerModal({
  open,
  onClose,
  onSelect,
  accept = "image/*,video/*,audio/*",
  type,
  title = "选择媒体",
}: MediaPickerModalProps) {
  const [items, setItems] = useState<MediaItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 12;

  const loadMedia = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listMedia(page, pageSize, type);
      setItems(data.items || []);
      setTotal(data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, [page, type]);

  useEffect(() => {
    if (open) {
      setPage(1);
      loadMedia();
    }
  }, [open, loadMedia]);

  useEffect(() => {
    if (open) loadMedia();
  }, [page, open, loadMedia]);

  const handleDirectUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setError(null);
    try {
      const item = await uploadMedia(file);
      onSelect(item);
    } catch (err) {
      setError(err instanceof Error ? err.message : "上传失败");
    }
    e.target.value = "";
  };

  const isImage = (item: MediaItem) => item.mimeType.startsWith("image/");
  const totalPages = Math.ceil(total / pageSize);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-xl shadow-xl w-[90vw] max-w-4xl max-h-[80vh] flex flex-col">
        <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
          <div className="flex items-center gap-3">
            <label className="px-3 py-1.5 text-xs border border-gray-300 rounded hover:bg-gray-50 cursor-pointer">
              上传
              <input type="file" accept={accept} onChange={handleDirectUpload} className="hidden" />
            </label>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl leading-none">&times;</button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-6">
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-800 text-sm">{error}</div>
          )}
          {loading ? (
            <div className="flex items-center justify-center h-48">
              <div className="text-gray-600">加载中...</div>
            </div>
          ) : items.length === 0 ? (
            <div className="flex items-center justify-center h-48">
              <p className="text-gray-500">暂无文件</p>
            </div>
          ) : (
            <div className="grid grid-cols-3 md:grid-cols-4 gap-3">
              {items.map((item) => (
                <button
                  key={item.id}
                  onClick={() => onSelect(item)}
                  className="group relative aspect-square bg-gray-100 rounded-lg overflow-hidden border-2 border-transparent hover:border-blue-500 transition-colors flex items-center justify-center"
                >
                  {isImage(item) ? (
                    <img src={item.url} alt={item.filename} className="w-full h-full object-cover" loading="lazy" />
                  ) : (
                    <div className="flex flex-col items-center gap-2 text-gray-400">
                      <span className="text-3xl">{item.mimeType.startsWith("video/") ? "\uD83C\uDFAC" : "\uD83C\uDFB5"}</span>
                      <span className="text-xs truncate max-w-full px-2">{item.filename}</span>
                    </div>
                  )}
                  <div className="absolute bottom-0 inset-x-0 bg-gradient-to-t from-black/60 to-transparent p-2">
                    <p className="text-xs text-white truncate">{item.filename}</p>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>

        {totalPages > 1 && (
          <div className="px-6 py-3 border-t border-gray-200 flex items-center justify-center gap-2">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-3 py-1 text-sm border rounded hover:bg-gray-50 disabled:opacity-50"
            >
              上一页
            </button>
            <span className="text-sm text-gray-600">{page} / {totalPages}</span>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="px-3 py-1 text-sm border rounded hover:bg-gray-50 disabled:opacity-50"
            >
              下一页
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
