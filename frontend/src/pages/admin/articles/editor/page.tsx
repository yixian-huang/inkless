import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import type { Category, Tag } from "@/api/articles";
import { getCategories, getTags } from "@/api/articles";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { useAuth } from "@/contexts/AuthContext";
import type { ArticleDraftSnapshot } from "./VersionHistoryPanel";
import type { ArticleTemplate } from "./articleTemplates";
import { LEAVE_UNSAVED_MESSAGE } from "./saveStatusUtils";
import { slugifyTitle } from "./utils/slugify";
import { statusLabelOf } from "./utils/constants";
import { toast } from "./utils/toast";
import { pickPayloadFields } from "./utils/buildArticlePayload";
import type { LocalDraftSnapshot } from "./utils/localDraft";
import { applyLocalDraftToFormAndEditors } from "./utils/applyLocalDraft";
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
import { useLocalDraft } from "./hooks/useLocalDraft";
import { useEditorShell } from "./hooks/useEditorShell";
import { usePublishGate } from "./hooks/usePublishGate";
import { useArticleAIMeta } from "./hooks/useArticleAIMeta";
import { EditorMessageBars } from "./components/EditorMessageBars";
import { EditorWorkspace } from "./components/EditorWorkspace";
import { LangEditorMount } from "./components/LangEditorMount";
import { LocalDraftBanner } from "./components/LocalDraftBanner";
import { EditorChrome } from "./components/EditorChrome";
import { EditorDialogs } from "./components/EditorDialogs";
import { ShortcutHelpModal } from "./components/ShortcutHelpModal";

