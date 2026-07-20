import { Fragment, useState, useEffect, useCallback, useRef } from "react";
import {
  importData,
  getMigrationJobs,
  getMigrationJob,
  retryMigrationJob,
  streamMigrationJob,
  type MigrationJob,
  type MigrationFormat,
  type MigrationJobPhase,
} from "@/api/migration";
import {
  AdminBadge,
  AdminPageHeader,
  AdminTable,
  AdminTableBody,
  AdminTableHead,
  AdminTd,
  AdminTh,
} from "@/components/admin/ui"  // AdminButton available;
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { isAxiosError } from "axios";

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

const phaseConfig: Record<MigrationJobPhase, { label: string; tone: "neutral" | "info" | "success" | "danger" }> = {
  parsing: { label: "解析中", tone: "neutral" },
  importing: { label: "导入中", tone: "info" },
  done: { label: "已完成", tone: "success" },
  failed: { label: "失败", tone: "danger" },
};

const streamReconnectDelays = [1000, 2000, 4000, 8000, 16000];

function isTerminalPhase(phase: MigrationJobPhase | undefined) {
  return phase === "done" || phase === "failed";
}

function getProgressPct(job: Pick<MigrationJob, "processed" | "total">) {
  return job.total > 0 ? Math.round((job.processed / job.total) * 100) : 0;
}

function ResultSummary({ job }: { job: MigrationJob }) {
  const totalErrors = job.errors?.length ?? 0;
  const finishedAt = job.finishedAt ? new Date(job.finishedAt).toLocaleString("zh-CN") : null;

  if (!isTerminalPhase(job.phase)) {
    return <span className="text-slate-400">导入完成后显示</span>;
  }

  return (
    <div className="space-y-1 text-sm">
      <div className="text-slate-700">
        成功 {job.succeeded} 条，失败 {job.failed} 条
      </div>
      <div className={totalErrors > 0 ? "text-red-600" : "text-slate-500"}>
        错误 {totalErrors} 个{finishedAt ? `，完成于 ${finishedAt}` : ""}
      </div>
    </div>
  );
}

