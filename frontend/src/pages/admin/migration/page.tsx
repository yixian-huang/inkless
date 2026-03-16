import { useState, useEffect, useCallback, useRef } from "react";
import {
  importData,
  getMigrationJobs,
  createMigrationJobStream,
  type MigrationJob,
  type MigrationFormat,
  type MigrationJobPhase,
} from "@/api/migration";

const formatOptions: { value: MigrationFormat; label: string; description: string }[] = [
  {
    value: "wordpress",
    label: "WordPress WXR",
    description: "导入 WordPress 导出的 XML 文件，支持文章、页面、标签、分类和媒体附件。",
  },
  {
    value: "halo",
    label: "Halo JSON",
    description: "导入 Halo 博客导出的 JSON 备份，支持文章、分类、标签和自定义内容。",
  },
  {
    value: "markdown",
    label: "Markdown ZIP",
    description: "导入包含 Markdown 文件的 ZIP 压缩包，支持 Front Matter 元数据（标题、日期、标签等）。",
  },
];

const phaseConfig: Record<MigrationJobPhase, { label: string; className: string }> = {
  parsing: { label: "解析中", className: "bg-gray-100 text-gray-700" },
  importing: { label: "导入中", className: "bg-blue-100 text-blue-700" },
  done: { label: "已完成", className: "bg-green-100 text-green-700" },
  failed: { label: "失败", className: "bg-red-100 text-red-700" },
};

// ---- Progress Bar ----
function ProgressBar({ value }: { value: number }) {
  const clamped = Math.min(100, Math.max(0, value));
  return (
    <div className="w-full bg-gray-200 rounded-full h-1.5">
      <div
        className={`h-1.5 rounded-full transition-all duration-300 ${
          clamped === 100 ? "bg-green-500" : "bg-blue-500"
        }`}
        style={{ width: `${clamped}%` }}
      />
    </div>
  );
}

// ---- File Dropzone ----
interface DropzoneProps {
  accept: string;
  file: File | null;
  onFileChange: (file: File | null) => void;
}

