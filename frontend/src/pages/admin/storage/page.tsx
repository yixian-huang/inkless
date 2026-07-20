import { useState, useEffect, useCallback } from "react";
import {
  getStorageConfig,
  updateStorageConfig,
  testStorageConnection,
  type StorageConfig,
  type UpdateStorageConfigRequest,
} from "@/api/storage";
import {
  AdminButton,
  AdminCard,
  AdminErrorBanner,
  AdminField,
  AdminInput,
  AdminLoading,
  AdminPageHeader,
  AdminSuccessBanner,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

type Strategy = "local" | "s3" | "oss";

interface StorageFormData {
  strategy: Strategy;
  bucket: string;
  region: string;
  endpoint: string;
  access_key: string;
  secret_key: string;
  base_path: string;
}

const emptyForm: StorageFormData = {
  strategy: "local",
  bucket: "",
  region: "",
  endpoint: "",
  access_key: "",
  secret_key: "",
  base_path: "",
};

const STRATEGIES = [
  { value: "local" as Strategy, label: "本地存储", description: "文件存储在服务器本地磁盘" },
  { value: "s3" as Strategy, label: "Amazon S3", description: "AWS S3 或兼容 S3 的对象存储" },
  { value: "oss" as Strategy, label: "阿里云 OSS", description: "阿里云对象存储服务" },
];

export default function AdminStoragePage() {
  useDocumentTitle("存储配置");
  const [config, setConfig] = useState<StorageConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [form, setForm] = useState<StorageFormData>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [saveSuccess, setSaveSuccess] = useState(false);

  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const data = await getStorageConfig();
      setConfig(data);
      setForm({
        strategy: (data.strategy as Strategy) || "local",
        bucket: data.bucket || "",
        region: data.region || "",
        endpoint: data.endpoint || "",
        access_key: data.accessKey || "",
        secret_key: "",
        base_path: data.basePath || "",
      });
    } catch {
      setError("加载存储配置失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const handleSave = async () => {
    setSaveError("");
    setSaveSuccess(false);
    if (form.strategy !== "local" && !form.bucket.trim()) {
      setSaveError("请填写存储桶名称");
      return;
    }
    setSaving(true);
    try {
      const data: UpdateStorageConfigRequest = {
        strategy: form.strategy,
        bucket: form.bucket,
        region: form.region,
        endpoint: form.endpoint,
        accessKey: form.access_key,
        basePath: form.base_path,
      };
      if (form.secret_key) {
        data.secretKey = form.secret_key;
      }
      const updated = await updateStorageConfig(data);
      setConfig(updated);
      setSaveSuccess(true);
      setTimeout(() => setSaveSuccess(false), 3000);
    } catch (err: any) {
      const msg = err?.response?.data?.error?.message || "保存失败";
      setSaveError(msg);
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      const result = await testStorageConnection();
      setTestResult(result);
    } catch (err: any) {
      setTestResult({
        success: false,
        message: err?.response?.data?.error?.message || "连接测试失败",
      });
    } finally {
      setTesting(false);
    }
  };

  const isRemote = form.strategy === "s3" || form.strategy === "oss";

  return (
    <div className="space-y-6">
      <AdminPageHeader title="存储配置" description="配置文件上传和存储方式" />

      {error && <AdminErrorBanner message={error} />}

      {loading ? (
        <AdminLoading />
      ) : (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <AdminCard className="space-y-6 lg:col-span-2">
            <div>
              <div className="mb-3 text-sm font-semibold text-slate-700">存储策略</div>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                {STRATEGIES.map((s) => {
                  const active = form.strategy === s.value;
                  return (
                    <label
                      key={s.value}
                      className={`relative flex cursor-pointer flex-col gap-1 rounded-2xl border-2 p-4 transition-all ${
                        active
                          ? "border-blue-500 bg-blue-50/80 shadow-sm shadow-blue-600/10"
                          : "border-slate-200 hover:border-slate-300"
                      }`}
                    >
                      <input
                        type="radio"
                        name="strategy"
                        value={s.value}
                        checked={active}
                        onChange={(e) => {
                          setForm((f) => ({ ...f, strategy: e.target.value as Strategy }));
                          setTestResult(null);
                        }}
                        className="sr-only"
                      />
                      <span className="text-sm font-semibold text-slate-900">{s.label}</span>
                      <span className="text-xs text-slate-500">{s.description}</span>
                    </label>
                  );
                })}
              </div>
            </div>

            {isRemote && (
              <div className="space-y-4 border-t border-slate-100 pt-4">
                <h3 className="text-sm font-semibold text-slate-700">
                  {form.strategy === "s3" ? "S3 配置" : "OSS 配置"}
                </h3>
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <AdminField label="存储桶 (Bucket) *">
                    <AdminInput
                      type="text"
                      value={form.bucket}
                      onChange={(e) => setForm((f) => ({ ...f, bucket: e.target.value }))}
                      className="font-mono"
                      placeholder="my-bucket"
                    />
                  </AdminField>
                  <AdminField label="地区 (Region)">
                    <AdminInput
                      type="text"
                      value={form.region}
                      onChange={(e) => setForm((f) => ({ ...f, region: e.target.value }))}
                      className="font-mono"
                      placeholder={form.strategy === "s3" ? "us-east-1" : "oss-cn-hangzhou"}
                    />
                  </AdminField>
                  <AdminField
                    className="sm:col-span-2"
                    label={
                      form.strategy === "s3" ? "Endpoint (自定义端点，可选)" : "Endpoint"
                    }
                  >
                    <AdminInput
                      type="text"
                      value={form.endpoint}
                      onChange={(e) => setForm((f) => ({ ...f, endpoint: e.target.value }))}
                      className="font-mono"
                      placeholder={
                        form.strategy === "s3"
                          ? "https://s3.amazonaws.com"
                          : "https://oss-cn-hangzhou.aliyuncs.com"
                      }
                    />
                  </AdminField>
                  <AdminField label="Access Key">
                    <AdminInput
                      type="text"
                      value={form.access_key}
                      onChange={(e) => setForm((f) => ({ ...f, access_key: e.target.value }))}
                      className="font-mono"
                      placeholder="输入 Access Key"
                      autoComplete="off"
                    />
                  </AdminField>
                  <AdminField
                    label={
                      config?.hasSecretKey
                        ? "Secret Key（已配置，留空不修改）"
                        : "Secret Key"
                    }
                  >
                    <AdminInput
                      type="password"
                      value={form.secret_key}
                      onChange={(e) => setForm((f) => ({ ...f, secret_key: e.target.value }))}
                      className="font-mono"
                      placeholder={config?.hasSecretKey ? "留空不修改" : "输入 Secret Key"}
                      autoComplete="new-password"
                    />
                  </AdminField>
                </div>
              </div>
            )}

            <AdminField
              label="基础路径 (Base Path)"
              hint={form.strategy === "local" ? "本地存储的相对目录路径" : "存储桶内的前缀路径"}
            >
              <AdminInput
                type="text"
                value={form.base_path}
                onChange={(e) => setForm((f) => ({ ...f, base_path: e.target.value }))}
                className="font-mono"
                placeholder={form.strategy === "local" ? "uploads/" : "media/"}
              />
            </AdminField>

            {saveError ? <AdminErrorBanner message={saveError} className="mb-0" /> : null}
            {saveSuccess ? <AdminSuccessBanner message="配置已保存" className="mb-0" /> : null}

            <div className="flex items-center justify-between border-t border-slate-100 pt-4">
              <div className="flex items-center gap-3">
                {isRemote && (
                  <AdminButton variant="secondary" onClick={handleTest} disabled={testing}>
                    {testing ? "测试中…" : "测试连接"}
                  </AdminButton>
                )}
                {testResult && (
                  <span
                    className={`text-sm ${testResult.success ? "text-emerald-600" : "text-red-600"}`}
                  >
                    {testResult.message}
                  </span>
                )}
              </div>
              <AdminButton onClick={handleSave} disabled={saving}>
                {saving ? "保存中…" : "保存配置"}
              </AdminButton>
            </div>
          </AdminCard>

          <AdminCard title="当前配置" className="h-fit">
            {config ? (
              <dl className="space-y-3">
                <div>
                  <dt className="text-xs text-slate-500">存储策略</dt>
                  <dd className="mt-0.5 text-sm font-medium text-slate-900">
                    {STRATEGIES.find((s) => s.value === config.strategy)?.label || config.strategy}
                  </dd>
                </div>
                {config.bucket && (
                  <div>
                    <dt className="text-xs text-slate-500">存储桶</dt>
                    <dd className="mt-0.5 font-mono text-sm text-slate-900">{config.bucket}</dd>
                  </div>
                )}
                {config.region && (
                  <div>
                    <dt className="text-xs text-slate-500">地区</dt>
                    <dd className="mt-0.5 font-mono text-sm text-slate-900">{config.region}</dd>
                  </div>
                )}
                {config.endpoint && (
                  <div>
                    <dt className="text-xs text-slate-500">Endpoint</dt>
                    <dd className="mt-0.5 break-all font-mono text-sm text-slate-900">
                      {config.endpoint}
                    </dd>
                  </div>
                )}
                {config.accessKey && (
                  <div>
                    <dt className="text-xs text-slate-500">Access Key</dt>
                    <dd className="mt-0.5 font-mono text-sm text-slate-900">
                      {config.accessKey.slice(0, 8)}
                      {"*".repeat(Math.max(0, config.accessKey.length - 8))}
                    </dd>
                  </div>
                )}
                <div>
                  <dt className="text-xs text-slate-500">Secret Key</dt>
                  <dd className="mt-0.5 text-sm">
                    {config.hasSecretKey ? (
                      <span className="text-emerald-600">已配置</span>
                    ) : (
                      <span className="text-slate-400">未配置</span>
                    )}
                  </dd>
                </div>
                {config.basePath && (
                  <div>
                    <dt className="text-xs text-slate-500">基础路径</dt>
                    <dd className="mt-0.5 font-mono text-sm text-slate-900">{config.basePath}</dd>
                  </div>
                )}
                {config.updatedAt && (
                  <div className="border-t border-slate-100 pt-2">
                    <dt className="text-xs text-slate-500">最后更新</dt>
                    <dd className="mt-0.5 text-xs text-slate-600">
                      {new Date(config.updatedAt).toLocaleString("zh-CN")}
                    </dd>
                  </div>
                )}
              </dl>
            ) : (
              <p className="text-sm text-slate-500">暂无配置</p>
            )}
          </AdminCard>
        </div>
      )}
    </div>
  );
}
