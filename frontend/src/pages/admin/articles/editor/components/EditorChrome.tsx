import type { RefObject } from "react";
import type { ScheduledPublication } from "@/api/scheduledPublications";
import type { Editor } from "@tiptap/react";
import { EditorToolbar, type ModalControls } from "@/components/admin/RichTextEditor";
import MarkdownToolbar from "@/components/admin/editor/MarkdownToolbar";
import type { MarkdownSelectionApi } from "@/components/admin/editor/MarkdownToolbar";
import type { EditorSavePhase } from "../saveStatusUtils";
import type { EditorMetaPanel } from "../hooks/useEditorShell";
import { EditorActionBar } from "./EditorActionBar";
import { EditorLangBar } from "./EditorLangBar";
import { FindReplaceBar } from "./FindReplaceBar";
import { ChromeMetaPanels, type ChromeMetaFormProps } from "./ChromeMetaPanels";

type WordStat = { chars: number; words: number };

type LangEntry = {
  editor: Editor | null;
  modals: ModalControls;
};

export type ChromeActionProps = {
  zenMode: boolean;
  title: string;
  titlePlaceholder: string;
  onTitleChange: (v: string) => void;
  onBack: () => void;
  savePhase: EditorSavePhase;
  lastSavedAt: Date | null;
  lastSaveWasAutosave: boolean;
  metaPanel: EditorMetaPanel;
  onToggleMetaPanel: (p: Exclude<EditorMetaPanel, null>) => void;
  showVersionHistory: boolean;
  isEditing: boolean;
  canPublish: boolean;
  saving: boolean;
  articleStatus: "draft" | "published" | "scheduled";
  scheduledPublication: ScheduledPublication | null;
  scheduleLoading: boolean;
  scheduleBusy: boolean;
  onToggleZen: () => void;
  onOpenHistory: () => void;
  onOpenTemplate: () => void;
  onPreview: () => void;
  onFind: () => void;
  onSave: () => void;
  onPublish: () => void;
  onSchedule: (at: string) => void;
  onCancelSchedule: () => void;
  onRetrySchedule: () => void;
  onRefreshSchedule: () => void;
};

export type ChromeLangProps = {
  enabledLangs: string[];
  activeLangIdx: number;
  viewLayout: "focus" | "split";
  wordStats: Record<"zh" | "en", WordStat>;
  editorMode: "richtext" | "markdown";
  translateBusy: boolean;
  showLangMenu: boolean;
  langMenuRef: RefObject<HTMLDivElement | null>;
  onSelectLang: (idx: number) => void;
  onRemoveLang: (key: string) => void;
  onAddLang: (key: string) => void;
  onToggleLangMenu: () => void;
  onToggleSplit: () => void;
  onCopyZhToEn: () => void;
  onTranslateZhToEn: () => void;
  onModeChange: (mode: "richtext" | "markdown") => void;
};

export type ChromeToolbarProps = {
  activeEditorEntry: LangEntry | undefined;
  markdownApi: MarkdownSelectionApi | null;
  findOpen: boolean;
  onCloseFind: () => void;
};

export type EditorChromeProps = ChromeActionProps & {
  meta: Omit<ChromeMetaFormProps, "metaPanel" | "zenMode">;
  lang: ChromeLangProps;
  toolbar: ChromeToolbarProps;
};

/**
 * Sticky top chrome: action bar, meta panels, lang bar, toolbars, find bar.
 * Props are grouped (action / meta / lang / toolbar) to keep the page call-site readable.
 */
export function EditorChrome({
  meta,
  lang,
  toolbar,
  ...action
}: EditorChromeProps) {
  const { zenMode, metaPanel, onToggleMetaPanel } = action;
  const { editorMode } = lang;
  const { activeEditorEntry, markdownApi, findOpen, onCloseFind } = toolbar;

  return (
    <div className="flex-shrink-0 z-20 bg-white border-b border-slate-200 shadow-sm">
      <EditorActionBar
        title={action.title}
        titlePlaceholder={action.titlePlaceholder}
        onTitleChange={action.onTitleChange}
        onBack={action.onBack}
        savePhase={action.savePhase}
        lastSavedAt={action.lastSavedAt}
        lastSaveWasAutosave={action.lastSaveWasAutosave}
        showBasicInfo={metaPanel === "basic"}
        showSeo={metaPanel === "seo"}
        showAdvanced={metaPanel === "advanced"}
        showVersionHistory={action.showVersionHistory}
        isEditing={action.isEditing}
        canPublish={action.canPublish}
        saving={action.saving}
        articleStatus={action.articleStatus}
        scheduledPublication={action.scheduledPublication}
        scheduleLoading={action.scheduleLoading}
        scheduleBusy={action.scheduleBusy}
        zenMode={zenMode}
        onToggleZen={action.onToggleZen}
        onToggleBasic={() => onToggleMetaPanel("basic")}
        onToggleSeo={() => onToggleMetaPanel("seo")}
        onToggleAdvanced={() => onToggleMetaPanel("advanced")}
        onOpenHistory={action.onOpenHistory}
        onOpenTemplate={action.onOpenTemplate}
        onPreview={action.onPreview}
        onFind={action.onFind}
        onSave={action.onSave}
        onPublish={action.onPublish}
        onSchedule={action.onSchedule}
        onCancelSchedule={action.onCancelSchedule}
        onRetrySchedule={action.onRetrySchedule}
        onRefreshSchedule={action.onRefreshSchedule}
      />

      <ChromeMetaPanels metaPanel={metaPanel} zenMode={zenMode} {...meta} />

      {!zenMode && (
        <EditorLangBar
          enabledLangs={lang.enabledLangs}
          activeLangIdx={lang.activeLangIdx}
          viewLayout={lang.viewLayout}
          wordStats={lang.wordStats}
          editorMode={editorMode}
          translateBusy={lang.translateBusy}
          showLangMenu={lang.showLangMenu}
          langMenuRef={lang.langMenuRef}
          onSelectLang={lang.onSelectLang}
          onRemoveLang={lang.onRemoveLang}
          onAddLang={lang.onAddLang}
          onToggleLangMenu={lang.onToggleLangMenu}
          onToggleSplit={lang.onToggleSplit}
          onCopyZhToEn={lang.onCopyZhToEn}
          onTranslateZhToEn={lang.onTranslateZhToEn}
          onModeChange={lang.onModeChange}
        />
      )}

      {!zenMode && (
        <div className="flex items-stretch border-t border-slate-200 bg-slate-50">
          {editorMode === "richtext" && activeEditorEntry?.editor ? (
            <div className="flex-1 min-w-0 overflow-x-auto">
              <EditorToolbar
                editor={activeEditorEntry.editor}
                modals={activeEditorEntry.modals}
              />
            </div>
          ) : editorMode === "markdown" ? (
            <div className="flex-1 min-w-0 overflow-x-auto">
              <MarkdownToolbar api={markdownApi} />
            </div>
          ) : (
            <div className="flex-1 py-2" />
          )}
        </div>
      )}

      {editorMode === "richtext" && (
        <FindReplaceBar
          open={findOpen}
          editor={activeEditorEntry?.editor ?? null}
          onClose={onCloseFind}
        />
      )}
    </div>
  );
}
