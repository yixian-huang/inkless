import { Suspense } from "react";
import ImagePickerModal from "@/components/admin/ImagePickerModal";
import type { Editor } from "@tiptap/react";
import type { ModalState } from "@/components/admin/editor/types-internal";
import { ArticleVersionHistoryPanel, type ArticleDraftSnapshot } from "../VersionHistoryPanel";
import ArticlePreviewModal, { type ArticlePreviewData } from "../ArticlePreviewModal";
import ArticleConflictDialog from "../ArticleConflictDialog";
import TemplatePickerModal from "../TemplatePickerModal";
import type { ArticleTemplate } from "../articleTemplates";
import { PublishChecklistDialog } from "./PublishChecklistDialog";
import type { ChecklistItem } from "../utils/publishChecklist";
import { LazyEditorModals } from "./lazyEditorSurfaces";
import { AIMetaPreviewDialog } from "./AIMetaPreviewDialog";
import type { ArticleMetaMode } from "@/api/ai";
import type { AIMetaApplyKey } from "../utils/applyAIMeta";
import type { QualityIssue } from "../utils/aiMetaQuality";

type LangEntry = {
  editor: Editor | null;
  state: ModalState;
};

/**
 * All editor page overlays / dialogs (version, preview, conflict, checklist, …).
 */
export function EditorDialogs({
  langEditors,
  showCoverPicker,
  onCloseCoverPicker,
  onSelectCover,
  showVersionHistory,
  isEditing,
  articleId,
  versionDraftSnapshot,
  onCloseVersionHistory,
  onRestoreVersion,
  showPreview,
  previewData,
  onClosePreview,
  showTemplatePicker,
  onCloseTemplatePicker,
  onSelectTemplate,
  conflict,
  saving,
  onDismissConflict,
  onReloadConflict,
  onForceOverwrite,
  publishChecklistOpen,
  publishChecklistItems,
  onCancelPublishChecklist,
  onForcePublish,
  onAIFillFromChecklist,
  aiMeta,
}: {
  langEditors: Record<string, LangEntry>;
  showCoverPicker: boolean;
  onCloseCoverPicker: () => void;
  onSelectCover: (url: string) => void;
  showVersionHistory: boolean;
  isEditing: boolean;
  articleId: string | undefined;
  versionDraftSnapshot: ArticleDraftSnapshot | null;
  onCloseVersionHistory: () => void;
  onRestoreVersion: (snap: ArticleDraftSnapshot) => void;
  showPreview: boolean;
  previewData: ArticlePreviewData | null;
  onClosePreview: () => void;
  showTemplatePicker: boolean;
  onCloseTemplatePicker: () => void;
  onSelectTemplate: (tpl: ArticleTemplate) => void;
  conflict: { serverUpdatedAt?: string } | null;
  saving: boolean;
  onDismissConflict: () => void;
  onReloadConflict: () => void;
  onForceOverwrite: () => void;
  publishChecklistOpen: boolean;
  publishChecklistItems: ChecklistItem[];
  onCancelPublishChecklist: () => void;
  onForcePublish?: () => void;
  onAIFillFromChecklist?: () => void;
  aiMeta?: {
    open: boolean;
    busy: boolean;
    mode: ArticleMetaMode;
    onModeChange: (m: ArticleMetaMode) => void;
    sourceLang: "zh" | "en";
    onSourceLangChange: (l: "zh" | "en") => void;
    values: Partial<Record<AIMetaApplyKey, string>>;
    selected: Set<AIMetaApplyKey>;
    onToggle: (k: AIMetaApplyKey) => void;
    skipped?: string[];
    titleIndex: number;
    titleCount: number;
    onCycleTitle: (delta: 1 | -1) => void;
    slugLocked: boolean;
    panelError: string | null;
    qualityIssues?: QualityIssue[];
    model?: string;
    onClose: () => void;
    onApply: () => void;
    onRegenerate: () => void;
    onFeedback?: (kind: "useful" | "needs_edit" | "unusable") => void;
  };
}) {
  return (
    <>
      {Object.entries(langEditors).map(([lang, entry]) =>
        entry.editor ? (
          <Suspense key={lang} fallback={null}>
            <LazyEditorModals editor={entry.editor} state={entry.state} />
          </Suspense>
        ) : null,
      )}

      <ImagePickerModal
        open={showCoverPicker}
        onClose={onCloseCoverPicker}
        onSelect={(item) => onSelectCover(item.url)}
      />

      {showVersionHistory && isEditing && articleId && (
        <ArticleVersionHistoryPanel
          articleId={Number(articleId)}
          onClose={onCloseVersionHistory}
          currentDraft={versionDraftSnapshot}
          onRestore={onRestoreVersion}
          canRestore
        />
      )}

      <ArticlePreviewModal
        open={showPreview}
        data={previewData}
        onClose={onClosePreview}
      />
      <TemplatePickerModal
        open={showTemplatePicker}
        onClose={onCloseTemplatePicker}
        onSelect={onSelectTemplate}
      />

      {conflict && (
        <ArticleConflictDialog
          serverUpdatedAt={conflict.serverUpdatedAt}
          busy={saving}
          onDismiss={onDismissConflict}
          onReload={onReloadConflict}
          onForceOverwrite={onForceOverwrite}
        />
      )}

      <PublishChecklistDialog
        open={publishChecklistOpen}
        items={publishChecklistItems}
        busy={saving}
        onCancel={onCancelPublishChecklist}
        onForcePublish={onForcePublish}
        onAIFill={onAIFillFromChecklist}
      />

      {aiMeta && (
        <AIMetaPreviewDialog
          open={aiMeta.open}
          busy={aiMeta.busy}
          mode={aiMeta.mode}
          onModeChange={aiMeta.onModeChange}
          sourceLang={aiMeta.sourceLang}
          onSourceLangChange={aiMeta.onSourceLangChange}
          values={aiMeta.values}
          selected={aiMeta.selected}
          onToggle={aiMeta.onToggle}
          skipped={aiMeta.skipped}
          titleIndex={aiMeta.titleIndex}
          titleCount={aiMeta.titleCount}
          onCycleTitle={aiMeta.onCycleTitle}
          slugLocked={aiMeta.slugLocked}
          panelError={aiMeta.panelError}
          qualityIssues={aiMeta.qualityIssues}
          model={aiMeta.model}
          onClose={aiMeta.onClose}
          onApply={aiMeta.onApply}
          onRegenerate={aiMeta.onRegenerate}
          onFeedback={aiMeta.onFeedback}
        />
      )}
    </>
  );
}
