import { useState } from "react";
import {
  generateWizardPlan,
  applyWizardPlan,
  type WizardPlan,
  type WizardPlanRequest,
} from "@/api/wizard";
import { AdminErrorBanner, AdminPageHeader } from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

const INDUSTRY_OPTIONS = [
  { value: "technology", label: "科技 / IT" },
  { value: "consulting", label: "咨询 / 专业服务" },
  { value: "education", label: "教育 / 培训" },
  { value: "healthcare", label: "医疗 / 健康" },
  { value: "retail", label: "零售 / 电商" },
  { value: "finance", label: "金融 / 投资" },
  { value: "media", label: "媒体 / 创意" },
  { value: "manufacturing", label: "制造 / 工业" },
  { value: "other", label: "其他" },
];

const STYLE_OPTIONS = [
  { value: "modern", label: "现代简约" },
  { value: "corporate", label: "商务专业" },
  { value: "creative", label: "创意活泼" },
  { value: "minimal", label: "极简风格" },
  { value: "warm", label: "温暖亲切" },
];

const FEATURE_OPTIONS = [
  { value: "blog", label: "博客/资讯" },
  { value: "contact_form", label: "联系表单" },
  { value: "gallery", label: "图片展示" },
  { value: "team", label: "团队介绍" },
  { value: "testimonials", label: "客户评价" },
  { value: "pricing", label: "定价方案" },
  { value: "faq", label: "常见问题" },
  { value: "newsletter", label: "邮件订阅" },
];

const CONTENT_TYPE_OPTIONS = [
  { value: "home", label: "首页" },
  { value: "about", label: "关于我们" },
  { value: "services", label: "服务/产品" },
  { value: "portfolio", label: "案例展示" },
  { value: "contact", label: "联系我们" },
];

// ---- Step Indicator ----
function StepIndicator({ current }: { current: number }) {
  const steps = ["填写信息", "审核方案", "确认应用"];
  return (
    <div className="flex items-center justify-center mb-8">
      {steps.map((label, i) => {
        const stepNum = i + 1;
        const isComplete = stepNum < current;
        const isActive = stepNum === current;
        return (
          <div key={stepNum} className="flex items-center">
            <div className="flex flex-col items-center">
              <div
                className={`w-9 h-9 rounded-full flex items-center justify-center text-sm font-semibold border-2 transition-colors ${
                  isComplete
                    ? "bg-blue-600 border-blue-600 text-white"
                    : isActive
                    ? "bg-white border-blue-600 text-blue-600"
                    : "bg-white border-slate-200 text-slate-400"
                }`}
              >
                {isComplete ? (
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                ) : stepNum}
              </div>
              <span className={`mt-1.5 text-xs font-medium ${isActive ? "text-blue-600" : "text-slate-400"}`}>
                {label}
              </span>
            </div>
            {i < steps.length - 1 && (
              <div className={`w-16 md:w-24 h-0.5 mx-2 mb-5 transition-colors ${stepNum < current ? "bg-blue-600" : "bg-slate-200"}`} />
            )}
          </div>
        );
      })}
    </div>
  );
}

// ---- Step 1: Questionnaire ----
interface FormData {
  brand_name: string;
  description: string;
  industry: string;
  style_preference: string;
  features: string[];
  content_types: string[];
}

