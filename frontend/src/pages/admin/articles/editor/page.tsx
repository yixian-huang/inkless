import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import type { Category, Tag } from "@/api/articles";
import { getCategories, getTags } from "@/api/articles";
import ImagePickerModal from "@/components/admin/ImagePickerModal";
import { EditorToolbar, EditorModals } from "@/components/admin/RichTextEditor";
import MarkdownToolbar from "@/components/admin/editor/MarkdownToolbar";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { useAuth } from "@/contexts/AuthContext";
import ArticleForm from "./ArticleForm";
import { SeoFieldsPanel, AdvancedSettingsPanel } from "./SeoFields";
import { ArticleVersionHistoryPanel, type ArticleDraftSnapshot } from "./VersionHistoryPanel";
import ArticlePreviewModal, { type ArticlePreviewData } from "./ArticlePreviewModal";
import ArticleConflictDialog from "./ArticleConflictDialog";
import TemplatePickerModal from "./TemplatePickerModal";
import type { ArticleTemplate } from "./articleTemplates";
import { LEAVE_UNSAVED_MESSAGE } from "./saveStatusUtils";
import { slugifyTitle } from "./utils/slugify";
import { statusLabelOf } from "./utils/constants";
import { toast } from "./utils/toast";
import { pickPayloadFields } from "./utils/buildArticlePayload";
import { useDirtyState } from "./hooks/useDirtyState";
import { useArticleFormState } from "./hooks/useArticleFormState";
import {
  useArticlePersistence,
  type ArticleSaveSource,
} from "./hooks/useArticlePersistence";
import { useArticleEditors } from "./hooks/useArticleEditors";
import { useBilingualActions } from "./hooks/useBilingualActions";
import { useEditorShortcuts } from "./hooks/useEditorShortcuts";
import { useOutsideClick } from "./hooks/useOutsideClick";
import { useSlashMediaBridge } from "./hooks/useSlashMediaBridge";
import { useArticleSchedule } from "./hooks/useArticleSchedule";
import { useWordStats } from "./hooks/useWordStats";
import { EditorMessageBars } from "./components/EditorMessageBars";
import { EditorLangBar } from "./components/EditorLangBar";
import { EditorWorkspace } from "./components/EditorWorkspace";
import { EditorActionBar } from "./components/EditorActionBar";
import { LangEditorMount } from "./components/LangEditorMount";

