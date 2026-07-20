import { useEffect, useState } from "react";
import type { Editor } from "@tiptap/react";
import { countWords, htmlToPlainText } from "../bilingualUtils";
import { WORD_STATS_DEBOUNCE_MS } from "../utils/constants";

const EMPTY = { chars: 0, words: 0 };

/**
 * Debounced word/char counts for ZH (+ optional EN).
 * Depends on scalar markdown strings, not the whole content object.
 */
export function useWordStats(opts: {
  editorMode: "richtext" | "markdown";
  markdownZh: string;
  markdownEn: string;
  zhBody: string;
  enBody: string;
  zhEditor: Editor | null;
  enEditor: Editor | null;
  /** When false, EN is reported as zeros without scanning the EN doc. */
  includeEn?: boolean;
  /** Recompute when these flip (typing / save). */
  tick: unknown;
}) {
  const {
    editorMode,
    markdownZh,
    markdownEn,
    zhBody,
    enBody,
    zhEditor,
    enEditor,
    includeEn = true,
    tick,
  } = opts;

  const [wordStats, setWordStats] = useState({
    zh: EMPTY,
    en: EMPTY,
  });

  useEffect(() => {
    const t = window.setTimeout(() => {
      const zhText =
        editorMode === "markdown"
          ? markdownZh
          : (zhEditor?.getText() || htmlToPlainText(zhBody));

      const enText = includeEn
        ? editorMode === "markdown"
          ? markdownEn
          : (enEditor?.getText() || htmlToPlainText(enBody))
        : "";

      setWordStats({
        zh: countWords(zhText),
        en: includeEn ? countWords(enText) : EMPTY,
      });
    }, WORD_STATS_DEBOUNCE_MS);
    return () => window.clearTimeout(t);
  }, [
    editorMode,
    markdownZh,
    markdownEn,
    zhBody,
    enBody,
    zhEditor,
    enEditor,
    includeEn,
    tick,
  ]);

  return wordStats;
}