export default function ArticleEditorPage() {
  useDocumentTitle("编辑文章");
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEditing = !!id;
  const { hasPermission } = useAuth();
  const canPublish = hasPermission("articles:publish");

  const form = useArticleFormState();
  const dirty = useDirtyState(!isEditing);
  const shell = useEditorShell();
  const editors = useArticleEditors({
    zhBody: form.zhBody,
    enBody: form.enBody,
    setZhBody: form.setZhBody,
    setEnBody: form.setEnBody,
    touch: dirty.touch,
  });

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

  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const langMenuRef = useRef<HTMLDivElement>(null);

  useOutsideClick(langMenuRef, shell.showLangMenu, () => shell.setShowLangMenu(false));
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

  const aiMeta = useArticleAIMeta({
    getBodies: () => editors.resolveBodies(),
    getTitles: () => ({ zhTitle: form.zhTitle, enTitle: form.enTitle }),
    getExistingMeta: () => ({
      slug: form.slug,
      zhSeoTitle: form.zhSeoTitle,
      enSeoTitle: form.enSeoTitle,
      zhMetaDescription: form.zhMetaDescription,
      enMetaDescription: form.enMetaDescription,
    }),
    getExistingTagNames: () =>
      tags
        .filter((t) => form.selectedTagIds.includes(t.id))
        .map((t) => t.zhName || t.enName || "")
        .filter(Boolean),
    slugLocked: form.articleStatus === "published",
    setters: {
      setZhTitle: form.setZhTitle,
      setEnTitle: form.setEnTitle,
      setSlug: form.setSlug,
      setZhSeoTitle: form.setZhSeoTitle,
      setEnSeoTitle: form.setEnSeoTitle,
      setZhMetaDescription: form.setZhMetaDescription,
      setEnMetaDescription: form.setEnMetaDescription,
    },
    ensureEnEnabled: editors.ensureEnEnabled,
    touch: dirty.touch,
    setError: persistence.setError,
    setSuccessMessage: persistence.setSuccessMessage,
  });

  const wordStats = useWordStats({
    editorMode: editors.editorMode,
    markdownZh: editors.markdownContent.zh ?? "",
    markdownEn: editors.markdownContent.en ?? "",
    zhBody: form.zhBody,
    enBody: form.enBody,
    zhEditor: editors.zhEditor,
    enEditor: editors.enEditor,
    includeEn: editors.enabledLangs.includes("en") || editors.viewLayout === "split",
    tick: `${dirty.isDirty}-${persistence.saving}`,
  });

  const openPreview = useCallback(() => {
    const bodies = editors.resolveBodies();
    const isEn = editors.activeLang === "en";
    const finalSlug = form.slug.trim() || slugifyTitle(form.zhTitle, form.enTitle);
    shell.openPreviewWith({
      title: isEn ? (form.enTitle || form.zhTitle) : (form.zhTitle || form.enTitle),
      bodyHtml: isEn ? (bodies.enBody || bodies.zhBody) : (bodies.zhBody || bodies.enBody),
      coverImage: form.coverImage || undefined,
      author: form.author || undefined,
      langLabel: isEn ? "English" : "中文",
      statusLabel: statusLabelOf(form.articleStatus, dirty.isDirty),
      publicPath:
        form.articleStatus === "published" && finalSlug ? `/blog/${finalSlug}` : null,
      metadata: form.metadata,
    });
  }, [editors, form, dirty.isDirty, shell]);

  const applyLocalDraft = useCallback((draft: LocalDraftSnapshot) => {
    dirty.pauseReady();
    applyLocalDraftToFormAndEditors(draft, form, editors);
    toast(persistence.setSuccessMessage, "已恢复本地草稿（尚未保存到服务器）", 4000);
    dirty.resumeReady();
    dirty.touch();
  }, [dirty, form, editors, persistence.setSuccessMessage]);

  const {
    offer: localDraftOffer,
    restore: restoreLocalDraft,
    dismiss: dismissLocalDraft,
    clear: clearLocalDraftCache,
  } = useLocalDraft({
    articleId: id,
    loading: persistence.loading,
    isDirty: dirty.isDirty,
    baseUpdatedAt: persistence.baseUpdatedAt,
    getSnapshot: () => {
      const bodies = editors.resolveBodies();
      return {
        baseUpdatedAt: persistence.baseUpdatedAt,
        editorMode: editors.editorMode,
        enabledLangs: editors.enabledLangs,
        zhTitle: form.zhTitle,
        enTitle: form.enTitle,
        slug: form.slug,
        coverImage: form.coverImage,
        zhBody: bodies.zhBody,
        enBody: bodies.enBody,
        zhSeoTitle: form.zhSeoTitle,
        enSeoTitle: form.enSeoTitle,
        zhMetaDescription: form.zhMetaDescription,
        enMetaDescription: form.enMetaDescription,
        ogImage: form.ogImage,
        author: form.author,
        markdownZh: editors.markdownContent.zh ?? "",
        markdownEn: editors.markdownContent.en ?? "",
      };
    },
    applySnapshot: applyLocalDraft,
    getServerCompare: () => ({
      zhTitle: form.zhTitle,
      enTitle: form.enTitle,
      zhBody: form.zhBody,
      enBody: form.enBody,
    }),
  });

  useEffect(() => {
    if (dirty.savePhase === "saved" && !dirty.isDirty) {
      clearLocalDraftCache();
    }
  }, [dirty.savePhase, dirty.isDirty, clearLocalDraftCache]);

  const collectPublishInput = useCallback(() => {
    const bodies = editors.resolveBodies();
    return {
      zhTitle: form.zhTitle,
      enTitle: form.enTitle,
      slug: form.slug,
      coverImage: form.coverImage,
      zhBody: bodies.zhBody,
      enBody: bodies.enBody,
      zhMetaDescription: form.zhMetaDescription,
      enMetaDescription: form.enMetaDescription,
      zhSeoTitle: form.zhSeoTitle,
      enSeoTitle: form.enSeoTitle,
      enabledLangs: editors.enabledLangs,
      author: form.author,
    };
  }, [editors, form]);

  const publishGate = usePublishGate({
    canPublish,
    collect: collectPublishInput,
    onPublish: () => void persistence.handleSave("publish"),
  });

  const handleOpenFind = useCallback(() => {
    if (editors.editorMode === "markdown") {
      editors.markdownApi?.openSearch?.();
      return;
    }
    shell.openFind();
  }, [editors.editorMode, editors.markdownApi, shell]);

  const handleExitOverlay = useCallback(() => {
    if (shell.showShortcutHelp) shell.closeShortcutHelp();
    else if (shell.findOpen) shell.closeFind();
    else if (shell.zenMode) shell.toggleZen();
  }, [shell]);

  useEditorShortcuts({
    canPublish,
    zenMode: shell.zenMode,
    findOpen: shell.findOpen,
    shortcutHelpOpen: shell.showShortcutHelp,
    onSave: (intent) => {
      if (intent === "publish") publishGate.requestPublish();
      else void persistence.handleSave(intent);
    },
    onPreview: openPreview,
    onFind: handleOpenFind,
    onToggleZen: shell.toggleZen,
    onToggleShortcutHelp: shell.toggleShortcutHelp,
    onExitOverlay: handleExitOverlay,
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
    shell.closeVersionHistory();
    toast(persistence.setSuccessMessage, "已恢复到所选版本（尚未保存，请检查后保存）", 4000);
    dirty.resumeReady();
    dirty.touch();
  }, [dirty, form, editors, shell, persistence.setSuccessMessage]);

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
    shell.setShowTemplatePicker(false);
    toast(
      persistence.setSuccessMessage,
      tpl.id === "blank" ? "已清空为空白文档" : `已应用模板「${tpl.name}」（未保存）`,
    );
    dirty.resumeReady();
    dirty.touch();
  }, [form, editors, dirty, shell, persistence.setSuccessMessage]);

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

  const setZhTitleTracked = useMemo(() => dirty.track(form.setZhTitle), [dirty, form.setZhTitle]);
  const setEnTitleTracked = useMemo(() => dirty.track(form.setEnTitle), [dirty, form.setEnTitle]);

  const langTitleMap = useMemo(
    () => ({
      zh: {
        title: form.zhTitle,
        setTitle: setZhTitleTracked,
        placeholder: "输入中文标题",
      },
      en: {
        title: form.enTitle,
        setTitle: setEnTitleTracked,
        placeholder: "Enter English title",
      },
    }),
    [form.zhTitle, form.enTitle, setZhTitleTracked, setEnTitleTracked],
  );

  const activeTitle = langTitleMap[editors.activeLang as "zh" | "en"];

  const onMarkdownChange = useCallback((lang: string, val: string) => {
    editors.setMarkdownContent((prev) => ({ ...prev, [lang]: val }));
    dirty.touch();
  }, [editors, dirty]);

  if (persistence.loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-600">加载中...</div>
      </div>
    );
  }

  return (
    <div
      className={`flex flex-col min-h-0 bg-white ${
        shell.zenMode ? "fixed inset-0 z-50 h-screen" : "h-full"
      }`}
    >
      <EditorChrome
        zenMode={shell.zenMode}
        title={activeTitle?.title || ""}
        titlePlaceholder={activeTitle?.placeholder || "标题"}
        onTitleChange={(v) => activeTitle?.setTitle(v)}
        onBack={handleBack}
        savePhase={dirty.savePhase}
        lastSavedAt={dirty.lastSavedAt}
        lastSaveWasAutosave={dirty.lastSaveWasAutosave}
        metaPanel={shell.metaPanel}
        onToggleMetaPanel={shell.toggleMetaPanel}
        showVersionHistory={shell.showVersionHistory}
        isEditing={isEditing}
        canPublish={canPublish}
        saving={persistence.saving}
        articleStatus={form.articleStatus}
        scheduledPublication={schedule.scheduledPublication}
        scheduleLoading={schedule.scheduleLoading}
        scheduleBusy={schedule.scheduleBusy}
        onToggleZen={shell.toggleZen}
        onOpenShortcutHelp={shell.openShortcutHelp}
        onOpenHistory={() => {
          shell.openVersionHistory(
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
        }}
        onOpenTemplate={() => shell.setShowTemplatePicker(true)}
        onPreview={openPreview}
        onFind={handleOpenFind}
        onOpenAIMeta={() => void aiMeta.openPreview()}
        aiMetaBusy={aiMeta.busy}
        onSave={() => void persistence.handleSave("draft")}
        onPublish={() => publishGate.requestPublish()}
        onSchedule={schedule.handleSchedulePublish}
        onCancelSchedule={schedule.handleCancelSchedule}
        onRetrySchedule={schedule.handleRetrySchedule}
        onRefreshSchedule={schedule.loadArticleSchedule}
        meta={{
          slug: form.slug,
          setSlug: dirty.track(form.setSlug),
          author: form.author,
          setAuthor: dirty.track(form.setAuthor),
          coverImage: form.coverImage,
          setCoverImage: dirty.track(form.setCoverImage),
          showCoverPicker: shell.showCoverPicker,
          setShowCoverPicker: shell.setShowCoverPicker,
          categories,
          selectedCategoryIds: form.selectedCategoryIds,
          onToggleCategory: (cid) => {
            form.toggleCategory(cid);
            dirty.touch();
          },
          tags,
          selectedTagIds: form.selectedTagIds,
          onToggleTag: (tid) => {
            form.toggleTag(tid);
            dirty.touch();
          },
          zhSeoTitle: form.zhSeoTitle,
          setZhSeoTitle: dirty.track(form.setZhSeoTitle),
          enSeoTitle: form.enSeoTitle,
          setEnSeoTitle: dirty.track(form.setEnSeoTitle),
          zhMetaDescription: form.zhMetaDescription,
          setZhMetaDescription: dirty.track(form.setZhMetaDescription),
          enMetaDescription: form.enMetaDescription,
          setEnMetaDescription: dirty.track(form.setEnMetaDescription),
          ogImage: form.ogImage,
          setOgImage: dirty.track(form.setOgImage),
          visibility: form.visibility,
          setVisibility: dirty.track(form.setVisibility),
          autoSummary: form.autoSummary,
          setAutoSummary: dirty.track(form.setAutoSummary),
          allowComments: form.allowComments,
          setAllowComments: dirty.track(form.setAllowComments),
          pinned: form.pinned,
          setPinned: dirty.track(form.setPinned),
          metadata: form.metadata,
          setMetadata: dirty.track(form.setMetadata),
        }}
        lang={{
          enabledLangs: editors.enabledLangs,
          activeLangIdx: editors.activeLangIdx,
          viewLayout: editors.viewLayout,
          wordStats,
          editorMode: editors.editorMode,
          translateBusy: bilingual.translateBusy,
          showLangMenu: shell.showLangMenu,
          langMenuRef,
          onSelectLang: editors.setActiveLangIdx,
          onRemoveLang: editors.removeLang,
          onAddLang: (key) => {
            editors.addLang(key);
            shell.setShowLangMenu(false);
          },
          onToggleLangMenu: shell.toggleLangMenu,
          onToggleSplit: () =>
            editors.setViewLayout((v) => (v === "split" ? "focus" : "split")),
          onCopyZhToEn: () => bilingual.handleCopyToOtherLang("zh"),
          onTranslateZhToEn: () => void bilingual.handleTranslateToOtherLang("zh"),
          onModeChange: editors.handleModeChange,
        }}
        toolbar={{
          activeEditorEntry: editors.activeEntry,
          markdownApi: editors.markdownApi,
          findOpen: shell.findOpen,
          onCloseFind: shell.closeFind,
        }}
      />

      {localDraftOffer && (
        <LocalDraftBanner
          offer={localDraftOffer}
          onRestore={restoreLocalDraft}
          onDismiss={dismissLocalDraft}
        />
      )}

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
        showOutline
        outlineCompact={shell.zenMode}
        onSelectLangKey={editors.selectLangKey}
        onMarkdownChange={onMarkdownChange}
        onMarkdownApiReady={editors.setMarkdownApi}
        onJumpMarkdownLine={(line) => editors.markdownApi?.gotoLine?.(line)}
      />

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

      <EditorDialogs
        langEditors={editors.langEditors}
        showCoverPicker={shell.showCoverPicker}
        onCloseCoverPicker={() => shell.setShowCoverPicker(false)}
        onSelectCover={(url) => {
          form.setCoverImage(url);
          shell.setShowCoverPicker(false);
          dirty.touch();
        }}
        showVersionHistory={shell.showVersionHistory}
        isEditing={isEditing}
        articleId={id}
        versionDraftSnapshot={shell.versionDraftSnapshot}
        onCloseVersionHistory={shell.closeVersionHistory}
        onRestoreVersion={handleRestoreVersion}
        showPreview={shell.showPreview}
        previewData={shell.previewData}
        onClosePreview={shell.closePreview}
        showTemplatePicker={shell.showTemplatePicker}
        onCloseTemplatePicker={() => shell.setShowTemplatePicker(false)}
        onSelectTemplate={handleApplyTemplate}
        conflict={persistence.conflict}
        saving={persistence.saving}
        onDismissConflict={() => persistence.setConflict(null)}
        onReloadConflict={persistence.forceReload}
        onForceOverwrite={() => {
          persistence.setConflict(null);
          void persistence.handleSave("draft", { force: true });
        }}
        publishChecklistOpen={publishGate.open}
        publishChecklistItems={publishGate.items || []}
        onCancelPublishChecklist={publishGate.dismiss}
        onForcePublish={
          publishGate.hasBlocks
            ? undefined
            : () => publishGate.requestPublish({ force: true })
        }
        onAIFillFromChecklist={() => {
          publishGate.dismiss();
          void aiMeta.openPreview({ mode: "fill_empty" });
        }}
        aiMeta={{
          open: aiMeta.open,
          busy: aiMeta.busy,
          mode: aiMeta.mode,
          onModeChange: aiMeta.setMode,
          sourceLang: aiMeta.sourceLang,
          onSourceLangChange: aiMeta.setSourceLang,
          values: aiMeta.values,
          selected: aiMeta.selected,
          onToggle: aiMeta.toggleKey,
          skipped: aiMeta.response?.skipped,
          titleIndex: aiMeta.titleIndex,
          titleCount: aiMeta.titleCount,
          onCycleTitle: aiMeta.cycleTitle,
          slugLocked: aiMeta.slugLocked,
          panelError: aiMeta.panelError,
          qualityIssues: aiMeta.qualityIssues,
          model: aiMeta.response?.model,
          onClose: aiMeta.close,
          onApply: aiMeta.apply,
          onRegenerate: () =>
            void aiMeta.openPreview({
              mode: aiMeta.mode,
              sourceLang: aiMeta.sourceLang,
            }),
          onFeedback: aiMeta.feedback,
        }}
      />

      <ShortcutHelpModal
        open={shell.showShortcutHelp}
        onClose={shell.closeShortcutHelp}
      />
    </div>
  );
}
