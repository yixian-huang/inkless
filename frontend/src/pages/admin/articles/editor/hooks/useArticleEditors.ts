import { useCallback, useEffect, useMemo, useState } from "react";
import { useEditor, type Editor } from "@tiptap/react";
import {
  getEditorExtensions,
  useModalState,
} from "@/components/admin/RichTextEditor";
import type { MarkdownSelectionApi } from "@/components/admin/editor/MarkdownToolbar";
import { markdownToHtml, htmlToMarkdown } from "@/lib/markdown";
import { MODE_SWITCH_MESSAGE } from "../saveStatusUtils";
import { hasMeaningfulHtml } from "../utils/constants";
import type { ArticleDraftSnapshot } from "../VersionHistoryPanel";
import { slugifyTitle } from "../utils/slugify";

export type EditorMode = "richtext" | "markdown";
export type ViewLayout = "focus" | "split";

/**
 * Dual TipTap instances + markdown buffers, body resolve, and mode switching.
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
  const [markdownContent, setMarkdownContent] = useState<Record<string, string>>({ zh: "", en: "" });
  const [markdownApi, setMarkdownApi] = useState<MarkdownSelectionApi | null>(null);
  const [viewLayout, setViewLayout] = useState<ViewLayout>("focus");
  const [enabledLangs, setEnabledLangs] = useState<string[]>(["zh"]);
  const [activeLangIdx, setActiveLangIdx] = useState(0);

  const zhExtensions = useMemo(() => getEditorExtensions(), []);
  const enExtensions = useMemo(() => getEditorExtensions(), []);

  const zhEditor = useEditor({
    extensions: zhExtensions,
    content: zhBody,
    shouldRerenderOnTransaction: false,
    editorProps: { attributes: { class: "tiptap" } },
    onUpdate: () => touch(),
  });
  const enEditor = useEditor({
    extensions: enExtensions,
    content: enBody,
    shouldRerenderOnTransaction: false,
    editorProps: { attributes: { class: "tiptap" } },
    onUpdate: () => touch(),
  });

  const { modals: zhModals, state: zhModalState } = useModalState();
  const { modals: enModals, state: enModalState } = useModalState();

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
    zhEditor?.setEditable(viewLayout === "split" || activeLang === "zh");
    enEditor?.setEditable(viewLayout === "split" || activeLang === "en");
  }, [zhEditor, enEditor, activeLang, viewLayout]);

  useEffect(() => {
    if (zhEditor && zhBody && zhBody !== zhEditor.getHTML()) {
      zhEditor.commands.setContent(zhBody, { emitUpdate: false });
    }
  }, [zhBody, zhEditor]);

  useEffect(() => {
    if (enEditor && enBody && enBody !== enEditor.getHTML()) {
      enEditor.commands.setContent(enBody, { emitUpdate: false });
    }
  }, [enBody, enEditor]);

  useEffect(() => {
    if (viewLayout === "split" && !enabledLangs.includes("en")) {
      setEnabledLangs((prev) => (prev.includes("en") ? prev : [...prev, "en"]));
    }
  }, [viewLayout, enabledLangs]);

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
      zhBody: zhEditor?.getHTML() || zhBody || "",
      enBody: enEditor?.getHTML() || enBody || "",
    };
  }, [editorMode, markdownContent, zhEditor, enEditor, zhBody, enBody, setZhBody, setEnBody]);

  const applyBodiesToEditors = useCallback((nextZh: string, nextEn: string) => {
    setZhBody(nextZh);
    setEnBody(nextEn);
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
      : (zhEditor?.getHTML() || zhBody || "");
    const enHtml = editorMode === "markdown"
      ? markdownToHtml(markdownContent.en ?? "")
      : (enEditor?.getHTML() || enBody || "");
    if (
      (hasMeaningfulHtml(zhHtml) || hasMeaningfulHtml(enHtml) || isDirty)
      && !window.confirm(MODE_SWITCH_MESSAGE)
    ) {
      return;
    }
    if (newMode === "markdown") {
      setMarkdownContent({
        zh: htmlToMarkdown(zhEditor?.getHTML() || zhBody || ""),
        en: htmlToMarkdown(enEditor?.getHTML() || enBody || ""),
      });
    } else {
      for (const lang of ["zh", "en"] as const) {
        const ed: Editor | null = lang === "zh" ? zhEditor : enEditor;
        const html = markdownToHtml(markdownContent[lang] ?? "");
        ed?.commands.setContent(html, { emitUpdate: false });
        const normalized = ed?.getHTML() || html;
        if (lang === "zh") setZhBody(normalized);
        else setEnBody(normalized);
      }
      setMarkdownApi(null);
    }
    setEditorMode(newMode);
    touch();
  }, [
    editorMode, markdownContent, zhEditor, enEditor, zhBody, enBody,
    isDirty, touch, setZhBody, setEnBody,
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
      setActiveLangIdx((idx) => (idx >= next.length ? next.length - 1 : idx));
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
  }, []);

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
