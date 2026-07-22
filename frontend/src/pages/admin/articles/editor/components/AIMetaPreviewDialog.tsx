import type { ArticleMetaMode } from "@/api/ai";
import {
  AI_META_APPLY_KEYS,
  AI_META_LABELS,
  lengthHintForKey,
  type AIMetaApplyKey,
} from "../utils/applyAIMeta";
import {
  issuesForField,
  type QualityIssue,
} from "../utils/aiMetaQuality";

export function AIMetaPreviewDialog({
  open,
  busy,
  mode,
  onModeChange,
  sourceLang,
  onSourceLangChange,
  values,
  selected,
  onToggle,
  skipped,
  titleIndex,
  titleCount,
  onCycleTitle,
  slugLocked,
  panelError,
  qualityIssues = [],
  model,
  onClose,
  onApply,
  onRegenerate,
  onFeedback,
}: {
  open: boolean;
  busy: boolean;
  mode: ArticleMetaMode;
  onModeChange: (m: ArticleMetaMode) => void;
  sourceLang: "zh" | "en";
  onSourceLangChange: (l: "zh" | "en") => void;
  values: Partial<Record<AIMetaApplyKey, string>>;
  selected: Set<AIMetaApplyKey>;
  onToggle: (k: AIMetaApplyKey) => void;
  skipped?: string[];
  titleIndex: number;
  titleCount: number;
  onCycleTitle: (delta: 1 | -1) => void;
  slugLocked: boolean;
  panelError: string | null;
  qualityIssues?: QualityIssue[];
  model?: string;
  onClose: () => void;
  onApply: () => void;
  onRegenerate: () => void;
  onFeedback?: (kind: "useful" | "needs_edit" | "unusable") => void;
}) {
  if (!open) return null;

  const warnIssues = qualityIssues.filter((i) => i.severity === "warn");
  const infoIssues = qualityIssues.filter((i) => i.severity === "info");
  const canApply = !busy && !panelError && selected.size > 0 && Object.keys(values).length > 0;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="ai-meta-title"
        className="w-full max-w-lg rounded-xl bg-white shadow-xl border border-slate-200 overflow-hidden max-h-[90vh] flex flex-col"
      >
        <div className="px-4 py-3 border-b border-slate-100 flex-shrink-0">
          <h2 id="ai-meta-title" className="text-sm font-semibold text-slate-900">
            根据正文生成元数据
          </h2>
          <p className="text-xs text-slate-500 mt-0.5">
            预览后勾选再应用；默认不覆盖已有内容。应用前会做长度 / 语言 / 相关度质检。
          </p>
        </div>

        <div className="px-4 py-3 space-y-3 overflow-y-auto flex-1 min-h-0">
          <div className="flex flex-wrap gap-3 text-xs">
            <label className="flex items-center gap-1.5 text-slate-600">
              源语言
              <select
                value={sourceLang}
                disabled={busy}
                onChange={(e) => onSourceLangChange(e.target.value as "zh" | "en")}
                className="border border-slate-200 rounded-md px-1.5 py-1"
              >
                <option value="zh">中文</option>
                <option value="en">英文</option>
              </select>
            </label>
            <label className="flex items-center gap-1.5 text-slate-600">
              模式
              <select
                value={mode}
                disabled={busy}
                onChange={(e) => onModeChange(e.target.value as ArticleMetaMode)}
                className="border border-slate-200 rounded-md px-1.5 py-1"
              >
                <option value="fill_empty">仅填空</option>
                <option value="rewrite">全部重写</option>
              </select>
            </label>
            {titleCount > 1 && (
              <div className="flex items-center gap-1 ml-auto">
                <span className="text-slate-500">
                  标题候选 {titleIndex + 1}/{titleCount}
                </span>
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => onCycleTitle(-1)}
                  className="px-2 py-0.5 border border-slate-200 rounded hover:bg-slate-50"
                >
                  上一个
                </button>
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => onCycleTitle(1)}
                  className="px-2 py-0.5 border border-slate-200 rounded hover:bg-slate-50"
                >
                  换一个
                </button>
              </div>
            )}
          </div>

          {busy && (
            <div className="text-sm text-slate-500 py-8 text-center">正在根据正文生成…</div>
          )}

          {panelError && (
            <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
              {panelError}
              {panelError.includes("AI 未配置") && (
                <a
                  href="/admin/ai-settings"
                  className="block mt-1 text-xs underline text-red-700"
                >
                  打开 AI 配置
                </a>
              )}
            </div>
          )}

          {!busy && !panelError && warnIssues.length > 0 && (
            <div
              className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-950 space-y-1"
              role="status"
            >
              <div className="font-semibold">质检提醒（可仍应用，建议先核对）</div>
              <ul className="list-disc pl-4 space-y-0.5">
                {warnIssues.map((w) => (
                  <li key={`${w.code}-${w.field}-${w.message}`}>{w.message}</li>
                ))}
              </ul>
            </div>
          )}

          {!busy && !panelError && infoIssues.length > 0 && warnIssues.length === 0 && (
            <ul className="text-xs text-slate-600 bg-slate-50 border border-slate-100 rounded-lg px-3 py-2 space-y-0.5">
              {infoIssues.map((w) => (
                <li key={`${w.code}-${w.field}-${w.message}`}>{w.message}</li>
              ))}
            </ul>
          )}

          {!busy && !panelError && (
            <ul className="space-y-2">
              {AI_META_APPLY_KEYS.map((key) => {
                const value = values[key];
                const disabled = key === "slug" && slugLocked;
                const empty = !value?.trim();
                const hint = value ? lengthHintForKey(key, value) : null;
                const fieldIssues = issuesForField(qualityIssues, key);
                const fieldWarn = fieldIssues.some((i) => i.severity === "warn");
                return (
                  <li
                    key={key}
                    className={`rounded-lg border px-3 py-2 ${
                      empty || disabled
                        ? "border-slate-100 bg-slate-50/80 opacity-70"
                        : fieldWarn
                          ? "border-amber-200 bg-amber-50/40"
                          : "border-slate-200 bg-white"
                    }`}
                  >
                    <label className="flex items-start gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        className="mt-1"
                        checked={selected.has(key)}
                        disabled={disabled || empty || busy}
                        onChange={() => onToggle(key)}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center justify-between gap-2">
                          <span className="text-xs font-medium text-slate-700">
                            {AI_META_LABELS[key]}
                            {disabled && (
                              <span className="ml-1 text-amber-700 font-normal">（已发布锁定）</span>
                            )}
                            {fieldWarn && (
                              <span className="ml-1 text-amber-700 font-normal">· 需核对</span>
                            )}
                          </span>
                          {hint && (
                            <span
                              className={`text-[10px] tabular-nums ${
                                hint.warn ? "text-amber-600" : "text-slate-400"
                              }`}
                            >
                              {hint.length}
                              {hint.max != null ? `/${hint.max}` : ""}
                            </span>
                          )}
                        </div>
                        <div className="text-sm text-slate-800 mt-0.5 break-words whitespace-pre-wrap">
                          {empty ? (
                            <span className="text-slate-400 text-xs">
                              {disabled ? "不建议修改 slug" : "无建议（可能因仅填空而跳过）"}
                            </span>
                          ) : (
                            value
                          )}
                        </div>
                      </div>
                    </label>
                  </li>
                );
              })}
            </ul>
          )}

          {!!skipped?.length && !busy && (
            <p className="text-xs text-slate-500">
              已跳过：{skipped.join(", ")}
              {mode === "fill_empty" ? "（仅填空模式保留已有内容）" : ""}
            </p>
          )}

          {model && !busy && (
            <p className="text-[10px] text-slate-400">
              模型：{model}
              {qualityIssues.length > 0
                ? ` · 质检 ${qualityIssues.length} 条`
                : " · 质检通过"}
            </p>
          )}
        </div>

        <div className="flex items-center justify-between gap-2 px-4 py-3 border-t border-slate-100 bg-slate-50 flex-shrink-0">
          <div className="flex items-center gap-1">
            {onFeedback && (
              <>
                <button
                  type="button"
                  title="有用"
                  onClick={() => onFeedback("useful")}
                  className="px-2 py-1 text-[10px] text-slate-500 hover:text-slate-800"
                >
                  有用
                </button>
                <button
                  type="button"
                  title="需大改"
                  onClick={() => onFeedback("needs_edit")}
                  className="px-2 py-1 text-[10px] text-slate-500 hover:text-slate-800"
                >
                  需大改
                </button>
                <button
                  type="button"
                  title="不可用"
                  onClick={() => onFeedback("unusable")}
                  className="px-2 py-1 text-[10px] text-slate-500 hover:text-slate-800"
                >
                  不可用
                </button>
              </>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={onClose}
              disabled={busy}
              className="px-3 py-1.5 text-sm border border-slate-200 rounded-lg hover:bg-white disabled:opacity-50"
            >
              取消
            </button>
            <button
              type="button"
              onClick={onRegenerate}
              disabled={busy}
              className="px-3 py-1.5 text-sm border border-slate-200 rounded-lg hover:bg-white disabled:opacity-50"
            >
              重新生成
            </button>
            <button
              type="button"
              onClick={onApply}
              disabled={!canApply}
              className={`px-3 py-1.5 text-sm text-white rounded-lg disabled:opacity-50 ${
                warnIssues.length > 0
                  ? "bg-amber-600 hover:bg-amber-700"
                  : "bg-violet-600 hover:bg-violet-700"
              }`}
            >
              {warnIssues.length > 0 ? "仍要应用勾选项" : "应用勾选项"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