function Step1Questionnaire({
  formData,
  onChange,
  onNext,
  loading,
}: {
  formData: FormData;
  onChange: (data: Partial<FormData>) => void;
  onNext: () => void;
  loading: boolean;
}) {
  const toggleFeature = (value: string) => {
    const current = formData.features;
    const updated = current.includes(value)
      ? current.filter((f) => f !== value)
      : [...current, value];
    onChange({ features: updated });
  };

  const toggleContentType = (value: string) => {
    const current = formData.content_types;
    const updated = current.includes(value)
      ? current.filter((f) => f !== value)
      : [...current, value];
    onChange({ content_types: updated });
  };

  const isValid = formData.brand_name.trim() && formData.industry && formData.style_preference;

  return (
    <div className="max-w-2xl mx-auto">
      <h3 className="text-lg font-semibold text-slate-900 mb-6">告诉我们您的需求</h3>

      <div className="space-y-6">
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">
            品牌名称 <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            value={formData.brand_name}
            onChange={(e) => onChange({ brand_name: e.target.value })}
            placeholder="输入您的品牌或公司名称"
            className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm transition hover:border-slate-300 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">品牌描述</label>
          <textarea
            value={formData.description}
            onChange={(e) => onChange({ description: e.target.value })}
            rows={3}
            placeholder="简要描述您的业务或品牌特点..."
            className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm transition hover:border-slate-300 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20 resize-none"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700 mb-2">
            所属行业 <span className="text-red-500">*</span>
          </label>
          <select
            value={formData.industry}
            onChange={(e) => onChange({ industry: e.target.value })}
            className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm transition hover:border-slate-300 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
          >
            <option value="">请选择行业...</option>
            {INDUSTRY_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700 mb-2">
            风格偏好 <span className="text-red-500">*</span>
          </label>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
            {STYLE_OPTIONS.map((opt) => (
              <label
                key={opt.value}
                className={`flex items-center justify-center p-3 border rounded-lg cursor-pointer transition-colors text-sm font-medium ${
                  formData.style_preference === opt.value
                    ? "border-blue-600 bg-blue-50 text-blue-700"
                    : "border-slate-200 hover:border-slate-200 text-slate-700"
                }`}
              >
                <input
                  type="radio"
                  name="style"
                  value={opt.value}
                  checked={formData.style_preference === opt.value}
                  onChange={() => onChange({ style_preference: opt.value })}
                  className="sr-only"
                />
                {opt.label}
              </label>
            ))}
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700 mb-2">需要的功能模块</label>
          <div className="grid grid-cols-2 gap-2">
            {FEATURE_OPTIONS.map((opt) => (
              <label
                key={opt.value}
                className="flex items-center gap-2 p-2 border rounded-lg cursor-pointer hover:bg-slate-50 text-sm"
              >
                <input
                  type="checkbox"
                  checked={formData.features.includes(opt.value)}
                  onChange={() => toggleFeature(opt.value)}
                  className="rounded border-slate-200 text-blue-600"
                />
                <span className="text-slate-700">{opt.label}</span>
              </label>
            ))}
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700 mb-2">需要的页面</label>
          <div className="grid grid-cols-2 gap-2">
            {CONTENT_TYPE_OPTIONS.map((opt) => (
              <label
                key={opt.value}
                className="flex items-center gap-2 p-2 border rounded-lg cursor-pointer hover:bg-slate-50 text-sm"
              >
                <input
                  type="checkbox"
                  checked={formData.content_types.includes(opt.value)}
                  onChange={() => toggleContentType(opt.value)}
                  className="rounded border-slate-200 text-blue-600"
                />
                <span className="text-slate-700">{opt.label}</span>
              </label>
            ))}
          </div>
        </div>
      </div>

      <div className="mt-8 flex justify-end">
        <button
          onClick={onNext}
          disabled={!isValid || loading}
          className="px-6 py-2.5 bg-blue-600 text-white text-sm font-medium rounded-xl hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
        >
          {loading ? (
            <>
              <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              AI 生成方案中...
            </>
          ) : (
            <>
              生成方案
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
              </svg>
            </>
          )}
        </button>
      </div>
    </div>
  );
}

// ---- Step 2: Review Plan ----
function ColorSwatch({ color, label }: { color: string; label: string }) {
  return (
    <div className="flex flex-col items-center gap-1">
      <div
        className="w-12 h-12 rounded-lg border border-slate-200 shadow-sm"
        style={{ backgroundColor: color }}
      />
      <span className="text-xs text-slate-500">{label}</span>
      <span className="text-xs font-mono text-slate-400">{color}</span>
    </div>
  );
}

function Step2ReviewPlan({
  plan,
  onBack,
  onApply,
  loading,
}: {
  plan: WizardPlan;
  onBack: () => void;
  onApply: () => void;
  loading: boolean;
}) {
  return (
    <div className="max-w-2xl mx-auto">
      <h3 className="text-lg font-semibold text-slate-900 mb-6">AI 生成的建站方案</h3>

      <div className="space-y-6">
        {/* Recommended Theme */}
        <div className="p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <h4 className="text-sm font-semibold text-blue-800 mb-1">推荐主题</h4>
          <p className="text-blue-700 text-sm">{plan.recommended_theme || "默认主题"}</p>
        </div>

        {/* Color Scheme */}
        {plan.color_scheme && (
          <div>
            <h4 className="text-sm font-semibold text-slate-700 mb-3">配色方案</h4>
            <div className="flex gap-6 flex-wrap">
              {plan.color_scheme.primary && (
                <ColorSwatch color={plan.color_scheme.primary} label="主色" />
              )}
              {plan.color_scheme.secondary && (
                <ColorSwatch color={plan.color_scheme.secondary} label="辅色" />
              )}
              {plan.color_scheme.text && (
                <ColorSwatch color={plan.color_scheme.text} label="文字色" />
              )}
              {plan.color_scheme.background && (
                <ColorSwatch color={plan.color_scheme.background} label="背景色" />
              )}
            </div>
          </div>
        )}

        {/* Page List */}
        {plan.pages && plan.pages.length > 0 && (
          <div>
            <h4 className="text-sm font-semibold text-slate-700 mb-3">页面列表</h4>
            <div className="space-y-2">
              {plan.pages.map((p, i) => (
                <div key={i} className="flex items-start gap-3 p-3 bg-slate-50 rounded-lg border border-slate-200">
                  <span className="mt-0.5 w-6 h-6 rounded-full bg-blue-100 text-blue-600 text-xs font-semibold flex items-center justify-center shrink-0">
                    {i + 1}
                  </span>
                  <div>
                    <p className="text-sm font-medium text-slate-900">{p.name}</p>
                    {p.slug && <p className="text-xs text-slate-500 mt-0.5">/{p.slug}</p>}
                    {p.description && <p className="text-xs text-slate-600 mt-1">{p.description}</p>}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Raw Plan (collapsed) */}
        <details className="text-sm">
          <summary className="cursor-pointer text-slate-500 hover:text-slate-700 select-none">查看完整方案 JSON</summary>
          <pre className="mt-2 p-3 bg-gray-900 text-green-400 rounded-lg text-xs overflow-x-auto">
            {JSON.stringify(plan, null, 2)}
          </pre>
        </details>
      </div>

      <div className="mt-8 flex justify-between">
        <button
          onClick={onBack}
          disabled={loading}
          className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-xl hover:bg-slate-50 disabled:opacity-50 flex items-center gap-2"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
          </svg>
          返回修改
        </button>
        <button
          onClick={onApply}
          disabled={loading}
          className="px-6 py-2.5 bg-green-600 text-white text-sm font-medium rounded-xl hover:bg-green-700 disabled:opacity-50 flex items-center gap-2"
        >
          {loading ? (
            <>
              <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              应用中...
            </>
          ) : (
            <>
              应用方案
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
              </svg>
            </>
          )}
        </button>
      </div>
    </div>
  );
}

// ---- Step 3: Confirmation ----
function Step3Confirmation({
  plan,
  applyResult,
  onReset,
}: {
  plan: WizardPlan;
  applyResult: { success: boolean; pages_created: number } | null;
  onReset: () => void;
}) {
  return (
    <div className="max-w-xl mx-auto text-center">
      {applyResult?.success ? (
        <>
          <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h3 className="text-xl font-bold text-slate-900 mb-2">建站方案应用成功！</h3>
          <p className="text-slate-500 text-sm mb-6">
            已成功创建 <span className="font-semibold text-slate-700">{applyResult.pages_created}</span> 个页面，
            基于 <span className="font-semibold text-slate-700">{plan.recommended_theme || "默认主题"}</span> 主题。
          </p>
          <div className="p-4 bg-slate-50 rounded-lg border border-slate-200 text-left mb-6">
            <h4 className="text-sm font-semibold text-slate-700 mb-2">已创建的页面</h4>
            <ul className="space-y-1">
              {plan.pages?.map((p, i) => (
                <li key={i} className="flex items-center gap-2 text-sm text-slate-600">
                  <svg className="w-3.5 h-3.5 text-green-500 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                  </svg>
                  {p.name}
                </li>
              ))}
            </ul>
          </div>
          <div className="flex gap-3 justify-center">
            <a
              href="/admin/pages"
              className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-xl hover:bg-blue-700"
            >
              查看页面管理
            </a>
            <button
              onClick={onReset}
              className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-xl hover:bg-slate-50"
            >
              再次使用向导
            </button>
          </div>
        </>
      ) : (
        <>
          <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <h3 className="text-xl font-bold text-slate-900 mb-2">应用失败</h3>
          <p className="text-slate-500 text-sm mb-6">方案应用过程中遇到错误，请重试。</p>
          <button
            onClick={onReset}
            className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-xl hover:bg-blue-700"
          >
            重新开始
          </button>
        </>
      )}
    </div>
  );
}

// ---- Main Page ----
const DEFAULT_FORM: FormData = {
  brand_name: "",
  description: "",
  industry: "",
  style_preference: "",
  features: [],
  content_types: [],
};

export default function AdminWizardPage() {
  useDocumentTitle("建站向导");
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState<FormData>(DEFAULT_FORM);
  const [plan, setPlan] = useState<WizardPlan | null>(null);
  const [applyResult, setApplyResult] = useState<{ success: boolean; pages_created: number } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleFormChange = (data: Partial<FormData>) => {
    setFormData((prev) => ({ ...prev, ...data }));
  };

  const handleGeneratePlan = async () => {
    setLoading(true);
    setError(null);
    try {
      const req: WizardPlanRequest = {
        ...formData,
        locale: "zh",
      };
      const result = await generateWizardPlan(req);
      setPlan(result);
      setStep(2);
    } catch (error) {
      const message = (error as {
        response?: { data?: { error?: { message?: string } } };
      })?.response?.data?.error?.message;
      setError(message || "生成方案失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  };

  const handleApplyPlan = async () => {
    if (!plan) return;
    setLoading(true);
    setError(null);
    try {
      const result = await applyWizardPlan(plan);
      setApplyResult(result);
      setStep(3);
    } catch {
      setApplyResult({ success: false, pages_created: 0 });
      setStep(3);
    } finally {
      setLoading(false);
    }
  };

  const handleReset = () => {
    setStep(1);
    setFormData(DEFAULT_FORM);
    setPlan(null);
    setApplyResult(null);
    setError(null);
  };

  return (
    <div>
      <AdminPageHeader
        title="AI 建站向导"
        description="通过 AI 快速生成并应用个性化建站方案"
      />

      <StepIndicator current={step} />

      {error && (
        <div className="mx-auto mb-6 max-w-2xl">
          <AdminErrorBanner message={error} onDismiss={() => setError(null)} />
        </div>
      )}

      <div className="bg-white shadow rounded-lg p-6 md:p-8">
        {step === 1 && (
          <Step1Questionnaire
            formData={formData}
            onChange={handleFormChange}
            onNext={handleGeneratePlan}
            loading={loading}
          />
        )}
        {step === 2 && plan && (
          <Step2ReviewPlan
            plan={plan}
            onBack={() => setStep(1)}
            onApply={handleApplyPlan}
            loading={loading}
          />
        )}
        {step === 3 && plan && (
          <Step3Confirmation
            plan={plan}
            applyResult={applyResult}
            onReset={handleReset}
          />
        )}
      </div>
    </div>
  );
}
