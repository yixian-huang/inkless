import { useCallback, useEffect, useState } from "react";
import {
  compareArticleVersions,
  getArticleVersion,
  listArticleVersions,
  type ArticleVersionDetail,
  type ArticleVersionListItem,
} from "@/api/articles";
import { FieldDiff } from "./components/TextDiffView";
import { snapStr } from "./utils/snapshot";

const ACTION_LABEL: Record<string, string> = {
  create: "创建",
  save: "保存",
  publish: "发布",
  update: "更新",
  current: "当前编辑",
  restore: "恢复",
};

/** Snapshot shape used for compare/restore (subset of article fields). */
export type ArticleDraftSnapshot = {
  zhTitle?: string;
  enTitle?: string;
  slug?: string;
  status?: string;
  zhBody?: string;
  enBody?: string;
  coverImage?: string;
  zhSeoTitle?: string;
  enSeoTitle?: string;
  zhMetaDescription?: string;
  enMetaDescription?: string;
  ogImage?: string;
  author?: string;
  [key: string]: unknown;
};

function formatTime(iso: string) {
  try {
    return new Date(iso).toLocaleString("zh-CN");
  } catch {
    return iso;
  }
}

function syntheticCurrentDetail(
  articleId: number,
  draft: ArticleDraftSnapshot,
): ArticleVersionDetail {
  return {
    id: 0,
    articleId,
    version: 0,
    snapshot: { ...draft },
    action: "current",
    summary: "当前编辑器内容",
    createdBy: 0,
    createdAt: new Date().toISOString(),
  };
}

