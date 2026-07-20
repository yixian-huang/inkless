import { useState, useRef, useEffect } from "react";
import { EditorView, basicSetup } from "codemirror";
import { html } from "@codemirror/lang-html";
import { oneDark } from "@codemirror/theme-one-dark";
import { EditorState } from "@codemirror/state";
import type { EmailTemplate } from "./types";
import {
  AdminButton,
  AdminField,
  AdminFilterChip,
  AdminInput,
  AdminModal,
} from "@/components/admin/ui";

interface TemplateEditorTabProps {
  templates: Record<string, EmailTemplate>;
  onChange: (templates: Record<string, EmailTemplate>) => void;
  onSave: () => void;
  isSaving?: boolean;
}

const VARIABLES = [
  { key: "{{name}}", label: "姓名 / Name" },
  { key: "{{email}}", label: "邮箱 / Email" },
  { key: "{{phone}}", label: "电话 / Phone" },
  { key: "{{company}}", label: "公司 / Company" },
  { key: "{{message}}", label: "留言 / Message" },
  { key: "{{date}}", label: "日期 / Date" },
];

export default function TemplateEditorTab({
  templates,
  onChange,
  onSave,
  isSaving,
}: TemplateEditorTabProps) {
  const [locale, setLocale] = useState<"zh" | "en">("zh");
  const [showPreview, setShowPreview] = useState(false);
  const editorContainerRef = useRef<HTMLDivElement>(null);
  const editorViewRef = useRef<EditorView | null>(null);

  const currentTemplate = templates[locale] || { subject: "", body: "" };

  useEffect(() => {
    if (!editorContainerRef.current) return;

    if (editorViewRef.current) {
      editorViewRef.current.destroy();
    }

    const state = EditorState.create({
      doc: currentTemplate.body,
      extensions: [
        basicSetup,
        html(),
        oneDark,
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            const newBody = update.state.doc.toString();
            onChange({
              ...templates,
              [locale]: { ...templates[locale], body: newBody },
            });
          }
        }),
        EditorView.theme({
          "&": { height: "400px", fontSize: "13px" },
          ".cm-scroller": { overflow: "auto" },
        }),
      ],
    });

    const view = new EditorView({
      state,
      parent: editorContainerRef.current,
    });

    editorViewRef.current = view;

    return () => {
      view.destroy();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [locale]);

  const handleSubjectChange = (value: string) => {
    onChange({
      ...templates,
      [locale]: { ...currentTemplate, subject: value },
    });
  };

  const handleInsertVariable = (variable: string) => {
    const view = editorViewRef.current;
    if (view) {
      const { from, to } = view.state.selection.main;
      view.dispatch({
        changes: { from, to, insert: variable },
        selection: { anchor: from + variable.length },
      });
      view.focus();
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <span className="mr-1 text-sm font-medium text-slate-700">语言版本</span>
        <AdminFilterChip active={locale === "zh"} onClick={() => setLocale("zh")}>
          中文
        </AdminFilterChip>
        <AdminFilterChip active={locale === "en"} onClick={() => setLocale("en")}>
          English
        </AdminFilterChip>
      </div>

      <AdminField
        label="邮件主题"
        hint={'支持变量：{{name}}, {{email}}, {{company}}'}
      >
        <AdminInput
          type="text"
          value={currentTemplate.subject}
          onChange={(e) => handleSubjectChange(e.target.value)}
          placeholder={locale === "zh" ? "输入邮件主题…" : "Enter email subject…"}
        />
      </AdminField>

      <div>
        <div className="mb-2 text-sm font-medium text-slate-700">插入变量</div>
        <div className="flex flex-wrap gap-2">
          {VARIABLES.map((v) => (
            <button
              key={v.key}
              type="button"
              onClick={() => handleInsertVariable(v.key)}
              className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-1.5 font-mono text-xs text-slate-700 transition hover:bg-slate-100"
              title={v.label}
            >
              {v.key}
            </button>
          ))}
        </div>
      </div>

      <div>
        <div className="mb-2 text-sm font-medium text-slate-700">邮件正文 (HTML)</div>
        <div
          ref={editorContainerRef}
          className="overflow-hidden rounded-2xl border border-slate-200"
        />
      </div>

      <div className="flex items-center justify-between border-t border-slate-100 pt-4">
        <AdminButton variant="secondary" onClick={() => setShowPreview(true)}>
          预览模板
        </AdminButton>
        <AdminButton onClick={onSave} disabled={isSaving}>
          {isSaving ? "保存中…" : "保存配置"}
        </AdminButton>
      </div>

      <AdminModal
        open={showPreview}
        title="模板预览"
        onClose={() => setShowPreview(false)}
        widthClass="max-w-3xl"
      >
        <p className="mb-3 text-sm text-slate-500">主题：{currentTemplate.subject}</p>
        <iframe
          srcDoc={currentTemplate.body}
          sandbox="allow-same-origin"
          title="Template Preview"
          className="h-[500px] w-full rounded-xl border border-slate-200 bg-white"
        />
      </AdminModal>
    </div>
  );
}
