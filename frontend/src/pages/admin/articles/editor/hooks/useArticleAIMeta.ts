import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  generateArticleMeta,
  type ArticleMetaField,
  type ArticleMetaMode,
  type ArticleMetaResponse,
} from "@/api/ai";
import { plainTextFromHtml } from "../utils/publishChecklist";
import {
  applyAIMetaToForm,
  defaultSelectedKeys,
  resolveApplyValues,
  type AIMetaApplyKey,
  type AIMetaFeedbackKind,
  type AIMetaFormSetters,
} from "../utils/applyAIMeta";
import {
  evaluateAIMetaQuality,
  hasWarnSeverity,
  mergeQualityIssues,
  type QualityIssue,
} from "../utils/aiMetaQuality";
import {
  installAIMetaStatsDebug,
  recordAIMetaEvent,
} from "../utils/aiMetaTelemetry";

const MIN_BODY_RUNES = 80;

function axiosErrorMessage(err: unknown, fallback: string): string {
  const ax = err as {
    response?: { data?: { error?: { message?: string; code?: string } | string } };
  };
  const e = ax?.response?.data?.error;
  if (typeof e === "string") return e;
  if (e?.message) {
    if (e.code === "AI_NOT_CONFIGURED") {
      return "AI 未配置。请在「AI 配置」中启用 OpenAI 兼容或 Anthropic 接口。";
    }
    return e.message;
  }
  if (err instanceof Error) return err.message;
  return fallback;
}

export type OpenAIMetaOpts = {
  mode?: ArticleMetaMode;
  fields?: ArticleMetaField[];
  sourceLang?: "zh" | "en";
};