export function ArticleVersionHistoryPanel({
  articleId,
  onClose,
  currentDraft,
  onRestore,
  canRestore = true,
}: {
  articleId: number;
  onClose: () => void;
  /** Live editor snapshot for "compare with current". */
  currentDraft?: ArticleDraftSnapshot | null;
  /** Apply a historical snapshot into the editor (caller marks dirty). */
  onRestore?: (snapshot: ArticleDraftSnapshot) => void;
  canRestore?: boolean;
}) {
  const [versions, setVersions] = useState<ArticleVersionListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [leftVer, setLeftVer] = useState<number | null>(null);
  /** null version number means "current draft" when supported */
  const [rightVer, setRightVer] = useState<number | "current" | null>(null);
  const [comparing, setComparing] = useState(false);
  const [restoring, setRestoring] = useState(false);
  const [compareResult, setCompareResult] = useState<{
    left: ArticleVersionDetail;
    right: ArticleVersionDetail;
  } | null>(null);
  const [view, setView] = useState<"list" | "compare">("list");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listArticleVersions(articleId, 1, 50);
      const items = data.items || [];
      setVersions(items);
      if (items.length >= 1) {
        setLeftVer(items[0].version);
        setRightVer(currentDraft ? "current" : items.length >= 2 ? items[1].version : items[0].version);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载版本失败");
    } finally {
      setLoading(false);
    }
  }, [articleId, currentDraft]);

  useEffect(() => {
    void load();
  }, [load]);

  const runCompare = async () => {
    if (leftVer == null || rightVer == null) return;
    setComparing(true);
    setError(null);
    try {
      if (rightVer === "current") {
        if (!currentDraft) {
          setError("无法获取当前编辑内容");
          return;
        }
        const left = await getArticleVersion(articleId, leftVer);
        setCompareResult({
          left,
          right: syntheticCurrentDetail(articleId, currentDraft),
        });
      } else {
        const result = await compareArticleVersions(articleId, leftVer, rightVer);
        setCompareResult(result);
      }
      setView("compare");
    } catch (err) {
      setError(err instanceof Error ? err.message : "比对失败");
    } finally {
      setComparing(false);
    }
  };

  const handleRestore = async (version: number) => {
    if (!onRestore || !canRestore) return;
    if (
      !window.confirm(
        `确定将编辑器恢复到版本 v${version}？\n当前未保存的修改将被覆盖（恢复后仍需手动保存才会写回服务器）。`,
      )
    ) {
      return;
    }
    setRestoring(true);
    setError(null);
    try {
      const detail = await getArticleVersion(articleId, version);
      onRestore(detail.snapshot as ArticleDraftSnapshot);
    } catch (err) {
      setError(err instanceof Error ? err.message : "恢复失败");
    } finally {
      setRestoring(false);
    }
  };

  const leftSnap = compareResult?.left.snapshot;
  const rightSnap = compareResult?.right.snapshot;
  const compareRightIsCurrent = compareResult?.right.action === "current";

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/30" onClick={onClose}>
      <div
        className={`bg-white h-full shadow-xl flex flex-col ${
          view === "compare" ? "w-full max-w-4xl" : "w-full max-w-md"
        }`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 flex-shrink-0">
          <div className="flex items-center gap-2">
            {view === "compare" && (
              <button
                type="button"
                onClick={() => setView("list")}
                className="text-sm text-gray-500 hover:text-gray-800 mr-1"
              >
                ← 返回
              </button>
            )}
            <h3 className="text-base font-semibold text-gray-900">
              {view === "compare" ? "版本比对" : "历史版本"}
            </h3>
          </div>
          <button type="button" onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl leading-none">
            &times;
          </button>
        </div>

        {error && (
          <div className="px-4 py-2 bg-red-50 text-red-700 text-sm border-b border-red-100 flex-shrink-0">
            {error}
          </div>
        )}

        {view === "list" ? (
          <>
            <div className="px-4 py-3 border-b border-gray-100 space-y-2 flex-shrink-0 bg-gray-50">
              <p className="text-xs text-gray-500">
                选择两个版本比对；右侧可选「当前编辑」对比未保存内容。
              </p>
              <div className="flex items-center gap-2">
                <select
                  value={leftVer ?? ""}
                  onChange={(e) => setLeftVer(Number(e.target.value))}
                  className="flex-1 text-sm border border-gray-300 rounded-lg px-2 py-1.5"
                  disabled={versions.length === 0}
                >
                  {versions.map((v) => (
                    <option key={`L-${v.version}`} value={v.version}>
                      v{v.version} · {ACTION_LABEL[v.action] || v.action} · {formatTime(v.createdAt)}
                    </option>
                  ))}
                </select>
                <span className="text-xs text-gray-400">vs</span>
                <select
                  value={rightVer === "current" ? "current" : (rightVer ?? "")}
                  onChange={(e) => {
                    const v = e.target.value;
                    setRightVer(v === "current" ? "current" : Number(v));
                  }}
                  className="flex-1 text-sm border border-gray-300 rounded-lg px-2 py-1.5"
                  disabled={versions.length === 0 && !currentDraft}
                >
                  {currentDraft && (
                    <option value="current">当前编辑（含未保存）</option>
                  )}
                  {versions.map((v) => (
                    <option key={`R-${v.version}`} value={v.version}>
                      v{v.version} · {ACTION_LABEL[v.action] || v.action} · {formatTime(v.createdAt)}
                    </option>
                  ))}
                </select>
              </div>
              <button
                type="button"
                onClick={() => void runCompare()}
                disabled={
                  comparing ||
                  leftVer == null ||
                  rightVer == null ||
                  (versions.length === 0 && rightVer !== "current")
                }
                className="w-full px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                {comparing ? "比对中…" : "开始比对"}
              </button>
            </div>

            <div className="flex-1 overflow-y-auto p-4">
              {loading ? (
                <div className="text-center text-gray-500 py-8">加载中…</div>
              ) : versions.length === 0 ? (
                <div className="text-center text-gray-500 py-8 text-sm">
                  暂无版本记录
                  <p className="mt-1 text-xs text-gray-400">保存或发布文章后会自动生成版本快照。</p>
                </div>
              ) : (
                <div className="space-y-2">
                  {versions.map((v) => (
                    <div
                      key={v.id}
                      className="p-3 border border-gray-200 rounded-lg hover:border-blue-200 transition-colors"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="text-sm font-medium text-gray-800">
                          版本 {v.version}
                          <span className="ml-2 text-[10px] px-1.5 py-0.5 rounded bg-gray-100 text-gray-600">
                            {ACTION_LABEL[v.action] || v.action}
                          </span>
                        </div>
                        <div className="flex items-center gap-2">
                          {canRestore && onRestore && (
                            <button
                              type="button"
                              disabled={restoring}
                              className="text-xs text-amber-700 hover:text-amber-900 disabled:opacity-50"
                              onClick={() => void handleRestore(v.version)}
                            >
                              恢复
                            </button>
                          )}
                          <button
                            type="button"
                            className="text-xs text-blue-600 hover:text-blue-800"
                            onClick={() => {
                              setLeftVer(v.version);
                              setRightVer(currentDraft ? "current" : v.version);
                              void (async () => {
                                setComparing(true);
                                setError(null);
                                try {
                                  if (currentDraft) {
                                    const left = await getArticleVersion(articleId, v.version);
                                    setCompareResult({
                                      left,
                                      right: syntheticCurrentDetail(articleId, currentDraft),
                                    });
                                  } else {
                                    const older = versions.find((x) => x.version < v.version);
                                    const result = await compareArticleVersions(
                                      articleId,
                                      older?.version ?? v.version,
                                      v.version,
                                    );
                                    setCompareResult(result);
                                  }
                                  setView("compare");
                                } catch (err) {
                                  setError(err instanceof Error ? err.message : "比对失败");
                                } finally {
                                  setComparing(false);
                                }
                              })();
                            }}
                          >
                            {currentDraft ? "与当前比对" : "查看"}
                          </button>
                        </div>
                      </div>
                      <div className="text-xs text-gray-500 mt-1">{formatTime(v.createdAt)}</div>
                      {(v.zhTitle || v.summary) && (
                        <div className="text-xs text-gray-600 mt-1 truncate">
                          {v.zhTitle || v.summary}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 overflow-y-auto p-4 space-y-6">
            {compareResult && (
              <>
                <div className="flex items-center gap-3 text-sm bg-gray-50 rounded-lg p-3 border border-gray-100">
                  <div className="flex-1">
                    <div className="text-xs text-gray-400">左</div>
                    <div className="font-medium">v{compareResult.left.version}</div>
                    <div className="text-xs text-gray-500">{formatTime(compareResult.left.createdAt)}</div>
                  </div>
                  <div className="text-gray-300">→</div>
                  <div className="flex-1 text-right">
                    <div className="text-xs text-gray-400">右</div>
                    <div className="font-medium">
                      {compareRightIsCurrent
                        ? "当前编辑"
                        : `v${compareResult.right.version}`}
                    </div>
                    <div className="text-xs text-gray-500">
                      {compareRightIsCurrent
                        ? "含未保存更改"
                        : formatTime(compareResult.right.createdAt)}
                    </div>
                  </div>
                </div>

                {canRestore && onRestore && !compareRightIsCurrent && (
                  <button
                    type="button"
                    disabled={restoring}
                    onClick={() => void handleRestore(compareResult.left.version)}
                    className="w-full px-3 py-1.5 text-sm border border-amber-300 text-amber-800 rounded-lg hover:bg-amber-50 disabled:opacity-50"
                  >
                    恢复左侧版本 v{compareResult.left.version} 到编辑器
                  </button>
                )}
                {canRestore && onRestore && compareRightIsCurrent && (
                  <button
                    type="button"
                    disabled={restoring}
                    onClick={() => void handleRestore(compareResult.left.version)}
                    className="w-full px-3 py-1.5 text-sm border border-amber-300 text-amber-800 rounded-lg hover:bg-amber-50 disabled:opacity-50"
                  >
                    用 v{compareResult.left.version} 覆盖当前编辑
                  </button>
                )}

                <FieldDiff
                  label="中文标题"
                  left={snapStr(leftSnap, "zhTitle")}
                  right={snapStr(rightSnap, "zhTitle")}
                />
                <FieldDiff
                  label="英文标题"
                  left={snapStr(leftSnap, "enTitle")}
                  right={snapStr(rightSnap, "enTitle")}
                />
                <FieldDiff
                  label="Slug"
                  left={snapStr(leftSnap, "slug")}
                  right={snapStr(rightSnap, "slug")}
                />
                <FieldDiff
                  label="状态"
                  left={snapStr(leftSnap, "status")}
                  right={snapStr(rightSnap, "status")}
                />
                <FieldDiff
                  label="中文正文"
                  left={snapStr(leftSnap, "zhBody")}
                  right={snapStr(rightSnap, "zhBody")}
                  asHtml
                />
                <FieldDiff
                  label="英文正文"
                  left={snapStr(leftSnap, "enBody")}
                  right={snapStr(rightSnap, "enBody")}
                  asHtml
                />
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
