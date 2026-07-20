import { useCallback, useEffect, useMemo, useState } from "react";
import type { Editor } from "@tiptap/react";
import { useModalState } from "@/components/admin/RichTextEditor";
import type { MarkdownSelectionApi } from "@/components/admin/editor/MarkdownToolbar";
import { markdownToHtml, htmlToMarkdown } from "@/lib/markdown";
import { MODE_SWITCH_MESSAGE } from "../saveStatusUtils";
import { hasMeaningfulHtml } from "../utils/constants";
import type { ArticleDraftSnapshot } from "../VersionHistoryPanel";
import { slugifyTitle } from "../utils/slugify";

export type EditorMode = "richtext" | "markdown";
export type ViewLayout = "focus" | "split";

/**
 * TipTap instances are provided by LangEditorMount (ZH always, EN on demand).
 * resolveBodies falls back to React body state when an instance is unmounted.
 */
export function useArticleEditors(opts: {
  zhBody: string;
  enBody: string;
  setZhBody: (v: string) => void;
  setEnBody: (v: string) => void;
  isDirty: boolean;
  touch: () => void;
}) {
  const { zhBody, enBody, setZhBody, setEnBody, isDirty, touch } = opts;

  const [editorMode, setEditorMode] = useState<EditorMode>("richtext");
  const [markdownContent, setMarkdownContent] = useState<Record<string, string>>({
    zh: "",
    en: "",
  });
  const [markdownApi, setMarkdownApi] = useState<MarkdownSelectionApi | null>(null);
  const [viewLayout, setViewLayout] = useState<ViewLayout>("focus");
  const [enabledLangs, setEnabledLangs] = useState<string[]>(["zh"]);
  const [activeLangIdx, setActiveLangIdx] = useState(0);

  // Both languages inject TipTap via LangEditorMount
  const [zhEditor, setZhEditor] = useState<Editor | null>(null);
  const [enEditor, setEnEditor] = useState<Editor | null>(null);

  const { modals: zhModals, state: zhModalState } = useModalState();
  const { modals: enModals, state: enModalState } = useModalState();

  /** ZH is primary and always mounted. */
  const needZhEditor = true;
  /** EN only when bilingual is active. */
  const needEnEditor =
    enabledLangs.includes("en") || viewLayout === "split";

  const langEditors = useMemo(
    () => ({
      zh: { editor: zhEditor, modals: zhModals, state: zhModalState },
      en: { editor: enEditor, modals: enModals, state: enModalState },
    }),
    [zhEditor, enEditor, zhModals, enModals, zhModalState, enModalState],
  );

  const activeLang = enabledLangs[activeLangIdx] || "zh";
  const activeEntry = langEditors[activeLang as "zh" | "en"];

  useEffect(() => {
    if (viewLayout === "split" && !enabledLangs.includes("en")) {
      setEnabledLangs((prev) => (prev.includes("en") ? prev : [...prev, "en"]));
    }
  }, [viewLayout, enabledLangs]);

  const getEnHtml = useCallback(() => enEditor?.getHTML() || enBody || "", [enEditor, enBody]);
  const getZhHtml = useCallback(() => zhEditor?.getHTML() || zhBody || "", [zhEditor, zhBody]);

  const resolveBodies = useCallback(() => {
    if (editorMode === "markdown") {
      const zhHtml = markdownToHtml(markdownContent.zh ?? "");
      const enHtml = markdownToHtml(markdownContent.en ?? "");
      zhEditor?.commands.setContent(zhHtml, { emitUpdate: false });
      enEditor?.commands.setContent(enHtml, { emitUpdate: false });
      const normalizedZh = zhEditor?.getHTML() || zhHtml;
      const normalizedEn = enEditor?.getHTML() || enHtml;
      setZhBody(normalizedZh);
      setEnBody(normalizedEn);
      return { zhBody: normalizedZh, enBody: normalizedEn };
    }
    return {
      zhBody: getZhHtml(),
      enBody: getEnHtml(),
    };
  }, [
    editorMode, markdownContent, zhEditor, enEditor,
    getZhHtml, getEnHtml, setZhBody, setEnBody,
  ]);

  const applyBodiesToEditors = useCallback((nextZh: string, nextEn: string) => {
    setZhBody(nextZh);
    setEnBody(nextEn);
    // Mounted instances pick up via html prop / setContent; unmounted seed on remount
    zhEditor?.commands.setContent(nextZh || "", { emitUpdate: false });
    enEditor?.commands.setContent(nextEn || "", { emitUpdate: false });
    if (editorMode === "markdown") {
      setMarkdownContent({
        zh: htmlToMarkdown(nextZh || ""),
        en: htmlToMarkdown(nextEn || ""),
      });
    }
  }, [zhEditor, enEditor, editorMode, setZhBody, setEnBody]);

  const handleModeChange = useCallback((newMode: EditorMode) => {
    if (newMode === editorMode) return;
    const zhHtml = editorMode === "markdown"
      ? markdownToHtml(markdownContent.zh ?? "")
      : getZhHtml();
    const enHtml = editorMode === "markdown"
      ? markdownToHtml(markdownContent.en ?? "")
      : getEnHtml();
    if (
      (hasMeaningfulHtml(zhHtml) || hasMeaningfulHtml(enHtml) || isDirty)
      && !window.confirm(MODE_SWITCH_MESSAGE)
    ) {
      return;
    }
    if (newMode === "markdown") {
      setMarkdownContent({
        zh: htmlToMarkdown(getZhHtml()),
        en: htmlToMarkdown(getEnHtml()),
      });
    } else {
      const zhFromMd = markdownToHtml(markdownContent.zh ?? "");
      const enFromMd = markdownToHtml(markdownContent.en ?? "");
      zhEditor?.commands.setContent(zhFromMd, { emitUpdate: false });
      enEditor?.commands.setContent(enFromMd, { emitUpdate: false });
      setZhBody(zhEditor?.getHTML() || zhFromMd);
      setEnBody(enEditor?.getHTML() || enFromMd);
      setMarkdownApi(null);
    }
    setEditorMode(newMode);
    touch();
  }, [
    editorMode, markdownContent, zhEditor, enEditor,
    getZhHtml, getEnHtml, isDirty, touch, setZhBody, setEnBody,
  ]);

  const buildDraftSnapshot = useCallback((meta: {
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
  }): ArticleDraftSnapshot => {
    const bodies = resolveBodies();
    return {
      zhTitle: meta.zhTitle,
      enTitle: meta.enTitle,
      slug: meta.slug.trim() || slugifyTitle(meta.zhTitle),
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
    const idx = enabledLangs.indexOf(lang);
    if (idx >= 0) setActiveLangIdx(idx);
  }, [enabledLangs]);

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

  return {
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
    needZhEditor,
    needEnEditor,
    onZhEditorReady,
    onEnEditorReady,
    onZhFlushBody,
    onEnFlushBody,
    zhEditable: viewLayout === "split" || activeLang === "zh",
    enEditable: viewLayout === "split" || activeLang === "en",
    resolveBodies,
    applyBodiesToEditors,
    handleModeChange,
    buildDraftSnapshot,
    addLang,
    removeLang,
    selectLangKey,
    ensureEnEnabled,
    resetMarkdownOnLoad,
  };
}
