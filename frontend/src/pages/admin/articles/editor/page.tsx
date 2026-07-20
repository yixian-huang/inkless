import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useEditor, EditorContent } from "@tiptap/react";
import type { Article } from "@/api/articles";
import {
  getAdminArticle,
  createArticle,
  updateArticle,
  getCategories,
  getTags,
} from "@/api/articles";
import type { Category, Tag } from "@/api/articles";
import {
  cancelScheduledPublication,
  createScheduledPublication,
  getResourceScheduledPublication,
  retryScheduledPublication,
  updateScheduledPublication,
  type ScheduledPublication,
} from "@/api/scheduledPublications";
import { ScheduledPublicationPanel } from "@/components/admin/ScheduledPublicationPanel";
import ImagePickerModal from "@/components/admin/ImagePickerModal";
import {
  getEditorExtensions,
  EditorToolbar,
  EditorModals,
  useModalState,
} from "@/components/admin/RichTextEditor";
import EditorBubbleMenu from "@/components/admin/editor/EditorBubbleMenu";
import TableBubbleMenu from "@/components/admin/editor/TableBubbleMenu";
import EditorFloatingMenu from "@/components/admin/editor/EditorFloatingMenu";
import EditorModeSwitcher from "@/components/admin/editor/EditorModeSwitcher";
import MarkdownMode from "@/components/admin/editor/MarkdownMode";
import MarkdownToolbar from "@/components/admin/editor/MarkdownToolbar";
import type { MarkdownSelectionApi } from "@/components/admin/editor/MarkdownToolbar";
import { markdownToHtml, htmlToMarkdown } from "@/lib/markdown";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { useAuth } from "@/contexts/AuthContext";
import EditorSidebar from "./EditorSidebar";
import ArticleForm from "./ArticleForm";
import { SeoFieldsPanel, AdvancedSettingsPanel, PopoverButton } from "./SeoFields";
import ArticleTypographyRoot from "@/components/blog/ArticleTypographyRoot";
import { ArticleVersionHistoryPanel, type ArticleDraftSnapshot } from "./VersionHistoryPanel";
import ArticlePreviewModal, { type ArticlePreviewData } from "./ArticlePreviewModal";
import { translateText } from "@/api/translation";
import { countWords, htmlToPlainText, plainTextToHtml } from "./bilingualUtils";
import { SaveStatusBadge } from "./saveStatus";
import {
  LEAVE_UNSAVED_MESSAGE,
  MODE_SWITCH_MESSAGE,
  resolveSaveStatus,
  type EditorSavePhase,
} from "./saveStatusUtils";

function slugifyTitle(title: string): string {
  return title
    .trim()
    .toLowerCase()
    .replace(/[\s_]+/g, "-")
    .replace(/[^\w\u4e00-\u9fff-]+/g, "")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 80) || `article-${Date.now()}`;
}

