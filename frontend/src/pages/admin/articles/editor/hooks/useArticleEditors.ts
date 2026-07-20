import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { Editor } from "@tiptap/react";
import { useModalState } from "@/components/admin/RichTextEditor";
import type { MarkdownSelectionApi } from "@/components/admin/editor/MarkdownToolbar";
import { markdownToHtml, htmlToMarkdown } from "@/lib/markdown";
import type { ArticleDraftSnapshot } from "../VersionHistoryPanel";
import { slugifyTitle } from "../utils/slugify";
import {
  buildModeSwitchConfirmMessage,
  detectModeSwitchLossFromBodies,
} from "../utils/modeSwitchLoss";
import {
  readPreferredEditorMode,
  writePreferredEditorMode,
} from "../utils/editorPrefs";

export type EditorMode = "richtext" | "markdown";
export type ViewLayout = "focus" | "split";

export type DraftSnapshotMeta = {
  zhTitle: string;
  enTitle: string;
  slug: string;
  articleStatus: string;
  coverImage: string;
  zhSeoTitle: string;
  enSeoTitle: string;
  zhMetaDescription: string;
  enMetaDescription: string;
  ogImage: string;
  author: string;
};

/**
 * TipTap instances are provided by LangEditorMount (ZH always, EN on demand).
 * resolveBodies falls back to React body state when an instance is unmounted.
 *
 * Return identity is stable when none of the surface values change (useMemo).
 */
