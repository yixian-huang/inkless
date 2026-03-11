import { useState, useEffect, useCallback, useRef, useMemo, memo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useEditor, useEditorState, EditorContent } from "@tiptap/react";
import type { Editor } from "@tiptap/react";
import {
  getAdminArticle,
  createArticle,
  updateArticle,
  getCategories,
  getTags,
} from "@/api/articles";
import type { Article, Category, Tag } from "@/api/articles";
import MetadataEditor from "@/components/admin/MetadataEditor";
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

export default function ArticleEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEditing = !!id;

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

  // UI state
  const [loading, setLoading] = useState(isEditing);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);

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

  // Load article for editing
  const loadArticle = useCallback(async () => {
    if (!id) return;
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
      // Auto-enable English if it has content
      if (article.enBody || article.enTitle) {
        setEnabledLangs(["zh", "en"]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load article");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadMeta();
    if (isEditing) loadArticle();
  }, [loadMeta, loadArticle, isEditing]);

  const buildPayload = (status: "draft" | "published"): Record<string, unknown> => {
    const payload: Record<string, unknown> = {
      zhTitle, enTitle, slug, coverImage,
      zhBody: zhEditor?.getHTML() || "",
      enBody: enEditor?.getHTML() || "",
      zhSeoTitle, enSeoTitle, zhMetaDescription, enMetaDescription, ogImage,
      status, categoryIds: selectedCategoryIds, tagIds: selectedTagIds,
      author, autoSummary, allowComments, pinned, visibility, metadata,
    };
    if (status === "published") payload.publishedAt = new Date().toISOString();
    return payload;
  };

  const handleSave = async (status: "draft" | "published") => {
    if (!slug.trim()) { setError("请填写 Slug"); return; }
    setSaving(true);
    setError(null);
    try {
      const payload = buildPayload(status);
      if (isEditing) await updateArticle(Number(id), payload as Partial<Article>);
      else await createArticle(payload as Partial<Article>);
      navigate("/admin/articles");
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
    } finally {
      setSaving(false);
    }
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
      {/* ═══ Sticky Header: Action Bar ═══ */}
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
            <button onClick={() => handleSave("draft")} disabled={saving}
              className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50">
              {saving ? "保存中..." : "草稿"}
            </button>
            <button onClick={() => handleSave("published")} disabled={saving}
              className="px-3 py-1.5 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50">
              {saving ? "发布中..." : "发布"}
            </button>
          </div>
        </div>

        {/* Settings Panels (slide down) */}
        {showBasicInfo && (
          <div className="px-4 py-3 border-t border-gray-100 bg-gray-50 space-y-3 max-h-80 overflow-y-auto">
            <div className="grid grid-cols-2 gap-3">
              <Field label="Slug" value={slug} onChange={setSlug} placeholder="article-url-slug" />
              <Field label="作者" value={author} onChange={setAuthor} placeholder="作者名" />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">封面图</label>
              <div className="flex items-center gap-2">
                <input type="text" value={coverImage} onChange={(e) => setCoverImage(e.target.value)}
                  className="flex-1 px-2 py-1.5 text-sm border border-gray-300 rounded-lg" placeholder="URL 或点击选择" />
                <button type="button" onClick={() => setShowCoverPicker(true)}
                  className="px-3 py-1.5 text-xs border border-gray-300 rounded-lg hover:bg-gray-100">选择</button>
              </div>
              {coverImage && <img src={coverImage} alt="封面" className="mt-1.5 max-h-20 rounded border border-gray-200"
                onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }} />}
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">分类</label>
              {categories.length === 0 ? <span className="text-xs text-gray-400">无分类</span> : (
                <div className="flex flex-wrap gap-1.5">
                  {categories.map((cat) => (
                    <button key={cat.id} type="button" onClick={() => toggleCategory(cat.id)}
                      className={`px-2.5 py-1 text-xs rounded-full border transition-colors ${
                        selectedCategoryIds.includes(cat.id) ? "bg-purple-100 border-purple-300 text-purple-800" : "bg-white border-gray-200 text-gray-600 hover:bg-gray-50"
                      }`}>{cat.zhName || cat.enName}</button>
                  ))}
                </div>
              )}
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">标签</label>
              {tags.length === 0 ? <span className="text-xs text-gray-400">无标签</span> : (
                <div className="flex flex-wrap gap-1.5">
                  {tags.map((tag) => (
                    <button key={tag.id} type="button" onClick={() => toggleTag(tag.id)}
                      className={`px-2.5 py-1 text-xs rounded-full border transition-colors ${
                        selectedTagIds.includes(tag.id) ? "bg-blue-100 border-blue-300 text-blue-800" : "bg-white border-gray-200 text-gray-600 hover:bg-gray-50"
                      }`}>{tag.zhName || tag.enName}</button>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {showSeo && (
          <div className="px-4 py-3 border-t border-gray-100 bg-gray-50 space-y-3 max-h-80 overflow-y-auto">
            <div className="grid grid-cols-2 gap-3">
              <Field label="中文 SEO 标题" value={zhSeoTitle} onChange={setZhSeoTitle} placeholder="SEO 标题" />
              <Field label="英文 SEO 标题" value={enSeoTitle} onChange={setEnSeoTitle} placeholder="SEO Title" />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">中文 Meta 描述</label>
                <textarea value={zhMetaDescription} onChange={(e) => setZhMetaDescription(e.target.value)} rows={2}
                  className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded-lg" placeholder="Meta 描述" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">英文 Meta 描述</label>
                <textarea value={enMetaDescription} onChange={(e) => setEnMetaDescription(e.target.value)} rows={2}
                  className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded-lg" placeholder="Meta Description" />
              </div>
            </div>
            <Field label="OG Image URL" value={ogImage} onChange={setOgImage} placeholder="https://..." />
          </div>
        )}

        {showAdvanced && (
          <div className="px-4 py-3 border-t border-gray-100 bg-gray-50 space-y-3 max-h-80 overflow-y-auto">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">可见性</label>
                <select value={visibility} onChange={(e) => setVisibility(e.target.value)}
                  className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded-lg">
                  <option value="public">公开</option>
                  <option value="private">私密</option>
                  <option value="password_protected">密码保护</option>
                </select>
              </div>
              <div className="flex items-end gap-4 pb-1">
                <CheckboxField label="自动摘要" checked={autoSummary} onChange={setAutoSummary} />
                <CheckboxField label="允许评论" checked={allowComments} onChange={setAllowComments} />
                <CheckboxField label="置顶" checked={pinned} onChange={setPinned} />
              </div>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">元数据</label>
              <MetadataEditor value={metadata} onChange={setMetadata} />
            </div>
          </div>
        )}

        {/* Row 2: Editor Toolbar (for active language editor) */}
        {activeEntry?.editor && (
          <EditorToolbar editor={activeEntry.editor} modals={activeEntry.modals} />
        )}
      </div>

      {/* ═══ Error Bar ═══ */}
      {error && (
        <div className="px-4 py-2 bg-red-50 border-b border-red-200 text-red-800 text-sm flex-shrink-0">
          {error}
          <button onClick={() => setError(null)} className="ml-2 text-red-600 hover:text-red-800">&times;</button>
        </div>
      )}

      {/* ═══ Main Content: Editor + Sidebar ═══ */}
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
          <div className="flex-1 min-h-0 overflow-y-auto border-t border-gray-300">
            {enabledLangs.map((lang, idx) => {
              const entry = langEditors[lang];
              if (!entry?.editor) return null;
              return (
                <div key={lang} className={idx === activeLangIdx ? "h-full" : "hidden"}>
                  <EditorBubbleMenu editor={entry.editor} />
                  <TableBubbleMenu editor={entry.editor} />
                  <EditorFloatingMenu editor={entry.editor} />
                  <EditorContent editor={entry.editor} className="h-full article-editor-content" />
                </div>
              );
            })}
          </div>
        </div>

        {/* Right: Sidebar — Outline + Details */}
        <EditorSidebar
          editor={activeEntry?.editor ?? null}
          article={sidebarArticle}
        />
      </div>

      {/* ═══ Modals ═══ */}
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

// ─── Sidebar: Outline + Details ───

interface HeadingItem {
  level: number;
  text: string;
  pos: number;
}

const EditorSidebar = memo(function EditorSidebar({ editor, article }: {
  editor: Editor | null;
  article: { slug: string; author: string; createdAt: string | null; publishedAt: string | null } | null;
}) {
  const [activeTab, setActiveTab] = useState<"outline" | "details">("outline");

  // Extract headings & stats reactively from editor state
  const { headings, charCount, wordCount } = useEditorState({
    editor,
    selector: ({ editor: e }) => {
      if (!e) return { headings: [] as HeadingItem[], charCount: 0, wordCount: 0 };
      const h: HeadingItem[] = [];
      e.state.doc.descendants((node, pos) => {
        if (node.type.name === "heading") {
          h.push({ level: node.attrs.level as number, text: node.textContent, pos });
        }
      });
      const text = e.state.doc.textContent;
      const chars = text.length;
      // Count words: CJK chars each count as one word; latin words split by spaces
      const cjk = (text.match(/[\u4e00-\u9fff\u3400-\u4dbf\uf900-\ufaff]/g) || []).length;
      const latin = text.replace(/[\u4e00-\u9fff\u3400-\u4dbf\uf900-\ufaff]/g, " ").trim().split(/\s+/).filter(Boolean).length;
      return { headings: h, charCount: chars, wordCount: cjk + latin };
    },
    equalityFn: (a, b) => {
      if (a.charCount !== b.charCount || a.wordCount !== b.wordCount) return false;
      if (a.headings.length !== b.headings.length) return false;
      return a.headings.every((h, i) =>
        h.level === b.headings[i].level && h.text === b.headings[i].text && h.pos === b.headings[i].pos
      );
    },
  });

  const scrollToHeading = (pos: number) => {
    if (!editor) return;
    editor.chain().focus().setTextSelection(pos).run();
    // Scroll the DOM node into view
    try {
      const dom = editor.view.domAtPos(pos);
      const el = dom.node instanceof HTMLElement ? dom.node : dom.node.parentElement;
      el?.scrollIntoView({ behavior: "smooth", block: "center" });
    } catch { /* ignore */ }
  };

  const formatDate = (d: string | null) => {
    if (!d) return "—";
    try { return new Date(d).toLocaleString("zh-CN"); } catch { return d; }
  };

  return (
    <div className="w-60 flex-shrink-0 border-l border-gray-200 bg-gray-50 flex flex-col min-h-0">
      {/* Tab switcher */}
      <div className="flex border-b border-gray-200 flex-shrink-0">
        <button
          onClick={() => setActiveTab("outline")}
          className={`flex-1 px-3 py-2 text-xs font-medium transition-colors ${
            activeTab === "outline" ? "text-blue-700 border-b-2 border-blue-600 bg-white" : "text-gray-500 hover:text-gray-700"
          }`}
        >
          大纲
        </button>
        <button
          onClick={() => setActiveTab("details")}
          className={`flex-1 px-3 py-2 text-xs font-medium transition-colors ${
            activeTab === "details" ? "text-blue-700 border-b-2 border-blue-600 bg-white" : "text-gray-500 hover:text-gray-700"
          }`}
        >
          详情
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-3">
        {activeTab === "outline" ? (
          headings.length === 0 ? (
            <p className="text-xs text-gray-400 italic">暂无标题</p>
          ) : (
            <nav className="space-y-0.5">
              {headings.map((h, i) => (
                <button
                  key={i}
                  onClick={() => scrollToHeading(h.pos)}
                  className="block w-full text-left text-xs py-1 px-1.5 rounded hover:bg-gray-200 text-gray-700 truncate transition-colors"
                  style={{ paddingLeft: `${(h.level - 1) * 12 + 6}px` }}
                  title={h.text}
                >
                  <span className="text-gray-400 mr-1">H{h.level}</span>
                  {h.text || <span className="text-gray-300 italic">空标题</span>}
                </button>
              ))}
            </nav>
          )
        ) : (
          <div className="space-y-3 text-xs">
            <DetailRow label="字符数" value={charCount.toLocaleString()} />
            <DetailRow label="词数" value={wordCount.toLocaleString()} />
            {article && (
              <>
                <DetailRow label="创建时间" value={formatDate(article.createdAt)} />
                <DetailRow label="发布时间" value={formatDate(article.publishedAt)} />
                {article.author && <DetailRow label="作者" value={article.author} />}
                {article.slug && (
                  <div>
                    <div className="text-gray-400 mb-0.5">访问链接</div>
                    <a
                      href={`/articles/${article.slug}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:underline break-all"
                    >
                      /articles/{article.slug}
                    </a>
                  </div>
                )}
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
});

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <span className="text-gray-400">{label}</span>
      <span className="text-gray-700 font-medium">{value}</span>
    </div>
  );
}

// ─── Tiny helper components ───

function PopoverButton({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button type="button" onClick={onClick}
      className={`px-2.5 py-1.5 text-xs rounded-lg border transition-colors ${
        active ? "bg-blue-50 border-blue-300 text-blue-700" : "border-gray-200 text-gray-600 hover:bg-gray-50"
      }`}>
      {label}
    </button>
  );
}

function Field({ label, value, onChange, placeholder }: { label: string; value: string; onChange: (v: string) => void; placeholder?: string }) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-600 mb-1">{label}</label>
      <input type="text" value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
        className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded-lg focus:ring-1 focus:ring-blue-500 focus:border-blue-500" />
    </div>
  );
}

function CheckboxField({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="flex items-center gap-1.5 text-xs text-gray-600 cursor-pointer">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} className="rounded border-gray-300" />
      {label}
    </label>
  );
}
