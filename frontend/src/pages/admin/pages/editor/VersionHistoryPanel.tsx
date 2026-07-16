import { useState, useEffect } from "react";
import { listUnifiedPageVersions } from "@/api/unifiedPages";

// ---------------------------------------------------------------------------
// VersionHistoryPanel — slide-out panel listing versions
// ---------------------------------------------------------------------------
export function VersionHistoryPanel({
  pageId,
  onClose,
  onRollback,
  canRollback,
}: {
  pageId: number;
  onClose: () => void;
  onRollback: (version: number) => void;
  canRollback: boolean;
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
                  {canRollback && (
                    <button
                      onClick={() => onRollback(v.version ?? v.id)}
                      className="text-xs px-3 py-1 border border-blue-500 text-blue-600 rounded hover:bg-blue-50"
                    >
                      回滚
                    </button>
                  )}
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
export function ConflictDialog({
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