function FileDropzone({ accept, file, onFileChange }: DropzoneProps) {
  const [dragging, setDragging] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    const dropped = e.dataTransfer.files[0];
    if (dropped) onFileChange(dropped);
  };

  return (
    <div
      className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
        dragging
          ? "border-blue-400 bg-blue-50"
          : file
          ? "border-green-400 bg-green-50"
          : "border-gray-300 hover:border-gray-400 bg-gray-50"
      }`}
      onClick={() => inputRef.current?.click()}
      onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
      onDragLeave={() => setDragging(false)}
      onDrop={handleDrop}
    >
      <input
        ref={inputRef}
        type="file"
        accept={accept}
        className="hidden"
        onChange={(e) => onFileChange(e.target.files?.[0] || null)}
      />
      {file ? (
        <div className="flex flex-col items-center gap-2">
          <svg className="w-10 h-10 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p className="text-sm font-medium text-gray-900">{file.name}</p>
          <p className="text-xs text-gray-500">
            {file.size < 1024 * 1024
              ? `${(file.size / 1024).toFixed(1)} KB`
              : `${(file.size / (1024 * 1024)).toFixed(1)} MB`}
          </p>
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); onFileChange(null); }}
            className="text-xs text-red-500 hover:text-red-700 mt-1"
          >
            移除文件
          </button>
        </div>
      ) : (
        <div className="flex flex-col items-center gap-2">
          <svg className="w-10 h-10 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
          </svg>
          <p className="text-sm font-medium text-gray-700">点击或拖放文件到此处</p>
          <p className="text-xs text-gray-400">{accept}</p>
        </div>
      )}
    </div>
  );
}

// ---- SSE Progress Hook ----
function useJobStream(
  jobId: string | null,
  onUpdate: (job: Partial<MigrationJob>) => void,
  onDone: () => void
) {
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!jobId) return;

    const es = createMigrationJobStream(jobId);
    esRef.current = es;

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as Partial<MigrationJob>;
        onUpdate(data);
        if (data.phase === "done" || data.phase === "failed") {
          es.close();
          onDone();
        }
      } catch {
        // ignore parse errors
      }
    };

    es.onerror = () => {
      es.close();
      onDone();
    };

    return () => {
      es.close();
    };
  }, [jobId, onUpdate, onDone]);

  return esRef;
}

// ---- Import Section ----
function ImportSection({ onJobCreated }: { onJobCreated: () => void }) {
  const [format, setFormat] = useState<MigrationFormat>("wordpress");
  const [file, setFile] = useState<File | null>(null);
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const selectedFormat = formatOptions.find((f) => f.value === format)!;
  const acceptMap: Record<MigrationFormat, string> = {
    wordpress: ".xml,.wxr",
    halo: ".json",
    markdown: ".zip",
  };

  const handleImport = async () => {
    if (!file) return;
    setImporting(true);
    setError(null);
    setSuccessMsg(null);
    try {
      const result = await importData(file, format);
      setSuccessMsg(`导入任务已创建，任务 ID: ${result.jobId}`);
      setFile(null);
      onJobCreated();
    } catch {
      setError("创建导入任务失败，请检查文件格式后重试");
    } finally {
      setImporting(false);
    }
  };

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6 mb-6">
      <h3 className="text-base font-semibold text-gray-900 mb-4">导入数据</h3>

      {/* Format selector */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-2">选择格式</label>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          {formatOptions.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => { setFormat(opt.value); setFile(null); }}
              className={`text-left p-3 rounded-lg border-2 transition-colors ${
                format === opt.value
                  ? "border-blue-500 bg-blue-50"
                  : "border-gray-200 hover:border-gray-300"
              }`}
            >
              <div className="text-sm font-medium text-gray-900">{opt.label}</div>
              <div className="text-xs text-gray-500 mt-1 line-clamp-2">{opt.description}</div>
            </button>
          ))}
        </div>
      </div>

      {/* Format description */}
      <div className="mb-4 p-3 bg-blue-50 border border-blue-100 rounded-md text-sm text-blue-800">
        <strong>{selectedFormat.label}：</strong>{selectedFormat.description}
      </div>

      {/* File dropzone */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-2">上传文件</label>
        <FileDropzone
          accept={acceptMap[format]}
          file={file}
          onFileChange={setFile}
        />
      </div>

      {successMsg && (
        <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-md text-green-700 text-sm">
          {successMsg}
        </div>
      )}
      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md text-red-700 text-sm">
          {error}
        </div>
      )}

      <button
        onClick={handleImport}
        disabled={!file || importing}
        className="px-5 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {importing ? "提交中..." : "开始导入"}
      </button>
    </div>
  );
}

// ---- Jobs Table ----
function JobsTable() {
  const [jobs, setJobs] = useState<MigrationJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedErrors, setExpandedErrors] = useState<Set<string>>(new Set());
  const [streamJobId, setStreamJobId] = useState<string | null>(null);

  const fetchJobs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getMigrationJobs();
      setJobs(data);
      // Auto-attach SSE to any running job
      const running = data.find((j) => j.phase === "importing");
      if (running) setStreamJobId(running.jobId);
    } catch {
      setError("获取迁移任务列表失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchJobs();
  }, [fetchJobs]);

  const handleStreamUpdate = useCallback((update: Partial<MigrationJob>) => {
    setJobs((prev) =>
      prev.map((job) =>
        job.jobId === streamJobId ? { ...job, ...update } : job
      )
    );
  }, [streamJobId]);

  const handleStreamDone = useCallback(() => {
    setStreamJobId(null);
    fetchJobs();
  }, [fetchJobs]);

  useJobStream(streamJobId, handleStreamUpdate, handleStreamDone);

  const toggleErrors = (jobId: string) => {
    setExpandedErrors((prev) => {
      const next = new Set(prev);
      if (next.has(jobId)) next.delete(jobId);
      else next.add(jobId);
      return next;
    });
  };

  const sourceLabel: Record<MigrationFormat, string> = {
    wordpress: "WordPress WXR",
    halo: "Halo JSON",
    markdown: "Markdown ZIP",
  };

  return (
    <div className="bg-white rounded-lg border border-gray-200">
      <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
        <h3 className="text-base font-semibold text-gray-900">导入任务</h3>
        <button
          onClick={fetchJobs}
          disabled={loading}
          className="px-3 py-1.5 text-sm text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
        >
          {loading ? "刷新中..." : "刷新"}
        </button>
      </div>

      {error && (
        <div className="mx-6 mt-4 p-3 bg-red-50 border border-red-200 rounded-md text-red-700 text-sm">
          {error}
        </div>
      )}

      {loading && jobs.length === 0 ? (
        <div className="py-12 text-center text-gray-500 text-sm">加载中...</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">格式</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">进度</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">条目数</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">创建时间</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">错误</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {jobs.map((job) => {
                const phaseInfo = phaseConfig[job.phase];
                const showErrors = expandedErrors.has(job.jobId);
                const progressPct = job.total > 0 ? Math.round((job.processed / job.total) * 100) : 0;
                return (
                  <>
                    <tr key={job.jobId} className="hover:bg-gray-50">
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                        {sourceLabel[job.source] || job.source}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className={`px-2 py-0.5 text-xs font-medium rounded-full ${phaseInfo.className}`}>
                          {phaseInfo.label}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap" style={{ minWidth: 140 }}>
                        <div className="flex items-center gap-2">
                          <ProgressBar value={progressPct} />
                          <span className="text-xs text-gray-500 shrink-0">{progressPct}%</span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                        {job.processed} / {job.total}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {new Date(job.startedAt).toLocaleString("zh-CN")}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        {job.errors && job.errors.length > 0 ? (
                          <button
                            onClick={() => toggleErrors(job.jobId)}
                            className="text-red-600 hover:text-red-800 font-medium"
                          >
                            {job.errors.length} 个错误 {showErrors ? "▲" : "▼"}
                          </button>
                        ) : (
                          <span className="text-gray-400">无</span>
                        )}
                      </td>
                    </tr>
                    {showErrors && job.errors && job.errors.length > 0 && (
                      <tr key={`${job.jobId}-errors`}>
                        <td colSpan={6} className="px-6 py-3 bg-red-50">
                          <div className="text-xs font-medium text-red-700 mb-2">错误详情：</div>
                          <ul className="space-y-1 max-h-40 overflow-y-auto">
                            {job.errors.map((err, i) => (
                              <li key={i} className="text-xs text-red-600 flex gap-2">
                                <span className="shrink-0 text-red-400">{i + 1}.</span>
                                <span>{err}</span>
                              </li>
                            ))}
                          </ul>
                        </td>
                      </tr>
                    )}
                  </>
                );
              })}
              {jobs.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-6 py-12 text-center text-sm text-gray-400">
                    暂无导入任务
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ---- Main Page ----
export default function AdminMigrationPage() {
  const [refreshKey, setRefreshKey] = useState(0);

  const handleJobCreated = () => {
    setRefreshKey((k) => k + 1);
  };

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-2">数据迁移</h2>
      <p className="text-sm text-gray-500 mb-6">
        从其他平台导入数据，支持 WordPress、Halo 和 Markdown 格式。
      </p>

      <ImportSection onJobCreated={handleJobCreated} />
      <JobsTable key={refreshKey} />
    </div>
  );
}
