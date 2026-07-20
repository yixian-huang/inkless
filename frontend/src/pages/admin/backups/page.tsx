import { useState, useEffect, useCallback, useRef } from "react";
import {
  getBackups,
  triggerBackup,
  triggerExport,
  downloadExport,
  validateImport,
  runImport,
  type BackupRecord,
  type ExportRecord,
  type ValidationResult,
} from "@/api/backups";
import {
  AdminEmptyState,
  AdminErrorBanner,
  AdminLoading,
  AdminPageHeader,
  AdminTable,
  AdminTableBody,
  AdminTableHead,
  AdminTd,
  AdminTh,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { removeStoredAccessToken, removeStoredRefreshToken } from "@/lib/browserStorage";

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

type TabKey = "db" | "export" | "import";

const tabs: { key: TabKey; label: string }[] = [
  { key: "db", label: "数据库备份" },
  { key: "export", label: "站点导出" },
  { key: "import", label: "站点导入" },
];

// ---- Database Backup Tab ----
function DatabaseBackupTab() {
  const [backups, setBackups] = useState<BackupRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const fetchBackups = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getBackups();
      setBackups(data);
    } catch {
      setError("获取备份列表失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchBackups();
  }, [fetchBackups]);

  const handleTriggerBackup = async () => {
    setCreating(true);
    setError(null);
    setSuccessMsg(null);
    try {
      const newBackup = await triggerBackup();
      setSuccessMsg(`备份创建成功: ${newBackup.filename}`);
      await fetchBackups();
    } catch {
      setError("创建备份失败，请稍后重试");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-4">
        <p className="text-sm text-slate-500">创建数据库级别的备份（不含媒体文件）</p>
        <div className="flex gap-2">
          <button
            onClick={handleTriggerBackup}
            disabled={creating}
            className="inline-flex h-9 items-center justify-center rounded-xl bg-emerald-600 px-3.5 text-sm font-medium text-white shadow-sm hover:bg-emerald-700 disabled:opacity-50"
          >
            {creating ? "创建中..." : "手动备份"}
          </button>
          <button
            onClick={fetchBackups}
            disabled={loading}
            className="inline-flex h-9 items-center justify-center rounded-xl bg-blue-600 px-3.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? "加载中..." : "刷新"}
          </button>
        </div>
      </div>

      {successMsg && (
        <div className="mb-4 rounded-2xl border border-emerald-200/80 bg-emerald-50/90 px-4 py-3 text-sm text-emerald-800">
          {successMsg}
        </div>
      )}
      {error && <AdminErrorBanner message={error} />}

      {loading && backups.length === 0 ? (
        <AdminLoading />
      ) : backups.length === 0 ? (
        <AdminEmptyState title="暂无备份记录" description="点击「手动备份」创建第一份数据库备份。" />
      ) : (
        <AdminTable>
          <AdminTableHead>
            <tr>
              <AdminTh>文件名</AdminTh>
              <AdminTh className="text-right">大小</AdminTh>
              <AdminTh className="text-right">创建时间</AdminTh>
            </tr>
          </AdminTableHead>
          <AdminTableBody>
            {backups.map((backup) => (
              <tr key={backup.id} className="hover:bg-slate-50/80">
                <AdminTd className="font-medium text-slate-900">{backup.filename}</AdminTd>
                <AdminTd className="text-right tabular-nums">{formatFileSize(backup.size)}</AdminTd>
                <AdminTd className="text-right whitespace-nowrap">
                  {new Date(backup.createdAt).toLocaleString("zh-CN")}
                </AdminTd>
              </tr>
            ))}
          </AdminTableBody>
        </AdminTable>
      )}
    </div>
  );
}

// ---- Site Export Tab ----
function SiteExportTab() {
  const [exporting, setExporting] = useState(false);
  const [exportResult, setExportResult] = useState<ExportRecord | null>(null);
  const [downloading, setDownloading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleExport = async () => {
    setExporting(true);
    setError(null);
    setExportResult(null);
    try {
      const result = await triggerExport();
      setExportResult(result);
    } catch {
      setError("导出失败，请稍后重试");
    } finally {
      setExporting(false);
    }
  };

  const handleDownload = async (filename: string) => {
    setDownloading(true);
    try {
      await downloadExport(filename);
    } catch {
      setError("下载失败");
    } finally {
      setDownloading(false);
    }
  };

  return (
    <div>
      <p className="text-sm text-slate-500 mb-4">
        导出全站数据（所有表数据 + 媒体文件）为 ZIP 归档，可用于迁移到其他实例。
      </p>

      <button
        onClick={handleExport}
        disabled={exporting}
        className="inline-flex h-9 items-center justify-center rounded-xl bg-emerald-600 px-3.5 text-sm font-medium text-white shadow-sm hover:bg-emerald-700 disabled:opacity-50"
      >
        {exporting ? "导出中..." : "生成导出包"}
      </button>

      {error && (
        <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-md text-red-700 text-sm">
          {error}
        </div>
      )}

      {exportResult && (
        <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded-md">
          <p className="text-green-700 text-sm font-medium mb-2">导出成功</p>
          <div className="text-sm text-slate-700 space-y-1">
            <p>文件名: {exportResult.filename}</p>
            <p>大小: {formatFileSize(exportResult.size)}</p>
          </div>
          <button
            onClick={() => handleDownload(exportResult.filename)}
            disabled={downloading}
            className="mt-3 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {downloading ? "下载中..." : "下载"}
          </button>
        </div>
      )}
    </div>
  );
}

// ---- Site Import Tab ----
function SiteImportTab() {
  const [file, setFile] = useState<File | null>(null);
  const [validating, setValidating] = useState(false);
  const [validation, setValidation] = useState<ValidationResult | null>(null);
  const [importing, setImporting] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0];
    if (!selected) return;

    setFile(selected);
    setValidation(null);
    setError(null);
    setShowConfirm(false);

    // Auto-validate on file selection
    setValidating(true);
    try {
      const result = await validateImport(selected);
      setValidation(result);
    } catch {
      setError("验证失败，请检查文件格式");
    } finally {
      setValidating(false);
    }
  };

  const handleImport = async () => {
    if (!file) return;
    setImporting(true);
    setError(null);
    try {
      await runImport(file);
      // Clear JWT and redirect to login
      removeStoredAccessToken();
      removeStoredRefreshToken();
      window.location.href = "/admin/login";
    } catch {
      setError("导入失败，请稍后重试");
      setImporting(false);
    }
  };

  const reset = () => {
    setFile(null);
    setValidation(null);
    setError(null);
    setShowConfirm(false);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const totalRecords = validation
    ? Object.values(validation.tables).reduce((sum, t) => sum + t.count, 0)
    : 0;

  return (
    <div>
      <p className="text-sm text-slate-500 mb-4">
        从导出归档恢复全站数据。此操作将覆盖所有现有数据，请谨慎操作。
      </p>

      <div className="flex items-center gap-3 mb-4">
        <input
          ref={fileInputRef}
          type="file"
          accept=".zip"
          onChange={handleFileChange}
          className="block text-sm text-slate-500 file:mr-4 file:py-2 file:px-4 file:rounded-md file:border-0 file:text-sm file:font-medium file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
        />
        {file && (
          <button onClick={reset} className="text-sm text-slate-400 hover:text-slate-600">
            清除
          </button>
        )}
      </div>

      {validating && (
        <div className="p-4 bg-blue-50 border border-blue-200 rounded-md text-blue-700 text-sm">
          正在验证文件...
        </div>
      )}

      {error && (
        <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-md text-red-700 text-sm">
          {error}
        </div>
      )}

      {validation && !validation.valid && (
        <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-md">
          <p className="text-red-700 text-sm font-medium mb-2">验证失败</p>
          <ul className="list-disc list-inside text-red-600 text-sm space-y-1">
            {validation.errors?.map((err, i) => <li key={i}>{err}</li>)}
          </ul>
        </div>
      )}

      {validation && validation.valid && (
        <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded-md">
          <p className="text-green-700 text-sm font-medium mb-3">验证通过 — 归档预览</p>
          <div className="text-sm text-slate-700 space-y-1 mb-3">
            <p>导出版本: {validation.version || "未知"}</p>
            <p>导出时间: {validation.exportedAt ? new Date(validation.exportedAt).toLocaleString("zh-CN") : "未知"}</p>
            <p>媒体文件: {validation.mediaFiles} 个</p>
            <p>数据记录: {totalRecords} 条</p>
          </div>

          <div className="mb-4">
            <AdminTable>
              <AdminTableHead>
                <tr>
                  <AdminTh>表名</AdminTh>
                  <AdminTh className="text-right">记录数</AdminTh>
                </tr>
              </AdminTableHead>
              <AdminTableBody>
                {Object.entries(validation.tables).map(([name, info]) => (
                  <tr key={name} className="hover:bg-slate-50/80">
                    <AdminTd>{name}</AdminTd>
                    <AdminTd className="text-right tabular-nums">{info.count}</AdminTd>
                  </tr>
                ))}
              </AdminTableBody>
            </AdminTable>
          </div>

          {!showConfirm ? (
            <button
              onClick={() => setShowConfirm(true)}
              className="px-4 py-2 bg-red-600 text-white text-sm font-medium rounded-md hover:bg-red-700"
            >
              确认导入
            </button>
          ) : (
            <div className="p-4 bg-red-50 border border-red-300 rounded-md">
              <p className="text-red-800 text-sm font-medium mb-3">
                此操作将清空所有现有数据并替换为归档中的数据。导入完成后需要重新登录。确定继续？
              </p>
              <div className="flex gap-2">
                <button
                  onClick={handleImport}
                  disabled={importing}
                  className="px-4 py-2 bg-red-600 text-white text-sm font-medium rounded-md hover:bg-red-700 disabled:opacity-50"
                >
                  {importing ? "导入中..." : "确定导入"}
                </button>
                <button
                  onClick={() => setShowConfirm(false)}
                  disabled={importing}
                  className="inline-flex h-9 items-center justify-center rounded-xl border border-slate-200 bg-white px-3.5 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:opacity-50"
                >
                  取消
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ---- Main Page ----
export default function AdminBackupsPage() {
  useDocumentTitle("数据备份");
  const [activeTab, setActiveTab] = useState<TabKey>("db");

  return (
    <div>
      <AdminPageHeader
        title="数据备份"
        description="数据库备份、站点导出与导入"
      />

      {/* Tab navigation */}
      <div className="border-b border-slate-200 mb-6">
        <nav className="-mb-px flex space-x-8">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`whitespace-nowrap py-3 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.key
                  ? "border-blue-500 text-blue-600"
                  : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-200"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab content */}
      {activeTab === "db" && <DatabaseBackupTab />}
      {activeTab === "export" && <SiteExportTab />}
      {activeTab === "import" && <SiteImportTab />}
    </div>
  );
}
