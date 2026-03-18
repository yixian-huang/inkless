import { EditorView, basicSetup } from "codemirror";
import { html } from "@codemirror/lang-html";
import { oneDark } from "@codemirror/theme-one-dark";
import { EditorState } from "@codemirror/state";
import type { EmailTemplate } from "./types";

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

export default function TemplateEditorTab({ templates, onChange, onSave, isSaving }: TemplateEditorTabProps) {
  const [locale, setLocale] = useState<"zh" | "en">("zh");
  const [showPreview, setShowPreview] = useState(false);
  const editorContainerRef = useRef<HTMLDivElement>(null);
  const editorViewRef = useRef<EditorView | null>(null);

  const currentTemplate = templates[locale] || { subject: "", body: "" };

  // Initialize / re-create CodeMirror when locale changes
  useEffect(() => {
    if (!editorContainerRef.current) return;

    // Destroy previous editor
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
    // We intentionally only re-create on locale change, not on every template change
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
      {/* Locale Toggle */}
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium text-gray-700 mr-2">语言版本：</span>
        <div className="inline-flex rounded-lg border border-gray-300 overflow-hidden">
          <button
            onClick={() => setLocale("zh")}
            className={`px-4 py-2 text-sm font-medium transition-colors ${
              locale === "zh"
                ? "bg-blue-600 text-white"
                : "bg-white text-gray-700 hover:bg-gray-50"
            }`}
          >
            中文
          </button>
          <button
            onClick={() => setLocale("en")}
            className={`px-4 py-2 text-sm font-medium transition-colors border-l border-gray-300 ${
              locale === "en"
                ? "bg-blue-600 text-white"
                : "bg-white text-gray-700 hover:bg-gray-50"
            }`}
          >
            English
          </button>
        </div>
      </div>

      {/* Subject Input */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          邮件主题
        </label>
        <input
          type="text"
          value={currentTemplate.subject}
          onChange={(e) => handleSubjectChange(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"
          placeholder={locale === "zh" ? "输入邮件主题..." : "Enter email subject..."}
        />
        <p className="mt-1 text-xs text-gray-500">
          支持变量：{"{{name}}"}, {"{{email}}"}, {"{{company}}"}
        </p>
      </div>

      {/* Variable Insert Buttons */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          插入变量
        </label>
        <div className="flex flex-wrap gap-2">
          {VARIABLES.map((v) => (
            <button
              key={v.key}
              onClick={() => handleInsertVariable(v.key)}
              className="px-3 py-1.5 text-xs font-mono bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 transition-colors border border-gray-200"
              title={v.label}
            >
              {v.key}
            </button>
          ))}
        </div>
      </div>

      {/* CodeMirror Editor */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          邮件正文 (HTML)
        </label>
        <div
          ref={editorContainerRef}
          className="border border-gray-300 rounded-lg overflow-hidden"
        />
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between pt-4 border-t border-gray-100">
        <button
          onClick={() => setShowPreview(true)}
          className="px-4 py-2 text-sm font-medium text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
        >
          预览模板
        </button>
        <button
          onClick={onSave}
          disabled={isSaving}
          className="px-6 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          {isSaving ? "保存中..." : "保存配置"}
        </button>
      </div>

      {/* Preview Modal */}
      {showPreview && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-xl shadow-2xl w-full max-w-3xl max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
              <div>
                <h3 className="text-lg font-semibold text-gray-900">模板预览</h3>
                <p className="text-sm text-gray-500 mt-0.5">
                  主题：{currentTemplate.subject}
                </p>
              </div>
              <button
                onClick={() => setShowPreview(false)}
                className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100 transition-colors"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="flex-1 overflow-auto p-6">
              <iframe
                srcDoc={currentTemplate.body}
                sandbox="allow-same-origin"
                title="Template Preview"
                className="w-full h-[500px] border border-gray-200 rounded-lg bg-white"
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
