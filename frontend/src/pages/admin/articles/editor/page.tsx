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
import TurndownService from "turndown";
import { markdownToHtml } from "@/lib/markdown";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useAuth } from "@/contexts/AuthContext";
import EditorSidebar from "./EditorSidebar";
import ArticleForm from "./ArticleForm";
import { SeoFieldsPanel, AdvancedSettingsPanel, PopoverButton } from "./SeoFields";
import ArticleTypographyRoot from "@/components/blog/ArticleTypographyRoot";

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

  // Editor mode (richtext / markdown)
  const [editorMode, setEditorMode] = useState<"richtext" | "markdown">("richtext");
  const [markdownContent, setMarkdownContent] = useState<Record<string, string>>({ zh: "", en: "" });
  // Track whether current article has been loaded (avoid re-fetch wiping edits)
  const loadedIdRef = useRef<string | null>(null);

  // Panel states
  const [showBasicInfo, setShowBasicInfo] = useState(false);
  const [showSeo, setShowSeo] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showCoverPicker, setShowCoverPicker] = useState(false);

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
  const zhEditor = useEditor({
    extensions: zhExtensions,
    content: zhBody,
    editorProps: { attributes: { class: "tiptap" } },
  });

  const enEditor = useEditor({
    extensions: enExtensions,
    content: enBody,
    editorProps: { attributes: { class: "tiptap" } },
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
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load article");
    } finally {
      setLoading(false);
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
      return;
    }
    if (loadedIdRef.current !== id) {
      void loadArticle();
    }
    void loadArticleSchedule();
  }, [id, loadArticle, loadArticleSchedule]);

  /** Resolve body HTML from the active editor mode. */
  const resolveBodies = useCallback(() => {
    if (editorMode === "markdown") {
      return {
        zhBody: markdownToHtml(markdownContent.zh ?? ""),
        enBody: markdownToHtml(markdownContent.en ?? ""),
      };
    }
    return {
      zhBody: zhEditor?.getHTML() || zhBody || "",
      enBody: enEditor?.getHTML() || enBody || "",
    };
  }, [editorMode, markdownContent, zhEditor, enEditor, zhBody, enBody]);

  const buildPayload = (status: "draft" | "published", publishedAt?: string): Record<string, unknown> => {
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
  };

  const handleSave = async (status: "draft" | "published") => {
    if (!zhTitle.trim()) { setError("请填写中文标题"); return; }
    // Auto-fill slug from title when empty so save is not blocked by hidden form field
    const finalSlug = slug.trim() || slugifyTitle(zhTitle);
    if (!slug.trim()) setSlug(finalSlug);

    setSaving(true);
    setError(null);
    setSuccessMessage("");
    try {
      const payload = buildPayload(status);
      if (isEditing) {
        await updateArticle(Number(id), payload as Partial<Article>);
        setArticleStatus(status);
        setSuccessMessage(status === "published" ? "已发布" : "已保存");
      } else {
        const created = await createArticle(payload as Partial<Article>);
        setArticleStatus(status);
        loadedIdRef.current = String(created.id);
        setSuccessMessage(status === "published" ? "已创建并发布" : "已保存");
        navigate(`/admin/articles/edit/${created.id}`, { replace: true });
      }
      window.setTimeout(() => setSuccessMessage(""), 3000);
    } catch (err: any) {
      const msg = err?.response?.data?.error?.message;
      setError(msg || (err instanceof Error ? err.message : "保存失败"));
    } finally {
      setSaving(false);
    }
  };

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
    if (newMode === "markdown" && editorMode === "richtext") {
      const turndown = new TurndownService();
      // Convert both language buffers so switching tabs in MD mode stays consistent
      const next: Record<string, string> = { ...markdownContent };
      for (const lang of enabledLangs) {
        const ed = langEditors[lang]?.editor;
        const html = ed?.getHTML() ?? (lang === "zh" ? zhBody : enBody) ?? "";
        next[lang] = turndown.turndown(html || "");
      }
      setMarkdownContent(next);
    } else if (newMode === "richtext" && editorMode === "markdown") {
      for (const lang of enabledLangs) {
        const ed = langEditors[lang]?.editor;
        const html = markdownToHtml(markdownContent[lang] ?? "");
        ed?.commands.setContent(html);
        if (lang === "zh") setZhBody(html);
        if (lang === "en") setEnBody(html);
      }
    }
    setEditorMode(newMode);
  };

  const toggleCategory = (catId: number) => {
    setSelectedCategoryIds((prev) => prev.includes(catId) ? prev.filter((i) => i !== catId) : [...prev, catId]);
  };
  const toggleTag = (tagId: number) => {
    setSelectedTagIds((prev) => prev.includes(tagId) ? prev.filter((i) => i !== tagId) : [...prev, tagId]);
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
    zh: { title: zhTitle, setTitle: setZhTitle, placeholder: "输入中文标题" },
    en: { title: enTitle, setTitle: setEnTitle, placeholder: "Enter English title" },
  };

  return (
    <div className="flex flex-col h-[calc(100vh-3.5rem)] -m-6">
      {/* Sticky Header: Action Bar */}
      <div className="sticky top-0 z-30 bg-white border-b border-gray-200 shadow-sm flex-shrink-0">
        {/* Row 1: Back + Title + Actions */}
        <div className="flex items-center gap-3 px-4 py-2">
          <button onClick={() => navigate("/admin/articles")} className="text-gray-500 hover:text-gray-700 text-sm flex-shrink-0">
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
            {/* Settings popover buttons */}
            <PopoverButton label="基本信息" active={showBasicInfo} onClick={() => { setShowBasicInfo(!showBasicInfo); setShowSeo(false); setShowAdvanced(false); }} />
            <PopoverButton label="SEO" active={showSeo} onClick={() => { setShowSeo(!showSeo); setShowBasicInfo(false); setShowAdvanced(false); }} />
            <PopoverButton label="高级" active={showAdvanced} onClick={() => { setShowAdvanced(!showAdvanced); setShowBasicInfo(false); setShowSeo(false); }} />
            <span className="w-px h-6 bg-gray-200 mx-1" />
            {/* Compact scheduled publish inline with actions */}
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
            <button onClick={() => handleSave("draft")} disabled={saving}
              className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50">
              {saving ? "保存中..." : "保存"}
            </button>
            {canPublish && (
              <button onClick={() => handleSave("published")} disabled={saving}
                className="px-3 py-1.5 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50">
                {saving ? "发布中..." : "发布"}
              </button>
            )}
          </div>
        </div>

        {/* Settings Panels (slide down) */}
        {showBasicInfo && (
          <ArticleForm
            slug={slug} setSlug={setSlug}
            author={author} setAuthor={setAuthor}
            coverImage={coverImage} setCoverImage={setCoverImage}
            showCoverPicker={showCoverPicker} setShowCoverPicker={setShowCoverPicker}
            categories={categories} selectedCategoryIds={selectedCategoryIds} toggleCategory={toggleCategory}
            tags={tags} selectedTagIds={selectedTagIds} toggleTag={toggleTag}
          />
        )}

        {showSeo && (
          <SeoFieldsPanel
            zhSeoTitle={zhSeoTitle} setZhSeoTitle={setZhSeoTitle}
            enSeoTitle={enSeoTitle} setEnSeoTitle={setEnSeoTitle}
            zhMetaDescription={zhMetaDescription} setZhMetaDescription={setZhMetaDescription}
            enMetaDescription={enMetaDescription} setEnMetaDescription={setEnMetaDescription}
            ogImage={ogImage} setOgImage={setOgImage}
          />
        )}

        {showAdvanced && (
          <AdvancedSettingsPanel
            visibility={visibility} setVisibility={setVisibility}
            autoSummary={autoSummary} setAutoSummary={setAutoSummary}
            allowComments={allowComments} setAllowComments={setAllowComments}
            pinned={pinned} setPinned={setPinned}
            metadata={metadata} setMetadata={setMetadata}
          />
        )}

        {/* Row 2: Editor Toolbar (for active language editor) + Mode Switcher */}
        <div className="flex items-center border-t border-gray-100">
          {editorMode === "richtext" && activeEntry?.editor ? (
            <div className="flex-1">
              <EditorToolbar editor={activeEntry.editor} modals={activeEntry.modals} />
            </div>
          ) : (
            <div className="flex-1" />
          )}
          <div className="px-3 py-1.5 flex-shrink-0">
            <EditorModeSwitcher mode={editorMode} onModeChange={handleModeChange} />
          </div>
        </div>
      </div>

      {/* Error Bar */}
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
        {/* Left: Language Carousel + Editor Content */}
        <div className="flex-1 flex flex-col min-h-0 min-w-0 bg-white">
          {/* Language tabs + add button */}
          <div className="flex items-center gap-1 px-4 pt-2 pb-0 flex-shrink-0">
            {enabledLangs.map((lang, idx) => {
              const info = ALL_LANGS.find((l) => l.key === lang);
              return (
                <button
                  key={lang}
                  onClick={() => setActiveLangIdx(idx)}
                  className={`group relative px-4 py-1.5 text-sm rounded-t-lg border border-b-0 transition-colors ${
                    idx === activeLangIdx
                      ? "bg-white border-gray-300 text-gray-900 font-medium"
                      : "bg-gray-50 border-transparent text-gray-500 hover:text-gray-700 hover:bg-gray-100"
                  }`}
                >
                  {info?.label || lang}
                  {lang !== "zh" && (
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

            {/* Add language button */}
            {enabledLangs.length < ALL_LANGS.length && (
              <div className="relative" ref={langMenuRef}>
                <button onClick={() => setShowLangMenu(!showLangMenu)}
                  className="px-2 py-1.5 text-sm text-gray-400 hover:text-gray-600 rounded-t-lg hover:bg-gray-50"
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

            {/* Carousel navigation arrows */}
            {enabledLangs.length > 1 && (
              <div className="ml-auto flex items-center gap-1">
                <button
                  onClick={() => setActiveLangIdx((i) => Math.max(0, i - 1))}
                  disabled={activeLangIdx === 0}
                  className="p-1 text-gray-400 hover:text-gray-600 disabled:opacity-30"
                >&larr;</button>
                <span className="text-xs text-gray-400">{activeLangIdx + 1}/{enabledLangs.length}</span>
                <button
                  onClick={() => setActiveLangIdx((i) => Math.min(enabledLangs.length - 1, i + 1))}
                  disabled={activeLangIdx === enabledLangs.length - 1}
                  className="p-1 text-gray-400 hover:text-gray-600 disabled:opacity-30"
                >&rarr;</button>
              </div>
            )}
          </div>

          {/* Editor Content Area — fill remaining space */}
          <div className={`flex-1 min-h-0 border-t border-gray-300 ${editorMode === "markdown" ? "overflow-hidden" : "overflow-y-auto"}`}>
            {editorMode === "markdown" ? (
              <div className="h-full min-h-0 p-3">
                <MarkdownMode
                  value={markdownContent[activeLang] ?? ""}
                  onChange={(val) => setMarkdownContent((prev) => ({ ...prev, [activeLang]: val }))}
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

        {/* Right: Sidebar — Outline + Details (richtext only; markdown has its own live preview) */}
        {editorMode === "richtext" && (
          <EditorSidebar
            editor={activeEntry?.editor ?? null}
            article={sidebarArticle}
          />
        )}
      </div>

      {/* Modals */}
      {Object.entries(langEditors).map(([lang, entry]) =>
        entry.editor ? <EditorModals key={lang} editor={entry.editor} state={entry.state} /> : null
      )}

      <ImagePickerModal
        open={showCoverPicker}
        onClose={() => setShowCoverPicker(false)}
        onSelect={(item) => { setCoverImage(item.url); setShowCoverPicker(false); }}
      />
    </div>
  );
}