export default function ArticleEditorPage() {
  useDocumentTitle("编辑文章");
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEditing = !!id;
  const { hasPermission } = useAuth();
  const canPublish = hasPermission("articles:publish");

  // ── Domain state ──
  const form = useArticleFormState();
  const dirty = useDirtyState(!isEditing);
  const editors = useArticleEditors({
    zhBody: form.zhBody,
    enBody: form.enBody,
    setZhBody: form.setZhBody,
    setEnBody: form.setEnBody,
    isDirty: dirty.isDirty,
    touch: dirty.touch,
  });

  // Live save source — mutated each render, read only inside save/schedule handlers
  const saveSourceRef = useRef<ArticleSaveSource>({
    getFields: () => pickPayloadFields(form),
    resolveBodies: () => editors.resolveBodies(),
    getArticleStatus: () => form.articleStatus,
    setSlug: form.setSlug,
    setArticleStatus: form.setArticleStatus,
  });
  saveSourceRef.current = {
    getFields: () => pickPayloadFields(form),
    resolveBodies: () => editors.resolveBodies(),
    getArticleStatus: () => form.articleStatus,
    setSlug: form.setSlug,
    setArticleStatus: form.setArticleStatus,
  };

  // ── Shell UI ──
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [showBasicInfo, setShowBasicInfo] = useState(false);
  const [showSeo, setShowSeo] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showCoverPicker, setShowCoverPicker] = useState(false);
  const [showVersionHistory, setShowVersionHistory] = useState(false);
  const [versionDraftSnapshot, setVersionDraftSnapshot] = useState<ArticleDraftSnapshot | null>(null);
  const [showPreview, setShowPreview] = useState(false);
  const [previewData, setPreviewData] = useState<ArticlePreviewData | null>(null);
  const [showTemplatePicker, setShowTemplatePicker] = useState(false);
  const [showLangMenu, setShowLangMenu] = useState(false);
  const langMenuRef = useRef<HTMLDivElement>(null);

  useOutsideClick(langMenuRef, showLangMenu, () => setShowLangMenu(false));
  useSlashMediaBridge(editors.activeEntry?.state);

  useEffect(() => {
    void (async () => {
      try {
        const [cats, tgs] = await Promise.all([getCategories(), getTags()]);
        setCategories(cats || []);
        setTags(tgs || []);
      } catch { /* non-critical */ }
    })();
  }, []);

  const persistence = useArticlePersistence({
    id,
    isEditing,
    sourceRef: saveSourceRef,
    hydrateFromArticle: (article) => {
      form.hydrateFromArticle(article);
      editors.resetMarkdownOnLoad(!!(article.enBody || article.enTitle));
    },
    navigate,
    dirty,
  });

  const schedule = useArticleSchedule({
    id,
    isEditing,
    canPublish,
    sourceRef: saveSourceRef,
    buildPayload: persistence.buildPayload,
    navigate,
    setError: persistence.setError,
  });

  const bilingual = useBilingualActions({
    editorMode: editors.editorMode,
    markdownContent: editors.markdownContent,
    setMarkdownContent: editors.setMarkdownContent,
    zhEditor: editors.zhEditor,
    enEditor: editors.enEditor,
    zhBody: form.zhBody,
    enBody: form.enBody,
    setZhBody: form.setZhBody,
    setEnBody: form.setEnBody,
    zhTitle: form.zhTitle,
    enTitle: form.enTitle,
    setZhTitle: form.setZhTitle,
    setEnTitle: form.setEnTitle,
    enabledLangs: editors.enabledLangs,
    ensureEnEnabled: editors.ensureEnEnabled,
    touch: dirty.touch,
    setError: persistence.setError,
    setSuccessMessage: persistence.setSuccessMessage,
  });

  const wordStats = useWordStats({
    editorMode: editors.editorMode,
    markdownContent: editors.markdownContent,
    zhBody: form.zhBody,
    enBody: form.enBody,
    zhEditor: editors.zhEditor,
    enEditor: editors.enEditor,
    tick: `${dirty.isDirty}-${persistence.saving}`,
  });

  const openPreview = useCallback(() => {
    const bodies = editors.resolveBodies();
    const title =
      editors.activeLang === "en"
        ? (form.enTitle || form.zhTitle)
        : (form.zhTitle || form.enTitle);
    const bodyHtml =
      editors.activeLang === "en"
        ? (bodies.enBody || bodies.zhBody)
        : (bodies.zhBody || bodies.enBody);
    const finalSlug = form.slug.trim() || slugifyTitle(form.zhTitle);
    setPreviewData({
      title,
      bodyHtml,
      coverImage: form.coverImage || undefined,
      author: form.author || undefined,
      langLabel: editors.activeLang === "en" ? "English" : "中文",
      statusLabel: statusLabelOf(form.articleStatus, dirty.isDirty),
      publicPath:
        form.articleStatus === "published" && finalSlug ? `/blog/${finalSlug}` : null,
      metadata: form.metadata,
    });
    setShowPreview(true);
  }, [editors, form, dirty.isDirty]);

  useEditorShortcuts({
    canPublish,
    onSave: (intent) => void persistence.handleSave(intent),
    onPreview: openPreview,
  });

  const { confirmLeave } = useUnsavedChangesGuard(dirty.isDirty, LEAVE_UNSAVED_MESSAGE);
  const handleBack = useCallback(() => {
    if (!confirmLeave()) return;
    navigate("/admin/articles");
  }, [confirmLeave, navigate]);

  const handleRestoreVersion = useCallback((snapshot: ArticleDraftSnapshot) => {
    dirty.pauseReady();
    const bodies = form.hydrateFromSnapshot(snapshot, {
      slug: form.slug,
      author: form.author,
    });
    editors.applyBodiesToEditors(bodies.zhBody, bodies.enBody);
    setShowVersionHistory(false);
    toast(persistence.setSuccessMessage, "已恢复到所选版本（尚未保存，请检查后保存）", 4000);
    dirty.resumeReady();
    dirty.touch();
  }, [dirty, form, editors, persistence.setSuccessMessage]);

  const handleApplyTemplate = useCallback((tpl: ArticleTemplate) => {
    if (tpl.id !== "blank") {
      const hasContent =
        form.zhTitle.trim()
        || form.enTitle.trim()
        || (editors.zhEditor?.getText() || "").trim()
        || (editors.enEditor?.getText() || "").trim()
        || (editors.markdownContent.zh || "").trim()
        || (editors.markdownContent.en || "").trim();
      if (
        hasContent
        && !window.confirm(`应用模板「${tpl.name}」将覆盖当前标题与正文，是否继续？`)
      ) {
        return;
      }
    }
    dirty.pauseReady();
    if (tpl.zhTitle) form.setZhTitle(tpl.zhTitle);
    if (tpl.enTitle) form.setEnTitle(tpl.enTitle);
    editors.applyBodiesToEditors(tpl.zhBody || "<p></p>", tpl.enBody || "<p></p>");
    if (tpl.enTitle || tpl.enBody) editors.ensureEnEnabled();
    setShowTemplatePicker(false);
    toast(
      persistence.setSuccessMessage,
      tpl.id === "blank" ? "已清空为空白文档" : `已应用模板「${tpl.name}」（未保存）`,
    );
    dirty.resumeReady();
    dirty.touch();
  }, [form, editors, dirty, persistence.setSuccessMessage]);

  const sidebarArticle = useMemo(
    () =>
      isEditing
        ? {
            slug: form.slug,
            author: form.author,
            createdAt: form.articleCreatedAt,
            publishedAt: form.articlePublishedAt,
          }
        : null,
    [isEditing, form.slug, form.author, form.articleCreatedAt, form.articlePublishedAt],
  );

  const langTitleMap = useMemo(
    () => ({
      zh: {
        title: form.zhTitle,
        setTitle: dirty.track(form.setZhTitle),
        placeholder: "输入中文标题",
      },
      en: {
        title: form.enTitle,
        setTitle: dirty.track(form.setEnTitle),
        placeholder: "Enter English title",
      },
    }),
    [form.zhTitle, form.enTitle, form.setZhTitle, form.setEnTitle, dirty],
  );

  const activeTitle = langTitleMap[editors.activeLang as "zh" | "en"];

  if (persistence.loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-600">加载中...</div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-h-0 bg-white">
      <div className="flex-shrink-0 z-20 bg-white border-b border-slate-200 shadow-sm">
        <EditorActionBar
          title={activeTitle?.title || ""}
          titlePlaceholder={activeTitle?.placeholder || "标题"}
          onTitleChange={(v) => activeTitle?.setTitle(v)}
          onBack={handleBack}
          savePhase={dirty.savePhase}
          lastSavedAt={dirty.lastSavedAt}
          lastSaveWasAutosave={dirty.lastSaveWasAutosave}
          showBasicInfo={showBasicInfo}
          showSeo={showSeo}
          showAdvanced={showAdvanced}
          showVersionHistory={showVersionHistory}
          isEditing={isEditing}
          canPublish={canPublish}
          saving={persistence.saving}
          articleStatus={form.articleStatus}
          scheduledPublication={schedule.scheduledPublication}
          scheduleLoading={schedule.scheduleLoading}
          scheduleBusy={schedule.scheduleBusy}
          onToggleBasic={() => {
            setShowBasicInfo((v) => !v);
            setShowSeo(false);
            setShowAdvanced(false);
          }}
          onToggleSeo={() => {
            setShowSeo((v) => !v);
            setShowBasicInfo(false);
            setShowAdvanced(false);
          }}
          onToggleAdvanced={() => {
            setShowAdvanced((v) => !v);
            setShowBasicInfo(false);
            setShowSeo(false);
          }}
          onOpenHistory={() => {
            setVersionDraftSnapshot(
              editors.buildDraftSnapshot({
                zhTitle: form.zhTitle,
                enTitle: form.enTitle,
                slug: form.slug,
                articleStatus: form.articleStatus,
                coverImage: form.coverImage,
                zhSeoTitle: form.zhSeoTitle,
                enSeoTitle: form.enSeoTitle,
                zhMetaDescription: form.zhMetaDescription,
                enMetaDescription: form.enMetaDescription,
                ogImage: form.ogImage,
                author: form.author,
              }),
            );
            setShowVersionHistory(true);
          }}
          onOpenTemplate={() => setShowTemplatePicker(true)}
          onPreview={openPreview}
          onSave={() => void persistence.handleSave("draft")}
          onPublish={() => void persistence.handleSave("publish")}
          onSchedule={schedule.handleSchedulePublish}
          onCancelSchedule={schedule.handleCancelSchedule}
          onRetrySchedule={schedule.handleRetrySchedule}
          onRefreshSchedule={schedule.loadArticleSchedule}
        />

        {showBasicInfo && (
          <ArticleForm
            slug={form.slug}
            setSlug={dirty.track(form.setSlug)}
            author={form.author}
            setAuthor={dirty.track(form.setAuthor)}
            coverImage={form.coverImage}
            setCoverImage={dirty.track(form.setCoverImage)}
            showCoverPicker={showCoverPicker}
            setShowCoverPicker={setShowCoverPicker}
            categories={categories}
            selectedCategoryIds={form.selectedCategoryIds}
            toggleCategory={(cid) => {
              form.toggleCategory(cid);
              dirty.touch();
            }}
            tags={tags}
            selectedTagIds={form.selectedTagIds}
            toggleTag={(tid) => {
              form.toggleTag(tid);
              dirty.touch();
            }}
          />
        )}
        {showSeo && (
          <SeoFieldsPanel
            zhSeoTitle={form.zhSeoTitle}
            setZhSeoTitle={dirty.track(form.setZhSeoTitle)}
            enSeoTitle={form.enSeoTitle}
            setEnSeoTitle={dirty.track(form.setEnSeoTitle)}
            zhMetaDescription={form.zhMetaDescription}
            setZhMetaDescription={dirty.track(form.setZhMetaDescription)}
            enMetaDescription={form.enMetaDescription}
            setEnMetaDescription={dirty.track(form.setEnMetaDescription)}
            ogImage={form.ogImage}
            setOgImage={dirty.track(form.setOgImage)}
          />
        )}
        {showAdvanced && (
          <AdvancedSettingsPanel
            visibility={form.visibility}
            setVisibility={dirty.track(form.setVisibility)}
            autoSummary={form.autoSummary}
            setAutoSummary={dirty.track(form.setAutoSummary)}
            allowComments={form.allowComments}
            setAllowComments={dirty.track(form.setAllowComments)}
            pinned={form.pinned}
            setPinned={dirty.track(form.setPinned)}
            metadata={form.metadata}
            setMetadata={dirty.track(form.setMetadata)}
          />
        )}

        <EditorLangBar
          enabledLangs={editors.enabledLangs}
          activeLangIdx={editors.activeLangIdx}
          viewLayout={editors.viewLayout}
          wordStats={wordStats}
          editorMode={editors.editorMode}
          translateBusy={bilingual.translateBusy}
          showLangMenu={showLangMenu}
          langMenuRef={langMenuRef}
          onSelectLang={editors.setActiveLangIdx}
          onRemoveLang={editors.removeLang}
          onAddLang={(key) => {
            editors.addLang(key);
            setShowLangMenu(false);
          }}
          onToggleLangMenu={() => setShowLangMenu((v) => !v)}
          onToggleSplit={() =>
            editors.setViewLayout((v) => (v === "split" ? "focus" : "split"))
          }
          onCopyZhToEn={() => bilingual.handleCopyToOtherLang("zh")}
          onTranslateZhToEn={() => void bilingual.handleTranslateToOtherLang("zh")}
          onModeChange={editors.handleModeChange}
        />

        <div className="flex items-stretch border-t border-slate-200 bg-slate-50">
          {editors.editorMode === "richtext" && editors.activeEntry?.editor ? (
            <div className="flex-1 min-w-0 overflow-x-auto">
              <EditorToolbar
                editor={editors.activeEntry.editor}
                modals={editors.activeEntry.modals}
              />
            </div>
          ) : editors.editorMode === "markdown" ? (
            <div className="flex-1 min-w-0 overflow-x-auto">
              <MarkdownToolbar api={editors.markdownApi} />
            </div>
          ) : (
            <div className="flex-1 py-2" />
          )}
        </div>
      </div>

      <EditorMessageBars
        error={persistence.error}
        onClearError={() => persistence.setError(null)}
        successMessage={persistence.successMessage}
        scheduleMessage={schedule.scheduleMessage}
        onClearSuccess={() => {
          schedule.setScheduleMessage("");
          persistence.setSuccessMessage("");
        }}
      />

      <EditorWorkspace
        viewLayout={editors.viewLayout}
        editorMode={editors.editorMode}
        enabledLangs={editors.enabledLangs}
        activeLang={editors.activeLang}
        activeLangIdx={editors.activeLangIdx}
        langEditors={editors.langEditors}
        langTitleMap={langTitleMap}
        wordStats={wordStats}
        markdownContent={editors.markdownContent}
        metadata={form.metadata}
        sidebarArticle={sidebarArticle}
        onSelectLangKey={editors.selectLangKey}
        onMarkdownChange={(lang, val) => {
          editors.setMarkdownContent((prev) => ({ ...prev, [lang]: val }));
          dirty.touch();
        }}
        onMarkdownApiReady={editors.setMarkdownApi}
      />

      {/* TipTap instances: ZH always; EN only when bilingual is active */}
      <LangEditorMount
        enabled={editors.needZhEditor}
        html={form.zhBody}
        editable={editors.zhEditable}
        onDirty={dirty.touch}
        onEditor={editors.onZhEditorReady}
        onFlushBody={editors.onZhFlushBody}
      />
      <LangEditorMount
        enabled={editors.needEnEditor}
        html={form.enBody}
        editable={editors.enEditable}
        onDirty={dirty.touch}
        onEditor={editors.onEnEditorReady}
        onFlushBody={editors.onEnFlushBody}
      />

      {Object.entries(editors.langEditors).map(([lang, entry]) =>
        entry.editor ? (
          <EditorModals key={lang} editor={entry.editor} state={entry.state} />
        ) : null,
      )}

      <ImagePickerModal
        open={showCoverPicker}
        onClose={() => setShowCoverPicker(false)}
        onSelect={(item) => {
          form.setCoverImage(item.url);
          setShowCoverPicker(false);
          dirty.touch();
        }}
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
      <TemplatePickerModal
        open={showTemplatePicker}
        onClose={() => setShowTemplatePicker(false)}
        onSelect={handleApplyTemplate}
      />

      {persistence.conflict && (
        <ArticleConflictDialog
          serverUpdatedAt={persistence.conflict.serverUpdatedAt}
          busy={persistence.saving}
          onDismiss={() => persistence.setConflict(null)}
          onReload={persistence.forceReload}
          onForceOverwrite={() => {
            persistence.setConflict(null);
            void persistence.handleSave("draft", { force: true });
          }}
        />
      )}
    </div>
  );
}
