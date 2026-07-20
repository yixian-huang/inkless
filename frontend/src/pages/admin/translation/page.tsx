import { useState, useEffect, useCallback } from "react";
import {
  translateText,
  getGlossary,
  addGlossaryTerm,
  deleteGlossaryTerm,
  type GlossaryTerm,
  type GlossaryListResponse,
} from "@/api/translation";
import {
  AdminPageHeader,
  AdminPagination,
  AdminTable,
  AdminTableBody,
  AdminTableHead,
  AdminTd,
  AdminTh,
  useAdminConfirm,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

const LANG_OPTIONS = [
  { value: "zh", label: "中文" },
  { value: "en", label: "英文" },
];

const PAGE_SIZE = 20;

// ---- Translation Tool ----
function TranslationTool() {
  const [sourceText, setSourceText] = useState("");
  const [translatedText, setTranslatedText] = useState("");
  const [sourceLang, setSourceLang] = useState("zh");
  const [targetLang, setTargetLang] = useState("en");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSwap = () => {
    setSourceLang(targetLang);
    setTargetLang(sourceLang);
    setSourceText(translatedText);
    setTranslatedText(sourceText);
  };

  const handleTranslate = async () => {
    if (!sourceText.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const result = await translateText({
        text: sourceText,
        sourceLang: sourceLang,
        targetLang: targetLang,
      });
      setTranslatedText(result.translatedText);
    } catch (error) {
      const message = (error as {
        response?: { data?: { error?: { message?: string } } };
      })?.response?.data?.error?.message;
      setError(message || "翻译失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white shadow rounded-lg p-6 mb-8">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">翻译工具</h3>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md text-red-700 text-sm">
          {error}
        </div>
      )}

      <div className="flex items-center gap-4 mb-4">
        <div className="flex-1">
          <label className="block text-sm font-medium text-gray-700 mb-1">源语言</label>
          <select
            value={sourceLang}
            onChange={(e) => setSourceLang(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {LANG_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        </div>

        <button
          onClick={handleSwap}
          className="mt-5 p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-md transition-colors"
          title="交换语言"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 3M21 7.5H7.5" />
          </svg>
        </button>

        <div className="flex-1">
          <label className="block text-sm font-medium text-gray-700 mb-1">目标语言</label>
          <select
            value={targetLang}
            onChange={(e) => setTargetLang(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {LANG_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4 mb-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">原文</label>
          <textarea
            value={sourceText}
            onChange={(e) => setSourceText(e.target.value)}
            rows={6}
            placeholder="请输入要翻译的文本..."
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">译文</label>
          <textarea
            value={translatedText}
            readOnly
            rows={6}
            placeholder="翻译结果将显示在这里..."
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-50 resize-none"
          />
        </div>
      </div>

      <button
        onClick={handleTranslate}
        disabled={loading || !sourceText.trim()}
        className="px-6 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? (
          <span className="flex items-center gap-2">
            <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            翻译中...
          </span>
        ) : "翻译"}
      </button>
    </div>
  );
}

// ---- Glossary Management ----
function GlossaryManagement() {
  const [data, setData] = useState<GlossaryListResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const { confirm, confirmDialog } = useAdminConfirm();

  // Add form state
  const [addForm, setAddForm] = useState({
    sourceTerm: "",
    targetTerm: "",
    sourceLang: "zh",
    targetLang: "en",
  });
  const [adding, setAdding] = useState(false);
  const [addError, setAddError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getGlossary(page, PAGE_SIZE);
      setData(result);
    } catch {
      setError("获取术语表失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!addForm.sourceTerm.trim() || !addForm.targetTerm.trim()) return;
    setAdding(true);
    setAddError(null);
    try {
      await addGlossaryTerm(addForm);
      setAddForm({ sourceTerm: "", targetTerm: "", sourceLang: "zh", targetLang: "en" });
      await fetchData();
    } catch {
      setAddError("添加术语失败");
    } finally {
      setAdding(false);
    }
  };

  const handleDelete = async (id: number) => {
    const ok = await confirm({
      title: "删除术语",
      message: "确认删除此术语？此操作不可撤销。",
      confirmLabel: "删除",
      danger: true,
    });
    if (!ok) return;
    try {
      await deleteGlossaryTerm(id);
      await fetchData();
    } catch {
      setError("删除术语失败");
    }
  };

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0;

  return (
    <div className="space-y-4">
      {confirmDialog}
      <h3 className="text-lg font-semibold text-slate-900">术语管理</h3>

      {/* Add form */}
      <form onSubmit={handleAdd} className="p-4 bg-slate-50 rounded-lg border border-slate-200">
        <h4 className="text-sm font-medium text-slate-700 mb-3">添加术语</h4>
        {addError && (
          <div className="mb-3 p-2 bg-red-50 border border-red-200 rounded text-red-700 text-sm">
            {addError}
          </div>
        )}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div>
            <label className="block text-xs text-slate-600 mb-1">源术语</label>
            <input
              type="text"
              value={addForm.sourceTerm}
              onChange={(e) => setAddForm((f) => ({ ...f, sourceTerm: e.target.value }))}
              placeholder="源术语"
              className="w-full border border-slate-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          <div>
            <label className="block text-xs text-slate-600 mb-1">目标术语</label>
            <input
              type="text"
              value={addForm.targetTerm}
              onChange={(e) => setAddForm((f) => ({ ...f, targetTerm: e.target.value }))}
              placeholder="目标术语"
              className="w-full border border-slate-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          <div>
            <label className="block text-xs text-slate-600 mb-1">源语言</label>
            <select
              value={addForm.sourceLang}
              onChange={(e) => setAddForm((f) => ({ ...f, sourceLang: e.target.value }))}
              className="w-full border border-slate-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {LANG_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-slate-600 mb-1">目标语言</label>
            <select
              value={addForm.targetLang}
              onChange={(e) => setAddForm((f) => ({ ...f, targetLang: e.target.value }))}
              className="w-full border border-slate-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {LANG_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="mt-3">
          <button
            type="submit"
            disabled={adding}
            className="px-4 py-2 bg-green-600 text-white text-sm font-medium rounded-md hover:bg-green-700 disabled:opacity-50"
          >
            {adding ? "添加中..." : "添加术语"}
          </button>
        </div>
      </form>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-md text-red-700 text-sm">
          {error}
        </div>
      )}

      {loading && !data ? (
        <div className="text-center py-8 text-slate-500">加载中...</div>
      ) : (
        <>
          <AdminTable>
            <AdminTableHead>
              <tr>
                <AdminTh>源术语</AdminTh>
                <AdminTh>目标术语</AdminTh>
                <AdminTh>源语言</AdminTh>
                <AdminTh>目标语言</AdminTh>
                <AdminTh>操作</AdminTh>
              </tr>
            </AdminTableHead>
            <AdminTableBody>
              {data?.items.map((term: GlossaryTerm) => (
                <tr key={term.id} className="hover:bg-slate-50/80">
                  <AdminTd className="text-slate-900">{term.sourceTerm}</AdminTd>
                  <AdminTd className="text-slate-700">{term.targetTerm}</AdminTd>
                  <AdminTd className="text-slate-500">
                    {LANG_OPTIONS.find((o) => o.value === term.sourceLang)?.label || term.sourceLang}
                  </AdminTd>
                  <AdminTd className="text-slate-500">
                    {LANG_OPTIONS.find((o) => o.value === term.targetLang)?.label || term.targetLang}
                  </AdminTd>
                  <AdminTd>
                    <button
                      onClick={() => handleDelete(term.id)}
                      className="text-red-600 hover:text-red-800 text-xs font-medium"
                    >
                      删除
                    </button>
                  </AdminTd>
                </tr>
              ))}
              {data?.items.length === 0 && (
                <tr>
                  <AdminTd colSpan={5} className="py-8 text-center text-slate-500">
                    暂无术语记录
                  </AdminTd>
                </tr>
              )}
            </AdminTableBody>
          </AdminTable>

          <AdminPagination
            page={page}
            totalPages={totalPages}
            total={data?.total ?? 0}
            onPageChange={setPage}
          />
        </>
      )}
    </div>
  );
}

// ---- Main Page ----
export default function AdminTranslationPage() {
  useDocumentTitle("翻译管理");
  return (
    <div>
      <AdminPageHeader
        title="翻译管理"
        description="双语内容翻译与术语表管理"
      />
      <TranslationTool />
      <GlossaryManagement />
    </div>
  );
}
