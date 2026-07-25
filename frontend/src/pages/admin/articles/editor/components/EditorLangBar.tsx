import type { RefObject } from "react";
import { EditorModeSwitcher } from "@inkless/editor";
import { ALL_LANGS, type LangKey } from "../utils/constants";
import {
  prefetchMarkdownEditor,
  prefetchRichTextEditor,
} from "./lazyEditorSurfaces";

type WordStat = { chars: number; words: number };

export function EditorLangBar({
  enabledLangs,
  activeLangIdx,
  viewLayout,
  wordStats,
  editorMode,
  translateBusy,
  showLangMenu,
  langMenuRef,
  onSelectLang,
  onRemoveLang,
  onAddLang,
  onToggleLangMenu,
  onToggleSplit,
  onCopyZhToEn,
  onTranslateZhToEn,
  onModeChange,
}: {
  enabledLangs: string[];
  activeLangIdx: number;
  viewLayout: "focus" | "split";
  wordStats: Record<"zh" | "en", WordStat>;
  editorMode: "richtext" | "markdown";
  translateBusy: boolean;
  showLangMenu: boolean;
  langMenuRef: RefObject<HTMLDivElement | null>;
  onSelectLang: (idx: number) => void;
  onRemoveLang: (key: string) => void;
  onAddLang: (key: string) => void;
  onToggleLangMenu: () => void;
  onToggleSplit: () => void;
  onCopyZhToEn: () => void;
  onTranslateZhToEn: () => void;
  onModeChange: (mode: "richtext" | "markdown") => void;
}) {
  return (
    <div className="flex items-center gap-1 px-4 pt-1.5 pb-0 border-t border-slate-100 bg-slate-50/80">
      {enabledLangs.map((lang, idx) => {
        const info = ALL_LANGS.find((l) => l.key === lang);
        return (
          <button
            key={lang}
            type="button"
            onClick={() => onSelectLang(idx)}
            className={`group relative px-4 py-1.5 text-sm rounded-t-lg border border-b-0 transition-colors ${
              idx === activeLangIdx
                ? "bg-white border-slate-200 text-slate-900 font-medium shadow-sm"
                : "bg-transparent border-transparent text-slate-500 hover:text-slate-700 hover:bg-slate-100/80"
            }`}
          >
            <span>{info?.label || lang}</span>
            <span className="ml-1.5 text-[10px] text-slate-400 font-normal tabular-nums">
              {(wordStats[lang as LangKey]?.words ?? 0).toLocaleString()} 词
            </span>
            {lang !== "zh" && viewLayout !== "split" && (
              <span
                onClick={(e) => {
                  e.stopPropagation();
                  onRemoveLang(lang);
                }}
                className="ml-1.5 text-slate-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity"
              >
                &times;
              </span>
            )}
          </button>
        );
      })}

      {enabledLangs.length < ALL_LANGS.length && viewLayout !== "split" && (
        <div className="relative" ref={langMenuRef}>
          <button
            type="button"
            onClick={onToggleLangMenu}
            className="px-2 py-1.5 text-sm text-slate-400 hover:text-slate-600 rounded-t-lg hover:bg-slate-100"
            title="添加语言"
          >
            + 语言
          </button>
          {showLangMenu && (
            <div className="absolute top-full left-0 mt-0.5 py-1 bg-white rounded-lg shadow-lg border border-slate-200 z-40 min-w-[100px]">
              {ALL_LANGS.filter((l) => !enabledLangs.includes(l.key)).map((l) => (
                <button
                  key={l.key}
                  type="button"
                  onClick={() => onAddLang(l.key)}
                  className="w-full text-left px-3 py-1.5 text-sm hover:bg-slate-100 text-slate-700"
                >
                  {l.label}
                </button>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="ml-auto flex items-center gap-1.5 pb-1 flex-wrap justify-end">
        <button
          type="button"
          onClick={onToggleSplit}
          title="中英并排编辑"
          className={`px-2 py-1 text-xs rounded-lg border transition-colors ${
            viewLayout === "split"
              ? "bg-blue-50 border-blue-300 text-blue-800"
              : "border-slate-200 text-slate-600 hover:bg-slate-100"
          }`}
        >
          并排
        </button>
        {enabledLangs.includes("en") && (
          <>
            <button
              type="button"
              disabled={translateBusy}
              onClick={onCopyZhToEn}
              className="px-2 py-1 text-xs rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-100 disabled:opacity-50"
              title="将中文标题与正文复制到英文"
            >
              中→英 复制
            </button>
            <button
              type="button"
              disabled={translateBusy}
              onClick={onTranslateZhToEn}
              className="px-2 py-1 text-xs rounded-lg border border-violet-200 text-violet-700 hover:bg-violet-50 disabled:opacity-50"
              title="使用翻译 API 将中文译为英文"
            >
              {translateBusy ? "翻译中…" : "中→英 翻译"}
            </button>
          </>
        )}
        <EditorModeSwitcher
          mode={editorMode}
          onModeChange={onModeChange}
          onPrefetch={(mode) => {
            if (mode === "markdown") prefetchMarkdownEditor();
            else prefetchRichTextEditor();
          }}
        />
      </div>
    </div>
  );
}