// ---- Progress Bar ----
function ProgressBar({ value }: { value: number }) {
  const clamped = Math.min(100, Math.max(0, value));
  return (
    <div className="w-full bg-slate-200 rounded-full h-1.5">
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
          : "border-slate-200 hover:border-slate-300 bg-slate-50"
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
          <p className="text-sm font-medium text-slate-900">{file.name}</p>
          <p className="text-xs text-slate-500">
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
          <svg className="w-10 h-10 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
          </svg>
          <p className="text-sm font-medium text-slate-700">点击或拖放文件到此处</p>
          <p className="text-xs text-slate-400">{accept}</p>
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
  const streamRef = useRef<AbortController | null>(null);
  const timerRef = useRef<number | null>(null);
  const retryCountRef = useRef(0);
  const activeJobIdRef = useRef<string | null>(null);
  const onUpdateRef = useRef(onUpdate);
  const onDoneRef = useRef(onDone);

  useEffect(() => {
    onUpdateRef.current = onUpdate;
  }, [onUpdate]);

  useEffect(() => {
    onDoneRef.current = onDone;
  }, [onDone]);

  useEffect(() => {
    if (!jobId) return;

    let disposed = false;
    activeJobIdRef.current = jobId;
    retryCountRef.current = 0;

    const clearReconnectTimer = () => {
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };

    const closeStream = () => {
      streamRef.current?.abort();
      streamRef.current = null;
    };

    const finish = () => {
      clearReconnectTimer();
      closeStream();
      onDoneRef.current();
    };

    const connect = () => {
      if (disposed || activeJobIdRef.current !== jobId || streamRef.current) return;

      const controller = new AbortController();
      streamRef.current = controller;
      void streamMigrationJob(jobId, controller.signal, (data) => {
        onUpdateRef.current(data);
        retryCountRef.current = 0;
        if (isTerminalPhase(data.phase)) {
          finish();
        }
      }).then(() => {
        if (streamRef.current === controller) {
          streamRef.current = null;
        }
        if (!disposed && activeJobIdRef.current === jobId) {
          void scheduleReconnect();
        }
      }).catch((streamError: unknown) => {
        if (streamRef.current === controller) {
          streamRef.current = null;
        }
        if (
          !disposed &&
          activeJobIdRef.current === jobId &&
          !(streamError instanceof DOMException && streamError.name === "AbortError")
        ) {
          void scheduleReconnect();
        }
      });
    };

    const scheduleNext = (callback: () => void) => {
      const delay = streamReconnectDelays[
        Math.min(retryCountRef.current, streamReconnectDelays.length - 1)
      ];
      retryCountRef.current += 1;
      timerRef.current = window.setTimeout(() => {
        timerRef.current = null;
        callback();
      }, delay);
    };

    const scheduleReconnect = async () => {
      closeStream();
      clearReconnectTimer();

      try {
        const current = await getMigrationJob(jobId);
        if (disposed || activeJobIdRef.current !== jobId) return;

        onUpdateRef.current(current);
        if (isTerminalPhase(current.phase)) {
          finish();
          return;
        }

        if (current.phase !== "importing") return;
      } catch {
        if (disposed || activeJobIdRef.current !== jobId) return;
        scheduleNext(scheduleReconnect);
        return;
      }

      scheduleNext(connect);
    };

    connect();

    return () => {
      disposed = true;
      clearReconnectTimer();
      closeStream();
    };
  }, [jobId]);

  return streamRef;
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
      const parseSummary = result.parseErrors.length > 0
        ? `，解析提示 ${result.parseErrors.length} 条`
        : "";
      setSuccessMsg(`导入任务已创建，任务 ID: ${result.jobId}，待导入 ${result.totalArticles} 条${parseSummary}`);
      setFile(null);
      onJobCreated();
    } catch (importError) {
      const message = isAxiosError(importError)
        ? importError.response?.data?.error?.message
        : null;
      const parseErrors = isAxiosError(importError) && Array.isArray(importError.response?.data?.parseErrors)
        ? importError.response.data.parseErrors.filter((item): item is string => typeof item === "string")
        : [];
      setError(
        [
          message || "创建导入任务失败，请检查文件格式后重试",
          parseErrors.length > 0 ? parseErrors.join("；") : "",
        ].filter(Boolean).join("："),
      );
    } finally {
      setImporting(false);
    }
  };

  return (
    <div className="rounded-2xl border border-slate-200/80 bg-white p-6 mb-6 shadow-[0_1px_2px_rgba(15,23,42,0.04)]">
      <h3 className="text-base font-semibold text-slate-900 mb-4">导入数据</h3>

      {/* Format selector */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-slate-700 mb-2">选择格式</label>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          {formatOptions.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => { setFormat(opt.value); setFile(null); }}
              className={`text-left p-3 rounded-lg border-2 transition-colors ${
                format === opt.value
                  ? "border-blue-500 bg-blue-50"
                  : "border-slate-200 hover:border-slate-200"
              }`}
            >
              <div className="text-sm font-medium text-slate-900">{opt.label}</div>
              <div className="text-xs text-slate-500 mt-1 line-clamp-2">{opt.description}</div>
            </button>
          ))}
        </div>
      </div>

      {/* Format description */}
      <div className="mb-4 p-3 bg-blue-50 border border-blue-100 rounded-xl text-sm text-blue-800">
        <strong>{selectedFormat.label}：</strong>{selectedFormat.description}
      </div>

      {/* File dropzone */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-slate-700 mb-2">上传文件</label>
        <FileDropzone
          accept={acceptMap[format]}
          file={file}
          onFileChange={setFile}
        />
      </div>

      {successMsg && (
        <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-xl text-green-700 text-sm">
          {successMsg}
        </div>
      )}
      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-xl text-red-700 text-sm">
          {error}
        </div>
      )}

      <button
        onClick={handleImport}
        disabled={!file || importing}
        className="px-5 py-2 bg-blue-600 text-white text-sm font-medium rounded-xl hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
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
  const [retryingJobId, setRetryingJobId] = useState<string | null>(null);

  const fetchJobs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getMigrationJobs();
      setJobs(data);
      // Auto-attach SSE to any running job
      const running = data.find((j) => j.phase === "importing");
      setStreamJobId(running?.jobId ?? null);
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

  const handleRetry = async (jobId: string) => {
    setRetryingJobId(jobId);
    setError(null);
    try {
      const retried = await retryMigrationJob(jobId);
      setJobs((prev) => prev.map((job) => (job.jobId === jobId ? retried : job)));
      if (retried.phase === "importing") {
        setStreamJobId(retried.jobId);
      } else if (isTerminalPhase(retried.phase)) {
        fetchJobs();
      }
    } catch {
      setError("重试迁移任务失败");
    } finally {
      setRetryingJobId(null);
    }
  };

  const sourceLabel: Record<MigrationFormat, string> = {
    wordpress: "WordPress WXR",
    halo: "Halo JSON",
    markdown: "Markdown ZIP",
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-base font-semibold text-slate-900">导入任务</h3>
        <button
          onClick={fetchJobs}
          disabled={loading}
          className="px-3 py-1.5 text-sm text-slate-600 border border-slate-300 rounded-xl hover:bg-slate-50 disabled:opacity-50"
        >
          {loading ? "刷新中..." : "刷新"}
        </button>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-red-700 text-sm">
          {error}
        </div>
      )}

      {loading && jobs.length === 0 ? (
        <div className="py-12 text-center text-slate-500 text-sm">加载中...</div>
      ) : jobs.length === 0 ? (
        <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50/80 px-6 py-12 text-center">
          <div className="text-sm font-medium text-slate-700">暂无导入任务</div>
          <div className="mt-1 text-sm text-slate-400">
            上传 WordPress、Halo 或 Markdown 文件后，导入进度和结果会显示在这里。
          </div>
        </div>
      ) : (
        <AdminTable>
          <AdminTableHead>
            <tr>
              <AdminTh>格式</AdminTh>
              <AdminTh>状态</AdminTh>
              <AdminTh>进度</AdminTh>
              <AdminTh>条目数</AdminTh>
              <AdminTh>结果</AdminTh>
              <AdminTh>创建时间</AdminTh>
              <AdminTh>错误</AdminTh>
              <AdminTh>操作</AdminTh>
            </tr>
          </AdminTableHead>
          <AdminTableBody>
            {jobs.map((job) => {
              const phaseInfo = phaseConfig[job.phase];
              const showErrors = expandedErrors.has(job.jobId);
              const progressPct = getProgressPct(job);
              return (
                <Fragment key={job.jobId}>
                  <tr className="hover:bg-slate-50/80">
                    <AdminTd className="whitespace-nowrap text-slate-900">
                      {sourceLabel[job.source] || job.source}
                    </AdminTd>
                    <AdminTd className="whitespace-nowrap">
                      <AdminBadge tone={phaseInfo.tone}>{phaseInfo.label}</AdminBadge>
                    </AdminTd>
                    <AdminTd className="whitespace-nowrap">
                      <div className="flex min-w-[140px] items-center gap-2">
                        <ProgressBar value={progressPct} />
                        <span className="text-xs text-slate-500 shrink-0">{progressPct}%</span>
                      </div>
                    </AdminTd>
                    <AdminTd className="whitespace-nowrap text-slate-600">
                      {job.processed} / {job.total}
                    </AdminTd>
                    <AdminTd className="text-slate-600">
                      <ResultSummary job={job} />
                    </AdminTd>
                    <AdminTd className="whitespace-nowrap text-slate-500">
                      {new Date(job.startedAt).toLocaleString("zh-CN")}
                    </AdminTd>
                    <AdminTd className="whitespace-nowrap">
                      {job.errors && job.errors.length > 0 ? (
                        <button
                          onClick={() => toggleErrors(job.jobId)}
                          className="text-red-600 hover:text-red-800 font-medium text-sm"
                        >
                          {job.errors.length} 个错误 {showErrors ? "▲" : "▼"}
                        </button>
                      ) : (
                        <span className="text-slate-400">无</span>
                      )}
                    </AdminTd>
                    <AdminTd className="whitespace-nowrap">
                      {job.phase === "failed" && job.retryable !== false ? (
                        <button
                          onClick={() => handleRetry(job.jobId)}
                          disabled={retryingJobId === job.jobId}
                          className="px-3 py-1.5 text-sm text-blue-600 border border-blue-200 rounded-xl hover:bg-blue-50 disabled:opacity-50"
                        >
                          {retryingJobId === job.jobId ? "重试中..." : "重试"}
                        </button>
                      ) : (
                        <span className="text-slate-400">-</span>
                      )}
                    </AdminTd>
                  </tr>
                  {showErrors && job.errors && job.errors.length > 0 && (
                    <tr>
                      <AdminTd colSpan={8} className="bg-red-50">
                        <div className="text-xs font-medium text-red-700 mb-2">错误详情：</div>
                        <ul className="space-y-1 max-h-40 overflow-y-auto">
                          {job.errors.map((err, i) => (
                            <li key={i} className="text-xs text-red-600 flex gap-2">
                              <span className="shrink-0 text-red-400">{i + 1}.</span>
                              <span>{err}</span>
                            </li>
                          ))}
                        </ul>
                      </AdminTd>
                    </tr>
                  )}
                </Fragment>
              );
            })}
          </AdminTableBody>
        </AdminTable>
      )}
    </div>
  );
}

// ---- Main Page ----
export default function AdminMigrationPage() {
  useDocumentTitle("数据迁移");
  const [refreshKey, setRefreshKey] = useState(0);

  const handleJobCreated = () => {
    setRefreshKey((k) => k + 1);
  };

  return (
    <div>
      <AdminPageHeader
        title="数据迁移"
        description="从其他平台导入数据，支持 WordPress、Halo 和 Markdown 格式。"
      />

      <ImportSection onJobCreated={handleJobCreated} />
      <JobsTable key={refreshKey} />
    </div>
  );
}
