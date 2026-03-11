import { useState, useRef, useCallback } from "react";
import Cropper, { ReactCropperElement } from "react-cropper";
import "cropperjs/dist/cropper.css";
import { recropMedia } from "@/api/media";
import type { MediaItem } from "@/api/media";
import {
  ZoomIn,
  ZoomOut,
  RotateCcw,
  RotateCw,
  RefreshCw,
} from "lucide-react";

interface RecropModalProps {
  item: MediaItem;
  onClose: () => void;
  onSuccess: (updated: MediaItem) => void;
}

export default function RecropModal({ item, onClose, onSuccess }: RecropModalProps) {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const cropperRef = useRef<ReactCropperElement>(null);

  const getCroppedBlob = useCallback((): Promise<Blob> => {
    return new Promise((resolve, reject) => {
      const cropper = cropperRef.current?.cropper;
      if (!cropper) {
        reject(new Error("Cropper not initialized"));
        return;
      }
      const canvas = cropper.getCroppedCanvas();
      if (!canvas) {
        reject(new Error("Failed to get cropped canvas"));
        return;
      }
      canvas.toBlob(
        (blob) => {
          if (blob) resolve(blob);
          else reject(new Error("Canvas toBlob failed"));
        },
        "image/jpeg",
        0.9
      );
    });
  }, []);

  const handleConfirm = async () => {
    setUploading(true);
    setError(null);
    try {
      const blob = await getCroppedBlob();
      const updated = await recropMedia(item.id, blob);
      onSuccess(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : "裁剪失败");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-xl shadow-xl w-[90vw] max-w-2xl">
        <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">重新裁剪 - {item.filename}</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl">&times;</button>
        </div>

        <div className="h-96 bg-gray-900">
          <Cropper
            ref={cropperRef}
            src={item.url}
            style={{ height: "100%", width: "100%" }}
            viewMode={1}
            guides={true}
            cropBoxMovable={true}
            cropBoxResizable={true}
            autoCropArea={0.8}
            zoomOnWheel={true}
            background={true}
            responsive={true}
            checkCrossOrigin={false}
          />
        </div>

        {/* Toolbar */}
        <div className="px-6 py-3 flex items-center justify-center gap-2 border-t border-gray-100">
          <button
            type="button"
            onClick={() => cropperRef.current?.cropper.zoom(-0.1)}
            className="p-2 rounded-md hover:bg-gray-100 text-gray-600"
            title="缩小"
          >
            <ZoomOut size={18} />
          </button>
          <button
            type="button"
            onClick={() => cropperRef.current?.cropper.zoom(0.1)}
            className="p-2 rounded-md hover:bg-gray-100 text-gray-600"
            title="放大"
          >
            <ZoomIn size={18} />
          </button>
          <div className="w-px h-5 bg-gray-300 mx-1" />
          <button
            type="button"
            onClick={() => cropperRef.current?.cropper.rotate(-90)}
            className="p-2 rounded-md hover:bg-gray-100 text-gray-600"
            title="左旋 90°"
          >
            <RotateCcw size={18} />
          </button>
          <button
            type="button"
            onClick={() => cropperRef.current?.cropper.rotate(90)}
            className="p-2 rounded-md hover:bg-gray-100 text-gray-600"
            title="右旋 90°"
          >
            <RotateCw size={18} />
          </button>
          <div className="w-px h-5 bg-gray-300 mx-1" />
          <button
            type="button"
            onClick={() => cropperRef.current?.cropper.reset()}
            className="p-2 rounded-md hover:bg-gray-100 text-gray-600"
            title="重置"
          >
            <RefreshCw size={18} />
          </button>
        </div>

        {error && (
          <div className="px-6 py-2">
            <p className="text-sm text-red-600">{error}</p>
          </div>
        )}

        <div className="px-6 py-4 border-t border-gray-200 flex items-center justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            disabled={uploading}
            className="px-4 py-2 text-sm border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
          >
            取消
          </button>
          <button
            type="button"
            onClick={handleConfirm}
            disabled={uploading}
            className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {uploading ? "处理中..." : "确认裁剪"}
          </button>
        </div>
      </div>
    </div>
  );
}