export function useArticleAIMeta(opts: {
  getBodies: () => { zhBody: string; enBody: string };
  getTitles: () => { zhTitle: string; enTitle: string };
  getExistingMeta: () => {
    slug: string;
    zhSeoTitle: string;
    enSeoTitle: string;
    zhMetaDescription: string;
    enMetaDescription: string;
  };
  getExistingTagNames?: () => string[];
  slugLocked: boolean;
  setters: AIMetaFormSetters;
  ensureEnEnabled?: () => void;
  touch: () => void;
  setError: (e: string | null) => void;
  setSuccessMessage: (s: string) => void;
}) {
  const optsRef = useRef(opts);
  optsRef.current = opts;

  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [response, setResponse] = useState<ArticleMetaResponse | null>(null);
  const [titleIndex, setTitleIndex] = useState(0);
  const [mode, setMode] = useState<ArticleMetaMode>("fill_empty");
  const [sourceLang, setSourceLang] = useState<"zh" | "en">("zh");
  const [selected, setSelected] = useState<Set<AIMetaApplyKey>>(new Set());
  const [panelError, setPanelError] = useState<string | null>(null);
  const [bodyPlain, setBodyPlain] = useState("");

  useEffect(() => {
    installAIMetaStatsDebug();
  }, []);

  const close = useCallback(() => {
    if (open && response) {
      recordAIMetaEvent({ type: "dismiss", model: response.model });
    }
    setOpen(false);
    setPanelError(null);
  }, [open, response]);

  const openPreview = useCallback(async (openOpts?: OpenAIMetaOpts) => {
    const o = optsRef.current;
    setPanelError(null);
    o.setError(null);
    const bodies = o.getBodies();
    const titles = o.getTitles();
    const existing = o.getExistingMeta();
    const lang = openOpts?.sourceLang ?? sourceLang;
    const nextMode = openOpts?.mode ?? mode;
    const body = lang === "en" ? bodies.enBody : bodies.zhBody;
    const plain = plainTextFromHtml(body);
    if (plain.length < MIN_BODY_RUNES) {
      o.setError(`正文过短（约需 ${MIN_BODY_RUNES} 字以上），请多写几段后再生成元数据。`);
      return;
    }

    setMode(nextMode);
    setSourceLang(lang);
    setBodyPlain(plain);
    setBusy(true);
    setOpen(true);
    setResponse(null);
    setTitleIndex(0);
    recordAIMetaEvent({ type: "open", mode: nextMode, sourceLang: lang });

    try {
      const resp = await generateArticleMeta({
        sourceLang: lang,
        zhTitle: titles.zhTitle,
        enTitle: titles.enTitle,
        zhBody: bodies.zhBody,
        enBody: bodies.enBody,
        existing: {
          slug: existing.slug,
          zhSeoTitle: existing.zhSeoTitle,
          enSeoTitle: existing.enSeoTitle,
          zhMetaDescription: existing.zhMetaDescription,
          enMetaDescription: existing.enMetaDescription,
          zhTitle: titles.zhTitle,
          enTitle: titles.enTitle,
        },
        fields: openOpts?.fields ?? ["titles", "slug", "seo", "meta"],
        mode: nextMode,
        titleCount: 3,
        existingTags: o.getExistingTagNames?.() ?? [],
        slugLocked: o.slugLocked,
      });
      setResponse(resp);
      const values = resolveApplyValues(resp, 0);
      setSelected(defaultSelectedKeys(values, o.slugLocked));
      recordAIMetaEvent({
        type: "generate_ok",
        model: resp.model,
        mode: nextMode,
        sourceLang: lang,
        warningCodes: (resp.warnings || []).map((w) => w.code),
        warnCount: (resp.warnings || []).filter((w) => w.severity !== "info").length,
      });
    } catch (err) {
      const msg = axiosErrorMessage(err, "生成元数据失败");
      setPanelError(msg);
      o.setError(msg);
      recordAIMetaEvent({ type: "generate_err", mode: nextMode, sourceLang: lang });
    } finally {
      setBusy(false);
    }
  }, [mode, sourceLang]);

  const cycleTitle = useCallback((delta: 1 | -1) => {
    setTitleIndex((i) => {
      const n = Math.max(
        response?.candidates?.zhTitles?.length || 0,
        response?.candidates?.enTitles?.length || 0,
        1,
      );
      return (i + delta + n) % n;
    });
  }, [response]);

  const toggleKey = useCallback((key: AIMetaApplyKey) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }, []);

  const values = useMemo(
    () => (response ? resolveApplyValues(response, titleIndex) : {}),
    [response, titleIndex],
  );

  const qualityIssues: QualityIssue[] = useMemo(() => {
    if (!response || busy) return [];
    const client = evaluateAIMetaQuality({
      values,
      bodyPlain,
      selected,
    });
    return mergeQualityIssues(response.warnings, client);
  }, [response, values, bodyPlain, selected, busy]);

  const apply = useCallback(() => {
    if (!response) return;
    const o = optsRef.current;
    const nextValues = resolveApplyValues(response, titleIndex);
    const n = applyAIMetaToForm(selected, nextValues, o.setters);
    if (selected.has("enTitle") || selected.has("enSeoTitle") || selected.has("enMetaDescription")) {
      o.ensureEnEnabled?.();
    }
    o.touch();
    const warnN = qualityIssues.filter((i) => i.severity === "warn").length;
    o.setSuccessMessage(
      warnN > 0
        ? `已应用 ${n} 项元数据（含 ${warnN} 条质检提醒，未保存）`
        : `已应用 ${n} 项元数据（未保存）`,
    );
    recordAIMetaEvent({
      type: "apply",
      model: response.model,
      applied: n,
      warnCount: warnN,
      warningCodes: qualityIssues.map((i) => i.code),
      mode,
      sourceLang,
    });
    setOpen(false);
    setPanelError(null);
  }, [response, titleIndex, selected, qualityIssues, mode, sourceLang]);

  const feedback = useCallback(
    (kind: AIMetaFeedbackKind) => {
      recordAIMetaEvent({
        type: "feedback",
        feedback: kind,
        model: response?.model,
      });
    },
    [response],
  );

  const titleCount = Math.max(
    response?.candidates?.zhTitles?.length || 0,
    response?.candidates?.enTitles?.length || 0,
    0,
  );

  return {
    open,
    busy,
    response,
    titleIndex,
    titleCount,
    mode,
    setMode,
    sourceLang,
    setSourceLang,
    selected,
    values,
    panelError,
    qualityIssues,
    hasQualityWarn: hasWarnSeverity(qualityIssues),
    slugLocked: opts.slugLocked,
    openPreview,
    close,
    cycleTitle,
    toggleKey,
    apply,
    feedback,
  };
}