export function useArticleEditors(opts: {
  zhBody: string;
  enBody: string;
  setZhBody: (v: string) => void;
  setEnBody: (v: string) => void;
  touch: () => void;
}) {
  const { zhBody, enBody, setZhBody, setEnBody, touch } = opts;

  // New articles start in the user's last-used mode; loaded articles reset to richtext
  const [editorMode, setEditorMode] = useState<EditorMode>(() => readPreferredEditorMode());
  const [markdownContent, setMarkdownContent] = useState<Record<string, string>>({
    zh: "",
    en: "",
  });
  const [markdownApi, setMarkdownApi] = useState<MarkdownSelectionApi | null>(null);
  const [viewLayout, setViewLayout] = useState<ViewLayout>("focus");
  const [enabledLangs, setEnabledLangs] = useState<string[]>(["zh"]);
  const [activeLangIdx, setActiveLangIdx] = useState(0);

  const [zhEditor, setZhEditor] = useState<Editor | null>(null);
  const [enEditor, setEnEditor] = useState<Editor | null>(null);

  const { modals: zhModals, state: zhModalState } = useModalState();
  const { modals: enModals, state: enModalState } = useModalState();

  // Refs for save-time body resolution without recreating callbacks every keystroke
  const editorModeRef = useRef(editorMode);
  editorModeRef.current = editorMode;
  const markdownRef = useRef(markdownContent);
  markdownRef.current = markdownContent;
  const zhEditorRef = useRef(zhEditor);
  zhEditorRef.current = zhEditor;
  const enEditorRef = useRef(enEditor);
  enEditorRef.current = enEditor;
  const zhBodyRef = useRef(zhBody);
  zhBodyRef.current = zhBody;
  const enBodyRef = useRef(enBody);
  enBodyRef.current = enBody;
  const enabledLangsRef = useRef(enabledLangs);
  enabledLangsRef.current = enabledLangs;

  const needEnEditor = enabledLangs.includes("en") || viewLayout === "split";

  const langEditors = useMemo(
    () => ({
      zh: { editor: zhEditor, modals: zhModals, state: zhModalState },
      en: { editor: enEditor, modals: enModals, state: enModalState },
    }),
    [zhEditor, enEditor, zhModals, enModals, zhModalState, enModalState],
  );

  const activeLang = enabledLangs[activeLangIdx] || "zh";
  const activeEntry = langEditors[activeLang as "zh" | "en"];

  const enEnabled = enabledLangs.includes("en");
  useEffect(() => {
    if (viewLayout === "split" && !enEnabled) {
      setEnabledLangs((prev) => (prev.includes("en") ? prev : [...prev, "en"]));
    }
  }, [viewLayout, enEnabled]);

  const getZhHtml = useCallback(
    () => zhEditorRef.current?.getHTML() || zhBodyRef.current || "",
    [],
  );
  const getEnHtml = useCallback(
    () => enEditorRef.current?.getHTML() || enBodyRef.current || "",
    [],
  );

  /** Stable across keystrokes — reads latest mode/content via refs. */
  const resolveBodies = useCallback(() => {
    if (editorModeRef.current === "markdown") {
      const md = markdownRef.current;
      const zhHtml = markdownToHtml(md.zh ?? "");
      const enHtml = markdownToHtml(md.en ?? "");
      zhEditorRef.current?.commands.setContent(zhHtml, { emitUpdate: false });
      enEditorRef.current?.commands.setContent(enHtml, { emitUpdate: false });
      const normalizedZh = zhEditorRef.current?.getHTML() || zhHtml;
      const normalizedEn = enEditorRef.current?.getHTML() || enHtml;
      setZhBody(normalizedZh);
      setEnBody(normalizedEn);
      return { zhBody: normalizedZh, enBody: normalizedEn };
    }
    return { zhBody: getZhHtml(), enBody: getEnHtml() };
  }, [getZhHtml, getEnHtml, setZhBody, setEnBody]);

  const applyBodiesToEditors = useCallback((nextZh: string, nextEn: string) => {
    setZhBody(nextZh);
    setEnBody(nextEn);
    zhEditorRef.current?.commands.setContent(nextZh || "", { emitUpdate: false });
    enEditorRef.current?.commands.setContent(nextEn || "", { emitUpdate: false });
    if (editorModeRef.current === "markdown") {
      setMarkdownContent({
        zh: htmlToMarkdown(nextZh || ""),
        en: htmlToMarkdown(nextEn || ""),
      });
    }
  }, [setZhBody, setEnBody]);

  const handleModeChange = useCallback((newMode: EditorMode) => {
    const current = editorModeRef.current;
    if (newMode === current) return;

    if (newMode === "markdown") {
      const zhHtml = getZhHtml();
      const enHtml = getEnHtml();
      const losses = detectModeSwitchLossFromBodies(zhHtml, enHtml);
      if (losses.length > 0 && !window.confirm(buildModeSwitchConfirmMessage(losses))) {
        return;
      }
      setMarkdownContent({
        zh: htmlToMarkdown(zhHtml),
        en: htmlToMarkdown(enHtml),
      });
    } else {
      const md = markdownRef.current;
      const zhFromMd = markdownToHtml(md.zh ?? "");
      const enFromMd = markdownToHtml(md.en ?? "");
      zhEditorRef.current?.commands.setContent(zhFromMd, { emitUpdate: false });
      enEditorRef.current?.commands.setContent(enFromMd, { emitUpdate: false });
      setZhBody(zhEditorRef.current?.getHTML() || zhFromMd);
      setEnBody(enEditorRef.current?.getHTML() || enFromMd);
      setMarkdownApi(null);
    }
    setEditorMode(newMode);
    writePreferredEditorMode(newMode);
    touch();
  }, [getZhHtml, getEnHtml, touch, setZhBody, setEnBody]);

  const buildDraftSnapshot = useCallback((meta: DraftSnapshotMeta): ArticleDraftSnapshot => {
    const bodies = resolveBodies();
    return {
      zhTitle: meta.zhTitle,
      enTitle: meta.enTitle,
      slug: meta.slug.trim() || slugifyTitle(meta.zhTitle, meta.enTitle),
      status: meta.articleStatus,
      zhBody: bodies.zhBody,
      enBody: bodies.enBody,
      coverImage: meta.coverImage,
      zhSeoTitle: meta.zhSeoTitle,
      enSeoTitle: meta.enSeoTitle,
      zhMetaDescription: meta.zhMetaDescription,
      enMetaDescription: meta.enMetaDescription,
      ogImage: meta.ogImage,
      author: meta.author,
    };
  }, [resolveBodies]);

  const addLang = useCallback((langKey: string) => {
    setEnabledLangs((prev) => {
      if (prev.includes(langKey)) return prev;
      const next = [...prev, langKey];
      setActiveLangIdx(next.length - 1);
      return next;
    });
  }, []);

  const removeLang = useCallback((langKey: string) => {
    if (langKey === "zh") return;
    setEnabledLangs((prev) => {
      const next = prev.filter((l) => l !== langKey);
      setActiveLangIdx((idx) => (idx >= next.length ? Math.max(0, next.length - 1) : idx));
      return next;
    });
  }, []);

  const selectLangKey = useCallback((lang: string) => {
    const idx = enabledLangsRef.current.indexOf(lang);
    if (idx >= 0) setActiveLangIdx(idx);
  }, []);

  const ensureEnEnabled = useCallback(() => {
    setEnabledLangs((prev) => (prev.includes("en") ? prev : [...prev, "en"]));
  }, []);

  const resetMarkdownOnLoad = useCallback((enableEn: boolean) => {
    setMarkdownContent({ zh: "", en: "" });
    setEditorMode("richtext");
    if (enableEn) setEnabledLangs(["zh", "en"]);
    else setEnabledLangs(["zh"]);
  }, []);

  const onZhEditorReady = useCallback((ed: Editor | null) => {
    setZhEditor(ed);
  }, []);

  const onEnEditorReady = useCallback((ed: Editor | null) => {
    setEnEditor(ed);
  }, []);

  const onZhFlushBody = useCallback((html: string) => {
    setZhBody(html);
  }, [setZhBody]);

  const onEnFlushBody = useCallback((html: string) => {
    setEnBody(html);
  }, [setEnBody]);

  const zhEditable = viewLayout === "split" || activeLang === "zh";
  const enEditable = viewLayout === "split" || activeLang === "en";

  return useMemo(
    () => ({
      editorMode,
      setEditorMode,
      markdownContent,
      setMarkdownContent,
      markdownApi,
      setMarkdownApi,
      viewLayout,
      setViewLayout,
      enabledLangs,
      setEnabledLangs,
      activeLangIdx,
      setActiveLangIdx,
      activeLang,
      activeEntry,
      langEditors,
      zhEditor,
      enEditor,
      needZhEditor: true as const,
      needEnEditor,
      onZhEditorReady,
      onEnEditorReady,
      onZhFlushBody,
      onEnFlushBody,
      zhEditable,
      enEditable,
      resolveBodies,
      applyBodiesToEditors,
      handleModeChange,
      buildDraftSnapshot,
      addLang,
      removeLang,
      selectLangKey,
      ensureEnEnabled,
      resetMarkdownOnLoad,
    }),
    [
      editorMode,
      markdownContent,
      markdownApi,
      viewLayout,
      enabledLangs,
      activeLangIdx,
      activeLang,
      activeEntry,
      langEditors,
      zhEditor,
      enEditor,
      needEnEditor,
      onZhEditorReady,
      onEnEditorReady,
      onZhFlushBody,
      onEnFlushBody,
      zhEditable,
      enEditable,
      resolveBodies,
      applyBodiesToEditors,
      handleModeChange,
      buildDraftSnapshot,
      addLang,
      removeLang,
      selectLangKey,
      ensureEnEnabled,
      resetMarkdownOnLoad,
    ],
  );
}

export type ArticleEditorsApi = ReturnType<typeof useArticleEditors>;