export default function ArticleEditorPage() {
  useDocumentTitle("编辑文章");
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEditing = !!id;
  const { hasPermission } = useAuth();
  const canPublish = hasPermission("articles:publish");

  // Form state
  const [zhTitle, setZhTitle] = useState("");
  const [enTitle, setEnTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [selectedCategoryIds, setSelectedCategoryIds] = useState<number[]>([]);
  const [selectedTagIds, setSelectedTagIds] = useState<number[]>([]);
  const [coverImage, setCoverImage] = useState("");
  const [zhBody, setZhBody] = useState("");
  const [enBody, setEnBody] = useState("");
  const [zhSeoTitle, setZhSeoTitle] = useState("");
  const [enSeoTitle, setEnSeoTitle] = useState("");
  const [zhMetaDescription, setZhMetaDescription] = useState("");
  const [enMetaDescription, setEnMetaDescription] = useState("");
  const [ogImage, setOgImage] = useState("");

  // Advanced settings
  const [author, setAuthor] = useState("");
  const [autoSummary, setAutoSummary] = useState(false);
  const [allowComments, setAllowComments] = useState(true);
  const [pinned, setPinned] = useState(false);
  const [visibility, setVisibility] = useState("public");
  const [metadata, setMetadata] = useState<Record<string, unknown>>({});

  // Article metadata (from API)
  const [articleCreatedAt, setArticleCreatedAt] = useState<string | null>(null);
  const [articlePublishedAt, setArticlePublishedAt] = useState<string | null>(null);
  const [articleStatus, setArticleStatus] = useState<"draft" | "published" | "scheduled">("draft");

  // UI state
  const [loading, setLoading] = useState(isEditing);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState("");
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [scheduledPublication, setScheduledPublication] = useState<ScheduledPublication | null>(null);
  const [scheduleLoading, setScheduleLoading] = useState(isEditing);
  const [scheduleBusy, setScheduleBusy] = useState(false);
  const [scheduleMessage, setScheduleMessage] = useState("");

  // P0: dirty / autosave / leave guards
  const [isDirty, setIsDirty] = useState(false);
  const [savePhase, setSavePhase] = useState<EditorSavePhase>("clean");
  const [lastSavedAt, setLastSavedAt] = useState<Date | null>(null);
  const [lastSaveWasAutosave, setLastSaveWasAutosave] = useState(false);
  /** Ignore touch() during initial hydrate / programmatic setContent. */
  const readyRef = useRef(!isEditing);
  const savingRef = useRef(false);

  // Editor mode (richtext / markdown)
  const [editorMode, setEditorMode] = useState<"richtext" | "markdown">("richtext");
  const [markdownContent, setMarkdownContent] = useState<Record<string, string>>({ zh: "", en: "" });
  const [markdownApi, setMarkdownApi] = useState<MarkdownSelectionApi | null>(null);
  // Track whether current article has been loaded (avoid re-fetch wiping edits)
  const loadedIdRef = useRef<string | null>(null);

  const touch = useCallback(() => {
    if (!readyRef.current) return;
    setIsDirty(true);
    setSavePhase((p) => (p === "saving" ? p : "dirty"));
  }, []);

  const track = useCallback(
    <T,>(setter: (v: T) => void) =>
      (v: T) => {
        setter(v);
        touch();
      },
    [touch],
  );

  // Panel states
  const [showBasicInfo, setShowBasicInfo] = useState(false);
  const [showSeo, setShowSeo] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showCoverPicker, setShowCoverPicker] = useState(false);
  const [showVersionHistory, setShowVersionHistory] = useState(false);
  /** Snapshot frozen when opening history panel (for compare-with-current). */
  const [versionDraftSnapshot, setVersionDraftSnapshot] = useState<ArticleDraftSnapshot | null>(null);
  const [showPreview, setShowPreview] = useState(false);
  const [previewData, setPreviewData] = useState<ArticlePreviewData | null>(null);
  /** focus = single active language; split = side-by-side bilingual */
  const [viewLayout, setViewLayout] = useState<"focus" | "split">("focus");
  const [translateBusy, setTranslateBusy] = useState(false);
  const [wordStats, setWordStats] = useState({
    zh: { chars: 0, words: 0 },
    en: { chars: 0, words: 0 },
  });

  // Language carousel
  const [enabledLangs, setEnabledLangs] = useState<string[]>(["zh"]);
  const [activeLangIdx, setActiveLangIdx] = useState(0);
  const [showLangMenu, setShowLangMenu] = useState(false);
  const langMenuRef = useRef<HTMLDivElement>(null);

  // All available languages
  const ALL_LANGS = [
    { key: "zh", label: "中文" },
    { key: "en", label: "English" },
  ];

  // Each editor gets its own extension instances to prevent shared state issues.
  // TipTap extensions bind `this.editor` during initialization — sharing instances
  // between editors causes cross-contamination of editor references.
  const zhExtensions = useMemo(() => getEditorExtensions(), []);
  const enExtensions = useMemo(() => getEditorExtensions(), []);

  // Editors for each language — no onUpdate setState to avoid expensive
  // getHTML() serialization + React re-render cascade on every keystroke.
  // Content is read directly from editors in buildPayload when saving.
  const markEditorDirty = useCallback(() => {
    if (!readyRef.current) return;
    setIsDirty(true);
    setSavePhase((p) => (p === "saving" ? p : "dirty"));
  }, []);

  const zhEditor = useEditor({
    extensions: zhExtensions,
    content: zhBody,
    editorProps: { attributes: { class: "tiptap" } },
    onUpdate: () => { markEditorDirty(); },
  });

  const enEditor = useEditor({
    extensions: enExtensions,
    content: enBody,
    editorProps: { attributes: { class: "tiptap" } },
    onUpdate: () => { markEditorDirty(); },
  });

  const { modals: zhModals, state: zhModalState } = useModalState();
  const { modals: enModals, state: enModalState } = useModalState();

  const langEditors: Record<string, { editor: typeof zhEditor; modals: typeof zhModals; state: typeof zhModalState }> = {
    zh: { editor: zhEditor, modals: zhModals, state: zhModalState },
    en: { editor: enEditor, modals: enModals, state: enModalState },
  };

  // Current active language
  const activeLang = enabledLangs[activeLangIdx] || "zh";
  const activeEntry = langEditors[activeLang];

  // Sync API-loaded content to editors (fires only when loadArticle sets state)
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

  // Close lang menu on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (langMenuRef.current && !langMenuRef.current.contains(e.target as Node)) setShowLangMenu(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  // Handle slash-command-media and editor-replace-media events
  useEffect(() => {
    const ms = activeEntry?.state;
    if (!ms) return;
    const onSlashMedia = (e: Event) => {
      const type = (e as CustomEvent).detail?.type;
      if (type === "image") ms.setShowImagePicker(true);
      else if (type === "video") ms.setShowVideoPicker(true);
      else if (type === "audio") ms.setShowAudioPicker(true);
      else if (type === "embed") ms.setShowEmbedUrl(true);
      else if (type === "gallery") ms.setShowGalleryPicker(true);
    };
    const onReplaceMedia = (e: Event) => {
      const type = (e as CustomEvent).detail?.type;
      if (type === "image") ms.setShowImagePicker(true);
      else if (type === "video") ms.setShowVideoPicker(true);
    };
    document.addEventListener("slash-command-media", onSlashMedia);
    document.addEventListener("editor-replace-media", onReplaceMedia);
    return () => {
      document.removeEventListener("slash-command-media", onSlashMedia);
      document.removeEventListener("editor-replace-media", onReplaceMedia);
    };
  }, [activeEntry?.state]);

  // Load categories and tags
  const loadMeta = useCallback(async () => {
    try {
      const [cats, tgs] = await Promise.all([getCategories(), getTags()]);
      setCategories(cats || []);
      setTags(tgs || []);
    } catch { /* non-critical */ }
  }, []);

  // Load article for editing (once per id — never wipe in-progress edits on remount)
  const loadArticle = useCallback(async () => {
    if (!id) return;
    if (loadedIdRef.current === id) return;
    readyRef.current = false;
    setLoading(true);
    setError(null);
    try {
      const article = await getAdminArticle(Number(id));
      setZhTitle(article.zhTitle || "");
      setEnTitle(article.enTitle || "");
      setSlug(article.slug || "");
      if (article.categoryIds && article.categoryIds.length > 0) {
        setSelectedCategoryIds(article.categoryIds);
      } else if (article.categories && article.categories.length > 0) {
        setSelectedCategoryIds(article.categories.map((c) => c.id));
      } else if (article.categoryId) {
        setSelectedCategoryIds([article.categoryId]);
      }
      setSelectedTagIds(article.tags?.map((t) => t.id) || []);
      setCoverImage(article.coverImage || "");
      setZhBody(article.zhBody || "");
      setEnBody(article.enBody || "");
      setZhSeoTitle(article.zhSeoTitle || "");
      setEnSeoTitle(article.enSeoTitle || "");
      setZhMetaDescription(article.zhMetaDescription || "");
      setEnMetaDescription(article.enMetaDescription || "");
      setOgImage(article.ogImage || "");
      setAuthor(article.author || "");
      setAutoSummary(article.autoSummary || false);
      setAllowComments(article.allowComments !== false);
      setPinned(article.pinned || false);
      setVisibility(article.visibility || "public");
      setMetadata(article.metadata || {});
      setArticleCreatedAt(article.createdAt || null);
      setArticlePublishedAt(article.publishedAt || null);
      setArticleStatus(article.status || "draft");
      setMarkdownContent({ zh: "", en: "" });
      setEditorMode("richtext");
      // Auto-enable English if it has content
      if (article.enBody || article.enTitle) {
        setEnabledLangs(["zh", "en"]);
      }
      loadedIdRef.current = id;
      setIsDirty(false);
      setSavePhase("clean");
      setLastSavedAt(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load article");
    } finally {
      setLoading(false);
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          readyRef.current = true;
        });
      });
    }
  }, [id]);

  const loadArticleSchedule = useCallback(async () => {
    if (!id) {
      setScheduleLoading(false);
      return;
    }
    setScheduleLoading(true);
    try {
      const schedule = await getResourceScheduledPublication("article", Number(id));
      setScheduledPublication(schedule);
    } catch {
      setScheduledPublication(null);
    } finally {
      setScheduleLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadMeta();
  }, [loadMeta]);

  useEffect(() => {
    if (!id) {
      loadedIdRef.current = null;
      setLoading(false);
      readyRef.current = true;
      return;
    }
    if (loadedIdRef.current !== id) {
      void loadArticle();
    }
    void loadArticleSchedule();
  }, [id, loadArticle, loadArticleSchedule]);

  /**
   * Resolve body HTML from the active editor mode.
   * Markdown path: MD → HTML → TipTap setContent → getHTML so mermaid/tables
   * land in the schema and survive later richtext reloads.
   */
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
  }, [editorMode, markdownContent, zhEditor, enEditor, zhBody, enBody]);

  const buildPayload = useCallback((status: "draft" | "published", publishedAt?: string): Record<string, unknown> => {
    const bodies = resolveBodies();
    const finalSlug = slug.trim() || slugifyTitle(zhTitle);
    const payload: Record<string, unknown> = {
      zhTitle, enTitle, slug: finalSlug, coverImage,
      zhBody: bodies.zhBody,
      enBody: bodies.enBody,
      zhSeoTitle, enSeoTitle, zhMetaDescription, enMetaDescription, ogImage,
      status, categoryIds: selectedCategoryIds, tagIds: selectedTagIds,
      author, autoSummary, allowComments, pinned, visibility, metadata,
    };
    if (status === "published") payload.publishedAt = publishedAt ?? new Date().toISOString();
    return payload;
  }, [
    resolveBodies, slug, zhTitle, enTitle, coverImage,
    zhSeoTitle, enSeoTitle, zhMetaDescription, enMetaDescription, ogImage,
    selectedCategoryIds, selectedTagIds, author, autoSummary, allowComments,
    pinned, visibility, metadata,
  ]);

  /**
   * intent:
   * - draft: manual 保存 (preserves published)
   * - publish: 发布
   * - autosave: quiet background save (preserves published)
   */
  const handleSave = useCallback(async (intent: "draft" | "publish" | "autosave" = "draft") => {
    if (savingRef.current) return;
    if (!zhTitle.trim()) {
      if (intent !== "autosave") setError("请填写中文标题");
      return;
    }
    const finalSlug = slug.trim() || slugifyTitle(zhTitle);
    if (!slug.trim()) setSlug(finalSlug);

    const status = resolveSaveStatus(intent, articleStatus);
    const silent = intent === "autosave";

    savingRef.current = true;
    setSaving(true);
    setSavePhase("saving");
    if (!silent) {
      setError(null);
      setSuccessMessage("");
    }
    try {
      const payload = buildPayload(status);
      const articleId = id ? Number(id) : loadedIdRef.current ? Number(loadedIdRef.current) : null;
      if (articleId) {
        await updateArticle(articleId, payload as Partial<Article>);
        setArticleStatus(
          status === "published"
            ? "published"
            : articleStatus === "scheduled"
              ? "scheduled"
              : status,
        );
        if (!silent) {
          setSuccessMessage(intent === "publish" ? "已发布" : "已保存");
        }
      } else {
        const created = await createArticle(payload as Partial<Article>);
        setArticleStatus(status === "published" ? "published" : "draft");
        loadedIdRef.current = String(created.id);
        if (!silent) {
          setSuccessMessage(intent === "publish" ? "已创建并发布" : "已保存");
        }
        navigate(`/admin/articles/edit/${created.id}`, { replace: true });
      }
      setIsDirty(false);
      setSavePhase("saved");
      setLastSavedAt(new Date());
      setLastSaveWasAutosave(silent);
      if (!silent) {
        window.setTimeout(() => setSuccessMessage(""), 3000);
      }
    } catch (err: any) {
      setSavePhase("error");
      const msg = err?.response?.data?.error?.message;
      if (!silent) {
        setError(msg || (err instanceof Error ? err.message : "保存失败"));
      } else {
        setError(msg || "自动保存失败，请手动保存");
      }
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  }, [zhTitle, slug, articleStatus, buildPayload, id, navigate]);

  // Debounced autosave while dirty (requires title)
  useEffect(() => {
    if (!isDirty || saving || loading) return;
    if (!zhTitle.trim()) return;
    const t = window.setTimeout(() => {
      void handleSave("autosave");
    }, 4000);
    return () => window.clearTimeout(t);
  }, [isDirty, saving, loading, zhTitle, handleSave]);

  const openPreview = useCallback(() => {
    const bodies = resolveBodies();
    const title = activeLang === "en" ? (enTitle || zhTitle) : (zhTitle || enTitle);
    const bodyHtml = activeLang === "en" ? (bodies.enBody || bodies.zhBody) : (bodies.zhBody || bodies.enBody);
    const langLabel = activeLang === "en" ? "English" : "中文";
    const statusLabel =
      articleStatus === "published" ? "已发布" : articleStatus === "scheduled" ? "定时" : "草稿";
    const finalSlug = slug.trim() || slugifyTitle(zhTitle);
    setPreviewData({
      title,
      bodyHtml,
      coverImage: coverImage || undefined,
      author: author || undefined,
      langLabel,
      statusLabel: isDirty ? `${statusLabel} · 含未保存` : statusLabel,
      publicPath:
        articleStatus === "published" && finalSlug ? `/blog/${finalSlug}` : null,
      metadata,
    });
    setShowPreview(true);
  }, [
    resolveBodies, activeLang, enTitle, zhTitle, articleStatus, slug,
    coverImage, author, isDirty, metadata,
  ]);

  // Global shortcuts: ⌘/Ctrl+S save, ⌘/Ctrl+Shift+S publish, ⌘/Ctrl+P preview
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;
      if (e.key === "s" || e.key === "S") {
        e.preventDefault();
        if (e.shiftKey) {
          if (canPublish) void handleSave("publish");
        } else {
          void handleSave("draft");
        }
        return;
      }
      if (e.key === "p" || e.key === "P") {
        // Avoid browser print when possible
        e.preventDefault();
        openPreview();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [handleSave, canPublish, openPreview]);

  const { confirmLeave } = useUnsavedChangesGuard(isDirty, LEAVE_UNSAVED_MESSAGE);

  const handleBack = useCallback(() => {
    if (!confirmLeave()) return;
    navigate("/admin/articles");
  }, [confirmLeave, navigate]);

  const buildCurrentDraftSnapshot = useCallback((): ArticleDraftSnapshot => {
    const bodies = resolveBodies();
    return {
      zhTitle,
      enTitle,
      slug: slug.trim() || slugifyTitle(zhTitle),
      status: articleStatus,
      zhBody: bodies.zhBody,
      enBody: bodies.enBody,
      coverImage,
      zhSeoTitle,
      enSeoTitle,
      zhMetaDescription,
      enMetaDescription,
      ogImage,
      author,
    };
  }, [
    resolveBodies, zhTitle, enTitle, slug, articleStatus, coverImage,
    zhSeoTitle, enSeoTitle, zhMetaDescription, enMetaDescription, ogImage, author,
  ]);

  const handleRestoreVersion = useCallback((snapshot: ArticleDraftSnapshot) => {
    readyRef.current = false;
    const nextZhTitle = typeof snapshot.zhTitle === "string" ? snapshot.zhTitle : "";
    const nextEnTitle = typeof snapshot.enTitle === "string" ? snapshot.enTitle : "";
    const nextSlug = typeof snapshot.slug === "string" ? snapshot.slug : slug;
    const nextZhBody = typeof snapshot.zhBody === "string" ? snapshot.zhBody : "";
    const nextEnBody = typeof snapshot.enBody === "string" ? snapshot.enBody : "";
    const nextCover = typeof snapshot.coverImage === "string" ? snapshot.coverImage : "";
    const nextAuthor = typeof snapshot.author === "string" ? snapshot.author : author;

    setZhTitle(nextZhTitle);
    setEnTitle(nextEnTitle);
    setSlug(nextSlug);
    setZhBody(nextZhBody);
    setEnBody(nextEnBody);
    setCoverImage(nextCover);
    setAuthor(nextAuthor);
    if (typeof snapshot.zhSeoTitle === "string") setZhSeoTitle(snapshot.zhSeoTitle);
    if (typeof snapshot.enSeoTitle === "string") setEnSeoTitle(snapshot.enSeoTitle);
    if (typeof snapshot.zhMetaDescription === "string") setZhMetaDescription(snapshot.zhMetaDescription);
    if (typeof snapshot.enMetaDescription === "string") setEnMetaDescription(snapshot.enMetaDescription);
    if (typeof snapshot.ogImage === "string") setOgImage(snapshot.ogImage);

    zhEditor?.commands.setContent(nextZhBody || "", { emitUpdate: false });
    enEditor?.commands.setContent(nextEnBody || "", { emitUpdate: false });
    if (editorMode === "markdown") {
      setMarkdownContent({
        zh: htmlToMarkdown(nextZhBody || ""),
        en: htmlToMarkdown(nextEnBody || ""),
      });
    }

    setShowVersionHistory(false);
    setSuccessMessage("已恢复到所选版本（尚未保存，请检查后保存）");
    window.setTimeout(() => setSuccessMessage(""), 4000);

    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        readyRef.current = true;
        setIsDirty(true);
        setSavePhase("dirty");
      });
    });
  }, [slug, author, zhEditor, enEditor, editorMode]);

  // Word counts for language tabs (debounced)
  useEffect(() => {
    const t = window.setTimeout(() => {
      const zhText =
        editorMode === "markdown"
          ? (markdownContent.zh ?? "")
          : (zhEditor?.getText() || htmlToPlainText(zhBody));
      const enText =
        editorMode === "markdown"
          ? (markdownContent.en ?? "")
          : (enEditor?.getText() || htmlToPlainText(enBody));
      setWordStats({
        zh: countWords(zhText),
        en: countWords(enText),
      });
    }, 250);
    return () => window.clearTimeout(t);
  }, [editorMode, markdownContent, zhBody, enBody, zhEditor, enEditor, isDirty, saving]);

  // Auto-enable English when entering split layout
  useEffect(() => {
    if (viewLayout === "split" && !enabledLangs.includes("en")) {
      setEnabledLangs((prev) => (prev.includes("en") ? prev : [...prev, "en"]));
    }
  }, [viewLayout, enabledLangs]);

  const handleCopyToOtherLang = useCallback(
    (from: "zh" | "en") => {
      const to = from === "zh" ? "en" : "zh";
      if (editorMode === "markdown") {
        const src = markdownContent[from] ?? "";
        setMarkdownContent((prev) => ({ ...prev, [to]: src }));
      } else {
        const srcEd = from === "zh" ? zhEditor : enEditor;
        const dstEd = to === "zh" ? zhEditor : enEditor;
        const html = srcEd?.getHTML() || (from === "zh" ? zhBody : enBody) || "";
        dstEd?.commands.setContent(html, { emitUpdate: false });
        if (to === "zh") setZhBody(html);
        else setEnBody(html);
      }
      if (from === "zh") {
        if (zhTitle.trim()) setEnTitle(zhTitle);
      } else if (enTitle.trim()) {
        setZhTitle(enTitle);
      }
      touch();
      setSuccessMessage(from === "zh" ? "已复制中文到英文（未保存）" : "已复制英文到中文（未保存）");
      window.setTimeout(() => setSuccessMessage(""), 2500);
    },
    [editorMode, markdownContent, zhEditor, enEditor, zhBody, enBody, zhTitle, enTitle, touch],
  );

  const handleTranslateToOtherLang = useCallback(
    async (from: "zh" | "en") => {
      const to = from === "zh" ? "en" : "zh";
      setTranslateBusy(true);
      setError(null);
      try {
        const srcTitle = from === "zh" ? zhTitle : enTitle;
        let srcBodyPlain: string;
        let srcMarkdown = "";
        if (editorMode === "markdown") {
          srcMarkdown = markdownContent[from] ?? "";
          srcBodyPlain = srcMarkdown;
        } else {
          const srcEd = from === "zh" ? zhEditor : enEditor;
          const html = srcEd?.getHTML() || (from === "zh" ? zhBody : enBody) || "";
          srcBodyPlain = htmlToPlainText(html);
        }
        if (!srcTitle.trim() && !srcBodyPlain.trim()) {
          setError("源语言内容为空，无法翻译");
          return;
        }
        const sourceLang = from === "zh" ? "zh" : "en";
        const targetLang = to === "zh" ? "zh" : "en";
        if (srcTitle.trim()) {
          const tr = await translateText({ text: srcTitle, sourceLang, targetLang });
          if (to === "zh") setZhTitle(tr.translatedText);
          else setEnTitle(tr.translatedText);
        }
        if (srcBodyPlain.trim()) {
          const tr = await translateText({ text: srcBodyPlain, sourceLang, targetLang });
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
        if (!enabledLangs.includes(to)) {
          setEnabledLangs((prev) => (prev.includes(to) ? prev : [...prev, to]));
        }
        touch();
        setSuccessMessage(from === "zh" ? "已翻译到英文（未保存，请校对）" : "已翻译到中文（未保存，请校对）");
        window.setTimeout(() => setSuccessMessage(""), 3000);
      } catch (err: any) {
        const msg = err?.response?.data?.error?.message ?? err?.response?.data?.error;
        setError(msg || (err instanceof Error ? err.message : "翻译失败（请检查 AI/翻译配置）"));
      } finally {
        setTranslateBusy(false);
      }
    },
    [
      zhTitle, enTitle, editorMode, markdownContent, zhEditor, enEditor, zhBody, enBody,
      enabledLangs, touch,
    ],
  );


  const handleSchedulePublish = async (scheduledAt: string) => {
    if (!canPublish) return;
    if (!zhTitle.trim()) { setError("请填写中文标题"); return; }
    const finalSlug = slug.trim() || slugifyTitle(zhTitle);
    if (!slug.trim()) setSlug(finalSlug);
    setScheduleBusy(true);
    setError(null);
    setScheduleMessage("");
    try {
      const publishPayload = buildPayload("published", scheduledAt);
      if (scheduledPublication?.status === "pending") {
        const updated = await updateScheduledPublication(scheduledPublication.id, {
          scheduledAt,
          publishPayload,
        });
        setScheduledPublication(updated);
        setScheduleMessage("定时发布已更新");
      } else {
        let resourceId = Number(id);
        if (!isEditing) {
          const created = await createArticle(buildPayload("draft") as Partial<Article>);
          resourceId = created.id;
        }
        const createdSchedule = await createScheduledPublication({
          resourceType: "article",
          resourceId,
          scheduledAt,
          publishPayload,
        });
        setScheduledPublication(createdSchedule);
        setArticleStatus(articleStatus === "published" ? "published" : "scheduled");
        setScheduleMessage("定时发布已安排");
        if (!isEditing) {
          navigate(`/admin/articles/edit/${resourceId}`, { replace: true });
        }
      }
    } catch (err: any) {
      const msg = err?.response?.data?.error?.message ?? err?.response?.data?.error;
      setError(msg || (err instanceof Error ? err.message : "定时发布失败"));
    } finally {
      setScheduleBusy(false);
    }
  };

  const handleCancelSchedule = async () => {
    if (!scheduledPublication || !canPublish) return;
    setScheduleBusy(true);
    setError(null);
    setScheduleMessage("");
    try {
      await cancelScheduledPublication(scheduledPublication.id);
      setScheduledPublication(null);
      setScheduleMessage("定时发布已取消");
    } catch (err) {
      setError(err instanceof Error ? err.message : "取消定时发布失败");
    } finally {
      setScheduleBusy(false);
    }
  };

  const handleRetrySchedule = async () => {
    if (!scheduledPublication || !canPublish) return;
    setScheduleBusy(true);
    setError(null);
    setScheduleMessage("");
    try {
      const retried = await retryScheduledPublication(scheduledPublication.id);
      setScheduledPublication(retried);
      setScheduleMessage("定时发布已重新入队");
    } catch (err) {
      setError(err instanceof Error ? err.message : "重试定时发布失败");
    } finally {
      setScheduleBusy(false);
    }
  };

  const handleModeChange = (newMode: "richtext" | "markdown") => {
    if (newMode === editorMode) return;

    const zhHtml = editorMode === "markdown"
      ? markdownToHtml(markdownContent.zh ?? "")
      : (zhEditor?.getHTML() || zhBody || "");
    const enHtml = editorMode === "markdown"
      ? markdownToHtml(markdownContent.en ?? "")
      : (enEditor?.getHTML() || enBody || "");
    const hasBody =
      (zhHtml && zhHtml !== "<p></p>" && zhHtml.replace(/<[^>]+>/g, "").trim().length > 0) ||
      (enHtml && enHtml !== "<p></p>" && enHtml.replace(/<[^>]+>/g, "").trim().length > 0);

    if (hasBody || isDirty) {
      if (!window.confirm(MODE_SWITCH_MESSAGE)) return;
    }

    if (newMode === "markdown" && editorMode === "richtext") {
      // Richtext HTML → Markdown (mermaid, tables, code fences preserved)
      const next: Record<string, string> = { ...markdownContent };
      for (const lang of ["zh", "en"]) {
        const ed = langEditors[lang]?.editor;
        const html = ed?.getHTML() ?? (lang === "zh" ? zhBody : enBody) ?? "";
        next[lang] = htmlToMarkdown(html || "");
      }
      setMarkdownContent(next);
    } else if (newMode === "richtext" && editorMode === "markdown") {
      // Markdown → HTML → TipTap schema (mermaid node + tables)
      for (const lang of ["zh", "en"]) {
        const ed = langEditors[lang]?.editor;
        const html = markdownToHtml(markdownContent[lang] ?? "");
        ed?.commands.setContent(html, { emitUpdate: false });
        // Re-read after schema normalization so state matches editor
        const normalized = ed?.getHTML() || html;
        if (lang === "zh") setZhBody(normalized);
        if (lang === "en") setEnBody(normalized);
      }
      setMarkdownApi(null);
    }
    setEditorMode(newMode);
    touch();
  };

  const toggleCategory = (catId: number) => {
    setSelectedCategoryIds((prev) => prev.includes(catId) ? prev.filter((i) => i !== catId) : [...prev, catId]);
    touch();
  };
  const toggleTag = (tagId: number) => {
    setSelectedTagIds((prev) => prev.includes(tagId) ? prev.filter((i) => i !== tagId) : [...prev, tagId]);
    touch();
  };

  const addLang = (langKey: string) => {
    if (!enabledLangs.includes(langKey)) {
      const newLangs = [...enabledLangs, langKey];
      setEnabledLangs(newLangs);
      setActiveLangIdx(newLangs.length - 1);
    }
    setShowLangMenu(false);
  };

  const removeLang = (langKey: string) => {
    if (langKey === "zh") return; // Can't remove default
    const newLangs = enabledLangs.filter((l) => l !== langKey);
    setEnabledLangs(newLangs);
    if (activeLangIdx >= newLangs.length) setActiveLangIdx(newLangs.length - 1);
  };

  // Memoize sidebar props so EditorSidebar skips re-renders on unrelated state changes
  const sidebarArticle = useMemo(
    () => isEditing ? { slug, author, createdAt: articleCreatedAt, publishedAt: articlePublishedAt } : null,
    [isEditing, slug, author, articleCreatedAt, articlePublishedAt],
  );

  if (loading) {
    return <div className="flex items-center justify-center h-64"><div className="text-gray-600">加载中...</div></div>;
  }

  // Titles per language for the top bar
  const langTitleMap: Record<string, { title: string; setTitle: (v: string) => void; placeholder: string }> = {
    zh: { title: zhTitle, setTitle: track(setZhTitle), placeholder: "输入中文标题" },
    en: { title: enTitle, setTitle: track(setEnTitle), placeholder: "Enter English title" },
  };

  return (
    <div className="flex flex-col h-full min-h-0 bg-white">
      {/* Header stack: action bar → language tabs → toolbar (never overlays content) */}
      <div className="flex-shrink-0 z-20 bg-white border-b border-gray-200 shadow-sm">
        {/* Row 1: Back + Title + Actions */}
        <div className="flex items-center gap-3 px-4 py-2">
          <button onClick={handleBack} className="text-gray-500 hover:text-gray-700 text-sm flex-shrink-0">
            &larr; 返回
          </button>
          <input
            type="text"
            value={langTitleMap[activeLang]?.title || ""}
            onChange={(e) => langTitleMap[activeLang]?.setTitle(e.target.value)}
            className="flex-1 px-3 py-1.5 text-base font-semibold border border-transparent rounded-lg hover:border-gray-300 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none transition-colors bg-transparent"
            placeholder={langTitleMap[activeLang]?.placeholder || "标题"}
          />
          <div className="flex items-center gap-1.5 flex-shrink-0">
            <SaveStatusBadge
              phase={savePhase}
              lastSavedAt={lastSavedAt}
              isAutosave={lastSaveWasAutosave}
            />
            <PopoverButton label="基本信息" active={showBasicInfo} onClick={() => { setShowBasicInfo(!showBasicInfo); setShowSeo(false); setShowAdvanced(false); }} />
            <PopoverButton label="SEO" active={showSeo} onClick={() => { setShowSeo(!showSeo); setShowBasicInfo(false); setShowAdvanced(false); }} />
            <PopoverButton label="高级" active={showAdvanced} onClick={() => { setShowAdvanced(!showAdvanced); setShowBasicInfo(false); setShowSeo(false); }} />
            {isEditing && (
              <PopoverButton
                label="历史版本"
                active={showVersionHistory}
                onClick={() => {
                  setVersionDraftSnapshot(buildCurrentDraftSnapshot());
                  setShowVersionHistory(true);
                }}
              />
            )}
            <button
              type="button"
              onClick={openPreview}
              title="预览 (⌘P / Ctrl+P)"
              className="px-2.5 py-1.5 text-xs border border-gray-300 rounded-lg hover:bg-gray-50 text-gray-700"
            >
              预览
            </button>
            <span className="w-px h-6 bg-gray-200 mx-1" />
            <ScheduledPublicationPanel
              compact
              item={scheduledPublication}
              loading={scheduleLoading}
              busy={scheduleBusy}
              canPublish={canPublish}
              disabledReason="需要 articles:publish 权限才能安排定时发布。"
              onSchedule={handleSchedulePublish}
              onCancel={handleCancelSchedule}
              onRetry={handleRetrySchedule}
              onRefresh={loadArticleSchedule}
              title={articleStatus === "published" ? "定时更新" : "定时"}
            />
            <span className="w-px h-6 bg-gray-200 mx-1" />
            <button
              onClick={() => void handleSave("draft")}
              disabled={saving}
              title="保存 (⌘S / Ctrl+S)"
              className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
            >
              {saving ? "保存中..." : "保存"}
            </button>
            {canPublish && (
              <button
                onClick={() => void handleSave("publish")}
                disabled={saving}
                title="发布 (⌘⇧S / Ctrl+Shift+S)"
                className="px-3 py-1.5 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50"
              >
                {saving ? "发布中..." : "发布"}
              </button>
            )}
          </div>
        </div>

        {/* Settings Panels (slide down) */}
        {showBasicInfo && (
          <ArticleForm
            slug={slug} setSlug={track(setSlug)}
            author={author} setAuthor={track(setAuthor)}
            coverImage={coverImage} setCoverImage={track(setCoverImage)}
            showCoverPicker={showCoverPicker} setShowCoverPicker={setShowCoverPicker}
            categories={categories} selectedCategoryIds={selectedCategoryIds} toggleCategory={toggleCategory}
            tags={tags} selectedTagIds={selectedTagIds} toggleTag={toggleTag}
          />
        )}

        {showSeo && (
          <SeoFieldsPanel
            zhSeoTitle={zhSeoTitle} setZhSeoTitle={track(setZhSeoTitle)}
            enSeoTitle={enSeoTitle} setEnSeoTitle={track(setEnSeoTitle)}
            zhMetaDescription={zhMetaDescription} setZhMetaDescription={track(setZhMetaDescription)}
            enMetaDescription={enMetaDescription} setEnMetaDescription={track(setEnMetaDescription)}
            ogImage={ogImage} setOgImage={track(setOgImage)}
          />
        )}

        {showAdvanced && (
          <AdvancedSettingsPanel
            visibility={visibility} setVisibility={track(setVisibility)}
            autoSummary={autoSummary} setAutoSummary={track(setAutoSummary)}
            allowComments={allowComments} setAllowComments={track(setAllowComments)}
            pinned={pinned} setPinned={track(setPinned)}
            metadata={metadata} setMetadata={track(setMetadata)}
          />
        )}

        {/* Row 2: Language tabs — above toolbar so they are never covered */}
        <div className="flex items-center gap-1 px-4 pt-1.5 pb-0 border-t border-gray-100 bg-gray-50/80">
          {enabledLangs.map((lang, idx) => {
            const info = ALL_LANGS.find((l) => l.key === lang);
            return (
              <button
                key={lang}
                onClick={() => setActiveLangIdx(idx)}
                className={`group relative px-4 py-1.5 text-sm rounded-t-lg border border-b-0 transition-colors ${
                  idx === activeLangIdx
                    ? "bg-white border-gray-300 text-gray-900 font-medium shadow-sm"
                    : "bg-transparent border-transparent text-gray-500 hover:text-gray-700 hover:bg-gray-100/80"
                }`}
              >
                <span>{info?.label || lang}</span>
                <span className="ml-1.5 text-[10px] text-gray-400 font-normal tabular-nums">
                  {(wordStats[lang as "zh" | "en"]?.words ?? 0).toLocaleString()} 词
                </span>
                {lang !== "zh" && viewLayout !== "split" && (
                  <span
                    onClick={(e) => { e.stopPropagation(); removeLang(lang); }}
                    className="ml-1.5 text-gray-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity"
                  >
                    &times;
                  </span>
                )}
              </button>
            );
          })}

          {enabledLangs.length < ALL_LANGS.length && viewLayout !== "split" && (
            <div className="relative" ref={langMenuRef}>
              <button onClick={() => setShowLangMenu(!showLangMenu)}
                className="px-2 py-1.5 text-sm text-gray-400 hover:text-gray-600 rounded-t-lg hover:bg-gray-100"
                title="添加语言">
                + 语言
              </button>
              {showLangMenu && (
                <div className="absolute top-full left-0 mt-0.5 py-1 bg-white rounded-lg shadow-lg border border-gray-200 z-40 min-w-[100px]">
                  {ALL_LANGS.filter((l) => !enabledLangs.includes(l.key)).map((l) => (
                    <button key={l.key} onClick={() => addLang(l.key)}
                      className="w-full text-left px-3 py-1.5 text-sm hover:bg-gray-100 text-gray-700">
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
              onClick={() => setViewLayout((v) => (v === "split" ? "focus" : "split"))}
              title="中英并排编辑"
              className={`px-2 py-1 text-xs rounded-lg border transition-colors ${
                viewLayout === "split"
                  ? "bg-blue-50 border-blue-300 text-blue-800"
                  : "border-gray-200 text-gray-600 hover:bg-gray-100"
              }`}
            >
              并排
            </button>
            {enabledLangs.includes("en") && (
              <>
                <button
                  type="button"
                  disabled={translateBusy}
                  onClick={() => handleCopyToOtherLang("zh")}
                  className="px-2 py-1 text-xs rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-100 disabled:opacity-50"
                  title="将中文标题与正文复制到英文"
                >
                  中→英 复制
                </button>
                <button
                  type="button"
                  disabled={translateBusy}
                  onClick={() => void handleTranslateToOtherLang("zh")}
                  className="px-2 py-1 text-xs rounded-lg border border-violet-200 text-violet-700 hover:bg-violet-50 disabled:opacity-50"
                  title="使用翻译 API 将中文译为英文"
                >
                  {translateBusy ? "翻译中…" : "中→英 翻译"}
                </button>
              </>
            )}
            <EditorModeSwitcher mode={editorMode} onModeChange={handleModeChange} />
          </div>
        </div>

        {/* Row 3: Editor Toolbar for active language/mode */}
        <div className="flex items-stretch border-t border-gray-200 bg-gray-50">
          {editorMode === "richtext" && activeEntry?.editor ? (
            <div className="flex-1 min-w-0 overflow-x-auto">
              <EditorToolbar editor={activeEntry.editor} modals={activeEntry.modals} />
            </div>
          ) : editorMode === "markdown" ? (
            <div className="flex-1 min-w-0 overflow-x-auto">
              <MarkdownToolbar api={markdownApi} />
            </div>
          ) : (
            <div className="flex-1 py-2" />
          )}
        </div>
      </div>

      {/* Error / success bars */}
      {error && (
        <div className="px-4 py-2 bg-red-50 border-b border-red-200 text-red-800 text-sm flex-shrink-0">
          {error}
          <button onClick={() => setError(null)} className="ml-2 text-red-600 hover:text-red-800">&times;</button>
        </div>
      )}
      {(scheduleMessage || successMessage) && (
        <div className="px-4 py-2 bg-green-50 border-b border-green-200 text-green-800 text-sm flex-shrink-0">
          {successMessage || scheduleMessage}
          <button
            onClick={() => { setScheduleMessage(""); setSuccessMessage(""); }}
            className="ml-2 text-green-600 hover:text-green-800"
          >
            &times;
          </button>
        </div>
      )}

      {/* Main Content: Editor + Sidebar */}
      <div className="flex-1 flex min-h-0">
        <div className="flex-1 flex flex-col min-h-0 min-w-0 bg-white">
          <div className={`flex-1 min-h-0 ${editorMode === "markdown" || viewLayout === "split" ? "overflow-hidden" : "overflow-y-auto"}`}>
            {viewLayout === "split" ? (
              <div className="h-full min-h-0 grid grid-cols-1 md:grid-cols-2 gap-0 divide-x divide-gray-200">
                {(["zh", "en"] as const).map((lang) => {
                  const entry = langEditors[lang];
                  const isActiveCol = activeLang === lang;
                  return (
                    <div
                      key={lang}
                      className={`flex flex-col min-h-0 min-w-0 ${isActiveCol ? "bg-white" : "bg-gray-50/40"}`}
                      onMouseDown={() => {
                        const idx = enabledLangs.indexOf(lang);
                        if (idx >= 0) setActiveLangIdx(idx);
                      }}
                    >
                      <div className="flex-shrink-0 px-3 py-2 border-b border-gray-100 space-y-1.5">
                        <div className="flex items-center justify-between gap-2">
                          <span className={`text-xs font-semibold ${isActiveCol ? "text-blue-700" : "text-gray-500"}`}>
                            {lang === "zh" ? "中文" : "English"}
                            {isActiveCol && (
                              <span className="ml-1.5 text-[10px] font-normal text-blue-500">编辑中</span>
                            )}
                          </span>
                          <span className="text-[10px] text-gray-400 tabular-nums">
                            {wordStats[lang].words.toLocaleString()} 词 · {wordStats[lang].chars.toLocaleString()} 字
                          </span>
                        </div>
                        <input
                          type="text"
                          value={langTitleMap[lang]?.title || ""}
                          onChange={(e) => langTitleMap[lang]?.setTitle(e.target.value)}
                          onFocus={() => {
                            const idx = enabledLangs.indexOf(lang);
                            if (idx >= 0) setActiveLangIdx(idx);
                          }}
                          className="w-full px-2 py-1 text-sm font-medium border border-gray-200 rounded-lg focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
                          placeholder={langTitleMap[lang]?.placeholder || "标题"}
                        />
                      </div>
                      <div className="flex-1 min-h-0">
                        {editorMode === "markdown" ? (
                          <div className="h-full min-h-0 p-2">
                            <MarkdownMode
                              key={`split-${lang}`}
                              contentKey={lang}
                              label={lang === "zh" ? "Markdown · 中文" : "Markdown · EN"}
                              showPreview={false}
                              value={markdownContent[lang] ?? ""}
                              onChange={(val) => {
                                setMarkdownContent((prev) => ({ ...prev, [lang]: val }));
                                touch();
                              }}
                              onApiReady={lang === activeLang ? setMarkdownApi : undefined}
                            />
                          </div>
                        ) : entry?.editor ? (
                          <div className="h-full overflow-y-auto">
                            {isActiveCol && (
                              <>
                                <EditorBubbleMenu editor={entry.editor} />
                                <TableBubbleMenu editor={entry.editor} />
                                <EditorFloatingMenu editor={entry.editor} />
                              </>
                            )}
                            <ArticleTypographyRoot
                              mode="editor"
                              articleMetadata={metadata}
                              className="h-full article-editor-content"
                            >
                              <EditorContent editor={entry.editor} className="h-full" />
                            </ArticleTypographyRoot>
                          </div>
                        ) : null}
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : editorMode === "markdown" ? (
              <div className="h-full min-h-0 p-3">
                <MarkdownMode
                  key={activeLang}
                  contentKey={activeLang}
                  value={markdownContent[activeLang] ?? ""}
                  onChange={(val) => {
                    setMarkdownContent((prev) => ({ ...prev, [activeLang]: val }));
                    touch();
                  }}
                  onApiReady={setMarkdownApi}
                />
              </div>
            ) : (
              enabledLangs.map((lang, idx) => {
                const entry = langEditors[lang];
                if (!entry?.editor) return null;
                return (
                  <div key={lang} className={idx === activeLangIdx ? "h-full" : "hidden"}>
                    <EditorBubbleMenu editor={entry.editor} />
                    <TableBubbleMenu editor={entry.editor} />
                    <EditorFloatingMenu editor={entry.editor} />
                    <ArticleTypographyRoot
                      mode="editor"
                      articleMetadata={metadata}
                      className="h-full article-editor-content"
                    >
                      <EditorContent editor={entry.editor} className="h-full" />
                    </ArticleTypographyRoot>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {editorMode === "richtext" && viewLayout === "focus" && (
          <EditorSidebar
            editor={activeEntry?.editor ?? null}
            article={sidebarArticle}
          />
        )}
      </div>

      {Object.entries(langEditors).map(([lang, entry]) =>
        entry.editor ? <EditorModals key={lang} editor={entry.editor} state={entry.state} /> : null
      )}

      <ImagePickerModal
        open={showCoverPicker}
        onClose={() => setShowCoverPicker(false)}
        onSelect={(item) => { setCoverImage(item.url); setShowCoverPicker(false); touch(); }}
      />

      {showVersionHistory && isEditing && (
        <ArticleVersionHistoryPanel
          articleId={Number(id)}
          onClose={() => {
            setShowVersionHistory(false);
            setVersionDraftSnapshot(null);
          }}
          currentDraft={versionDraftSnapshot}
          onRestore={handleRestoreVersion}
          canRestore
        />
      )}

      <ArticlePreviewModal
        open={showPreview}
        data={previewData}
        onClose={() => setShowPreview(false)}
      />
    </div>
  );
}
