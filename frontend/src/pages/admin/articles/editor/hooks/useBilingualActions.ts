import { useCallback, useState, type Dispatch, type SetStateAction } from "react";
import type { Editor } from "@tiptap/react";
import { translateText } from "@/api/translation";
import { htmlToPlainText, plainTextToHtml } from "../bilingualUtils";
import { toast } from "../utils/toast";
import type { EditorMode } from "./useArticleEditors";

/**
 * Copy / AI-translate content between zh and en.
 */
export function useBilingualActions(opts: {
  editorMode: EditorMode;
  markdownContent: Record<string, string>;
  setMarkdownContent: Dispatch<SetStateAction<Record<string, string>>>;
  zhEditor: Editor | null;
  enEditor: Editor | null;
  zhBody: string;
  enBody: string;
  setZhBody: (v: string) => void;
  setEnBody: (v: string) => void;
  zhTitle: string;
  enTitle: string;
  setZhTitle: (v: string) => void;
  setEnTitle: (v: string) => void;
  enabledLangs: string[];
  ensureEnEnabled: () => void;
  touch: () => void;
  setError: (e: string | null) => void;
  setSuccessMessage: (s: string) => void;
}) {
  const {
    editorMode, markdownContent, setMarkdownContent,
    zhEditor, enEditor, zhBody, enBody, setZhBody, setEnBody,
    zhTitle, enTitle, setZhTitle, setEnTitle,
    ensureEnEnabled, touch, setError, setSuccessMessage,
  } = opts;

  const [translateBusy, setTranslateBusy] = useState(false);

  const handleCopyToOtherLang = useCallback((from: "zh" | "en") => {
    const to = from === "zh" ? "en" : "zh";
    if (editorMode === "markdown") {
      setMarkdownContent((prev) => ({ ...prev, [to]: prev[from] ?? "" }));
    } else {
      const srcEd = from === "zh" ? zhEditor : enEditor;
      const dstEd = to === "zh" ? zhEditor : enEditor;
      const html = srcEd?.getHTML() || (from === "zh" ? zhBody : enBody) || "";
      dstEd?.commands.setContent(html, { emitUpdate: false });
      if (to === "zh") setZhBody(html);
      else setEnBody(html);
    }
    if (from === "zh" && zhTitle.trim()) setEnTitle(zhTitle);
    if (from === "en" && enTitle.trim()) setZhTitle(enTitle);
    if (to === "en") ensureEnEnabled();
    touch();
    toast(
      setSuccessMessage,
      from === "zh" ? "已复制中文到英文（未保存）" : "已复制英文到中文（未保存）",
      2500,
    );
  }, [
    editorMode, setMarkdownContent, zhEditor, enEditor, zhBody, enBody,
    setZhBody, setEnBody, zhTitle, enTitle, setZhTitle, setEnTitle,
    ensureEnEnabled, touch, setSuccessMessage,
  ]);

  const handleTranslateToOtherLang = useCallback(async (from: "zh" | "en") => {
    const to = from === "zh" ? "en" : "zh";
    setTranslateBusy(true);
    setError(null);
    try {
      const srcTitle = from === "zh" ? zhTitle : enTitle;
      let srcBodyPlain: string;
      if (editorMode === "markdown") {
        srcBodyPlain = markdownContent[from] ?? "";
      } else {
        const srcEd = from === "zh" ? zhEditor : enEditor;
        srcBodyPlain = htmlToPlainText(
          srcEd?.getHTML() || (from === "zh" ? zhBody : enBody) || "",
        );
      }
      if (!srcTitle.trim() && !srcBodyPlain.trim()) {
        setError("源语言内容为空，无法翻译");
        return;
      }
      if (srcTitle.trim()) {
        const tr = await translateText({ text: srcTitle, sourceLang: from, targetLang: to });
        if (to === "zh") setZhTitle(tr.translatedText);
        else setEnTitle(tr.translatedText);
      }
      if (srcBodyPlain.trim()) {
        const tr = await translateText({ text: srcBodyPlain, sourceLang: from, targetLang: to });
        if (editorMode === "markdown") {
          setMarkdownContent((prev) => ({ ...prev, [to]: tr.translatedText }));
        } else {
          const html = plainTextToHtml(tr.translatedText);
          const dstEd = to === "zh" ? zhEditor : enEditor;
          dstEd?.commands.setContent(html, { emitUpdate: false });
          if (to === "zh") setZhBody(html);
          else setEnBody(html);
        }
      }
      if (to === "en") ensureEnEnabled();
      touch();
      toast(
        setSuccessMessage,
        from === "zh" ? "已翻译到英文（未保存，请校对）" : "已翻译到中文（未保存，请校对）",
      );
    } catch (err: unknown) {
      const ax = err as { response?: { data?: { error?: { message?: string } | string } } };
      const msg = (ax?.response?.data?.error as { message?: string })?.message
        ?? (typeof ax?.response?.data?.error === "string" ? ax.response.data.error : undefined);
      setError(msg || (err instanceof Error ? err.message : "翻译失败（请检查 AI/翻译配置）"));
    } finally {
      setTranslateBusy(false);
    }
  }, [
    zhTitle, enTitle, editorMode, markdownContent, setMarkdownContent,
    zhEditor, enEditor, zhBody, enBody, setZhBody, setEnBody,
    setZhTitle, setEnTitle, ensureEnEnabled, touch, setError, setSuccessMessage,
  ]);

  return {
    translateBusy,
    handleCopyToOtherLang,
    handleTranslateToOtherLang,
  };
}
