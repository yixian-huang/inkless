import { useCallback, useEffect, useRef, useState } from "react";
import type { NavigateFunction } from "react-router-dom";
import type { Article } from "@/api/articles";
import {
  createArticle,
  getAdminArticle,
  isArticleVersionConflict,
  updateArticle,
} from "@/api/articles";
import { resolveSaveStatus } from "../saveStatusUtils";
import { slugifyTitle } from "../utils/slugify";
import { AUTOSAVE_DEBOUNCE_MS } from "../utils/constants";
import { toast } from "../utils/toast";
import type { useDirtyState } from "./useDirtyState";
import type { ArticleStatus } from "./useArticleFormState";

type DirtyApi = ReturnType<typeof useDirtyState>;

type BuildPayload = (
  status: "draft" | "published",
  publishedAt?: string,
) => Record<string, unknown>;

/**
 * Load / save / autosave with optimistic concurrency (baseUpdatedAt).
 */
export function useArticlePersistence(opts: {
  id: string | undefined;
  isEditing: boolean;
  zhTitle: string;
  slug: string;
  setSlug: (v: string) => void;
  articleStatus: ArticleStatus;
  setArticleStatus: (s: ArticleStatus) => void;
  buildPayload: BuildPayload;
  hydrateFromArticle: (a: Article) => void;
  onLoaded?: (a: Article) => void;
  navigate: NavigateFunction;
  dirty: DirtyApi;
}) {
  const {
    id,
    isEditing,
    zhTitle,
    slug,
    setSlug,
    articleStatus,
    setArticleStatus,
    buildPayload,
    hydrateFromArticle,
    onLoaded,
    navigate,
    dirty,
  } = opts;

  const [loading, setLoading] = useState(isEditing);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState("");
  const [baseUpdatedAt, setBaseUpdatedAt] = useState<string | null>(null);
  const [conflict, setConflict] = useState<{ serverUpdatedAt?: string } | null>(null);
  const loadedIdRef = useRef<string | null>(null);

  const {
    isDirty,
    savingRef,
    pauseReady,
    resumeReady,
    markHydrated,
    markSaving,
    markClean,
    markError,
    readyRef,
  } = dirty;

  const loadArticle = useCallback(async () => {
    if (!id || loadedIdRef.current === id) return;
    pauseReady();
    setLoading(true);
    setError(null);
    try {
      const article = await getAdminArticle(Number(id));
      hydrateFromArticle(article);
      onLoaded?.(article);
      loadedIdRef.current = id;
      setBaseUpdatedAt(article.updatedAt || null);
      setConflict(null);
      markHydrated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load article");
    } finally {
      setLoading(false);
      resumeReady();
    }
  }, [id, pauseReady, resumeReady, markHydrated, hydrateFromArticle, onLoaded]);

  useEffect(() => {
    if (!id) {
      loadedIdRef.current = null;
      setLoading(false);
      readyRef.current = true;
      return;
    }
    if (loadedIdRef.current !== id) void loadArticle();
  }, [id, loadArticle, readyRef]);

  const handleSave = useCallback(async (
    intent: "draft" | "publish" | "autosave" = "draft",
    opts?: { force?: boolean },
  ) => {
    if (savingRef.current) return;
    if (!zhTitle.trim()) {
      if (intent !== "autosave") setError("请填写中文标题");
      return;
    }
    const finalSlug = slug.trim() || slugifyTitle(zhTitle);
    if (!slug.trim()) setSlug(finalSlug);

    const status = resolveSaveStatus(intent, articleStatus);
    const silent = intent === "autosave";
    const force = !!opts?.force;

    savingRef.current = true;
    setSaving(true);
    markSaving();
    if (!silent) {
      setError(null);
      setSuccessMessage("");
    }
    try {
      const payload = buildPayload(status);
      const articleId = id ? Number(id) : loadedIdRef.current ? Number(loadedIdRef.current) : null;
      if (articleId) {
        const updated = await updateArticle(articleId, payload as Partial<Article>, {
          baseUpdatedAt: force ? null : baseUpdatedAt,
          force,
        });
        if (updated?.updatedAt) setBaseUpdatedAt(updated.updatedAt);
        setConflict(null);
        setArticleStatus(
          status === "published"
            ? "published"
            : articleStatus === "scheduled"
              ? "scheduled"
              : status,
        );
        if (!silent) {
          toast(
            setSuccessMessage,
            intent === "publish" ? "已发布" : force ? "已强制覆盖保存" : "已保存",
          );
        }
      } else {
        const created = await createArticle(payload as Partial<Article>);
        setArticleStatus(status === "published" ? "published" : "draft");
        loadedIdRef.current = String(created.id);
        if (created.updatedAt) setBaseUpdatedAt(created.updatedAt);
        if (!silent) {
          toast(
            setSuccessMessage,
            intent === "publish" ? "已创建并发布" : "已保存",
          );
        }
        navigate(`/admin/articles/edit/${created.id}`, { replace: true });
      }
      markClean({ autosave: silent });
    } catch (err: unknown) {
      const conf = isArticleVersionConflict(err);
      if (conf.conflict) {
        markError();
        setConflict({ serverUpdatedAt: conf.currentUpdatedAt });
        if (!silent) setError(conf.message || "保存冲突：文章已被他人修改");
        return;
      }
      markError();
      const ax = err as { response?: { data?: { error?: { message?: string } } } };
      const msg = ax?.response?.data?.error?.message;
      setError(
        msg
        || (err instanceof Error ? err.message : silent ? "自动保存失败，请手动保存" : "保存失败"),
      );
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  }, [
    zhTitle, slug, setSlug, articleStatus, setArticleStatus, buildPayload,
    id, navigate, baseUpdatedAt, savingRef, markSaving, markClean, markError,
  ]);

  // Debounced autosave
  useEffect(() => {
    if (!isDirty || saving || loading || !zhTitle.trim()) return;
    const t = window.setTimeout(() => void handleSave("autosave"), AUTOSAVE_DEBOUNCE_MS);
    return () => window.clearTimeout(t);
  }, [isDirty, saving, loading, zhTitle, handleSave]);

  const forceReload = useCallback(() => {
    setConflict(null);
    loadedIdRef.current = null;
    pauseReady();
    void loadArticle();
  }, [pauseReady, loadArticle]);

  return {
    loading,
    saving,
    error,
    setError,
    successMessage,
    setSuccessMessage,
    baseUpdatedAt,
    conflict,
    setConflict,
    loadedIdRef,
    loadArticle,
    handleSave,
    forceReload,
  };
}
