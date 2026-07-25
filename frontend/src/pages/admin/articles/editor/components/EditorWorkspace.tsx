import { Suspense, useMemo } from "react";
import type { Editor } from "@tiptap/react";
import type { MarkdownSelectionApi } from "@inkless/editor";
import { createInklessUploadPort } from "@/components/admin/editor-host/createUploadPort";
import EditorSidebar from "../EditorSidebar";
import { EditorChunkFallback } from "./EditorChunkFallback";
import { LazyMarkdownMode, LazyRichTextSurface } from "./lazyEditorSurfaces";

type LangEntry = {
  editor: Editor | null;
};

type WordStat = { chars: number; words: number };

type TitleMap = Record<
  string,
  { title: string; setTitle: (v: string) => void; placeholder: string }
>;

export function EditorWorkspace({
  viewLayout,
  editorMode,
  enabledLangs,
  activeLang,
  activeLangIdx,
  langEditors,
  langTitleMap,
  wordStats,
  markdownContent,
  metadata,
  sidebarArticle,
  showOutline,
  outlineCompact,
  onSelectLangKey,
  onMarkdownChange,
  onMarkdownApiReady,
  onJumpMarkdownLine,
}: {
  viewLayout: "focus" | "split";
  editorMode: "richtext" | "markdown";
  enabledLangs: string[];
  activeLang: string;
  activeLangIdx: number;
  langEditors: Record<string, LangEntry>;
  langTitleMap: TitleMap;
  wordStats: Record<"zh" | "en", WordStat>;
  markdownContent: Record<string, string>;
  metadata: Record<string, unknown>;
  sidebarArticle: {
    slug: string;
    author: string;
    createdAt: string | null;
    publishedAt: string | null;
  } | null;
  showOutline?: boolean;
  outlineCompact?: boolean;
  onSelectLangKey: (lang: string) => void;
  onMarkdownChange: (lang: string, val: string) => void;
  onMarkdownApiReady: (api: MarkdownSelectionApi | null) => void;
  onJumpMarkdownLine?: (line: number) => void;
}) {
  const activeEntry = langEditors[activeLang];
  const outlineVisible = showOutline !== false && viewLayout === "focus";
  const upload = useMemo(() => createInklessUploadPort(), []);

  return (
    <div className="flex-1 flex min-h-0">
      <div className="flex-1 flex flex-col min-h-0 min-w-0 bg-white">
        <div
          className={`flex-1 min-h-0 ${
            editorMode === "markdown" || viewLayout === "split" ? "overflow-hidden" : "overflow-y-auto"
          }`}
        >
          {viewLayout === "split" ? (
            <div className="h-full min-h-0 grid grid-cols-1 md:grid-cols-2 gap-0 divide-x divide-gray-200">
              {(["zh", "en"] as const).map((lang) => {
                const entry = langEditors[lang];
                const isActiveCol = activeLang === lang;
                return (
                  <div
                    key={lang}
                    className={`flex flex-col min-h-0 min-w-0 ${isActiveCol ? "bg-white" : "bg-slate-50/40"}`}
                    onMouseDown={() => onSelectLangKey(lang)}
                  >
                    <div className="flex-shrink-0 px-3 py-2 border-b border-slate-100 space-y-1.5">
                      <div className="flex items-center justify-between gap-2">
                        <span
                          className={`text-xs font-semibold ${isActiveCol ? "text-blue-700" : "text-slate-500"}`}
                        >
                          {lang === "zh" ? "中文" : "English"}
                          {isActiveCol && (
                            <span className="ml-1.5 text-[10px] font-normal text-blue-500">编辑中</span>
                          )}
                        </span>
                        <span className="text-[10px] text-slate-400 tabular-nums">
                          {wordStats[lang].words.toLocaleString()} 词 ·{" "}
                          {wordStats[lang].chars.toLocaleString()} 字
                        </span>
                      </div>
                      <input
                        type="text"
                        value={langTitleMap[lang]?.title || ""}
                        onChange={(e) => langTitleMap[lang]?.setTitle(e.target.value)}
                        onFocus={() => onSelectLangKey(lang)}
                        className="w-full px-2 py-1 text-sm font-medium border border-slate-200 rounded-lg focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
                        placeholder={langTitleMap[lang]?.placeholder || "标题"}
                      />
                    </div>
                    <div className="flex-1 min-h-0">
                      {editorMode === "markdown" ? (
                        <div className="h-full min-h-0 p-2">
                          <Suspense fallback={<EditorChunkFallback label="加载 Markdown…" />}>
                            <LazyMarkdownMode
                              key={`split-${lang}`}
                              contentKey={lang}
                              label={lang === "zh" ? "Markdown · 中文" : "Markdown · EN"}
                              showPreview={false}
                              value={markdownContent[lang] ?? ""}
                              onChange={(val) => onMarkdownChange(lang, val)}
                              onApiReady={lang === activeLang ? onMarkdownApiReady : undefined}
                              upload={upload}
                            />
                          </Suspense>
                        </div>
                      ) : entry?.editor ? (
                        <Suspense fallback={<EditorChunkFallback label="加载富文本…" />}>
                          <LazyRichTextSurface
                            editor={entry.editor}
                            showMenus={isActiveCol}
                            metadata={metadata}
                          />
                        </Suspense>
                      ) : (
                        <EditorChunkFallback label="初始化富文本…" />
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ) : editorMode === "markdown" ? (
            <div className="h-full min-h-0 p-3">
              <Suspense fallback={<EditorChunkFallback label="加载 Markdown…" />}>
                <LazyMarkdownMode
                  key={activeLang}
                  contentKey={activeLang}
                  value={markdownContent[activeLang] ?? ""}
                  onChange={(val) => onMarkdownChange(activeLang, val)}
                  onApiReady={onMarkdownApiReady}
                  upload={upload}
                />
              </Suspense>
            </div>
          ) : (
            enabledLangs.map((lang, idx) => {
              const entry = langEditors[lang];
              if (!entry?.editor) {
                return idx === activeLangIdx ? (
                  <EditorChunkFallback key={lang} label="初始化富文本…" />
                ) : null;
              }
              return (
                <div key={lang} className={idx === activeLangIdx ? "h-full" : "hidden"}>
                  <Suspense fallback={<EditorChunkFallback label="加载富文本…" />}>
                    <LazyRichTextSurface
                      editor={entry.editor}
                      showMenus
                      metadata={metadata}
                    />
                  </Suspense>
                </div>
              );
            })
          )}
        </div>
      </div>

      {outlineVisible && (
        <EditorSidebar
          editorMode={editorMode}
          editor={activeEntry?.editor ?? null}
          markdownSource={markdownContent[activeLang] ?? ""}
          onJumpMarkdownLine={onJumpMarkdownLine}
          article={sidebarArticle}
          compact={outlineCompact}
        />
      )}
    </div>
  );
}
