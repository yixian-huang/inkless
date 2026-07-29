import { useCallback, useEffect, useState } from "react";
import type { ReactNode } from "react";
import {
  applyHostUpdate,
  checkHostUpdate,
  getHostUpdateStatus,
  getSystemStatus,
  rollbackHostUpdate,
  type HostUpdateJob,
  type HostUpdateStatus,
  type SystemStatusResponse,
} from "@/api/systemStatus";
import {
  AdminButton,
  AdminErrorBanner,
  AdminLoading,
  AdminPageHeader,
  AdminSuccessBanner,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

interface Metric {
  label: string;
  value: string | number;
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat("zh-CN").format(value);
}

function formatMB(value: number): string {
  return `${value.toFixed(2)} MB`;
}

function formatBytes(value: number): string {
  if (value >= 1024 * 1024 * 1024) return `${(value / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (value >= 1024 * 1024) return `${(value / 1024 / 1024).toFixed(2)} MB`;
  if (value >= 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${value} B`;
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days} 天 ${hours} 小时`;
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟`;
  return `${minutes} 分钟`;
}

function errorMessage(error: unknown, fallback = "系统状态加载失败"): string {
  if (typeof error === "object" && error && "response" in error) {
    const data = (error as { response?: { data?: { error?: string | { message?: string } } } })
      .response?.data;
    if (typeof data?.error === "string") return data.error;
    if (data?.error && typeof data.error === "object" && data.error.message) {
      return data.error.message;
    }
    // apply may return job body on error
    const job = data as HostUpdateJob | undefined;
    if (job && typeof job === "object" && "error" in job && (job as HostUpdateJob).error) {
      return String((job as HostUpdateJob).error);
    }
  }
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

function StatusBadge({ healthy, status }: { healthy: boolean; status: string }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium ${
        healthy ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"
      }`}
    >
      {healthy ? "正常" : status || "异常"}
    </span>
  );
}

function InfoCard({
  title,
  children,
  action,
}: {
  title: string;
  children: ReactNode;
  action?: ReactNode;
}) {
  return (
    <section className="bg-white rounded-lg border border-gray-200 p-5">
      <div className="flex items-center justify-between gap-3 mb-4">
        <h2 className="text-base font-semibold text-gray-900">{title}</h2>
        {action}
      </div>
      {children}
    </section>
  );
}

function MetricGrid({ metrics }: { metrics: Metric[] }) {
  return (
    <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4">
      {metrics.map((metric) => (
        <div key={metric.label} className="rounded-lg bg-gray-50 px-4 py-3">
          <dt className="text-xs font-medium text-gray-500">{metric.label}</dt>
          <dd className="mt-1 text-sm font-semibold text-gray-900 break-words">{metric.value}</dd>
        </div>
      ))}
    </dl>
  );
}

function HealthMessage({ message }: { message?: string }) {
  if (!message) return null;
  return <p className="mt-4 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">{message}</p>;
}

export default function AdminSystemStatusPage() {
  useDocumentTitle("系统状态");
  const [status, setStatus] = useState<SystemStatusResponse | null>(null);
  const [update, setUpdate] = useState<HostUpdateStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [checking, setChecking] = useState(false);
  const [applying, setApplying] = useState(false);
  const [rollingBack, setRollingBack] = useState(false);
  const [lastJob, setLastJob] = useState<HostUpdateJob | null>(null);

  const fetchStatus = useCallback(async (manual = false) => {
    if (manual) {
      setRefreshing(true);
    } else {
      setLoading(true);
    }
    setError("");
    try {
      const [st, up] = await Promise.all([
        getSystemStatus(),
        getHostUpdateStatus().catch(() => null),
      ]);
      setStatus(st);
      setUpdate(up);
      if (up?.lastJob) setLastJob(up.lastJob);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  async function handleCheck() {
    setChecking(true);
    setError("");
    setSuccess("");
    try {
      const res = await checkHostUpdate();
      if (res.error) {
        setError(res.error);
      } else if (res.latest) {
        setSuccess(
          res.latest.newer
            ? `发现新版本 ${res.latest.version}`
            : `已是最新（远端 ${res.latest.version}）`,
        );
      } else {
        setSuccess("检查完成，未解析到远端版本");
      }
      const up = await getHostUpdateStatus();
      setUpdate(up);
    } catch (err) {
      setError(errorMessage(err, "检查更新失败"));
    } finally {
      setChecking(false);
    }
  }

  async function handleApply() {
    if (!update?.latest?.version) return;
    const ver = update.latest.version;
    if (!window.confirm(`将本实例升级到 ${ver}？仅影响当前站点进程，不会改其他域名的数据。`)) {
      return;
    }
    setApplying(true);
    setError("");
    setSuccess("");
    try {
      const job = await applyHostUpdate(ver);
      setLastJob(job);
      if (job.status === "success") {
        setSuccess(job.message || `已升级到 ${job.toVersion}`);
      } else if (job.status === "pending_restart") {
        setSuccess(job.message || "代码已切换，需手动 systemctl restart");
      } else if (job.error) {
        setError(job.error);
      }
      await fetchStatus(true);
    } catch (err) {
      setError(errorMessage(err, "应用更新失败"));
    } finally {
      setApplying(false);
    }
  }

  async function handleRollback() {
    if (!window.confirm("回滚到 previous 版本？仅本实例。")) return;
    setRollingBack(true);
    setError("");
    setSuccess("");
    try {
      const job = await rollbackHostUpdate("previous");
      setLastJob(job);
      if (job.status === "success") {
        setSuccess(`已回滚到 ${job.toVersion || "previous"}`);
      } else if (job.error) {
        setError(job.error);
      } else if (job.message) {
        setSuccess(job.message);
      }
      await fetchStatus(true);
    } catch (err) {
      setError(errorMessage(err, "回滚失败"));
    } finally {
      setRollingBack(false);
    }
  }

  return (
    <div className="space-y-6">
      <AdminPageHeader
        title="系统状态"
        description="查看应用版本、数据库、存储、运行时和内容统计；可选检查 Host Release 并一键升级本实例。"
        actions={
          <AdminButton
            size="sm"
            onClick={() => fetchStatus(true)}
            disabled={loading || refreshing}
          >
            {refreshing ? "刷新中…" : "刷新"}
          </AdminButton>
        }
      />

      {error && <AdminErrorBanner message={error} onDismiss={() => setError("")} />}
      {success && <AdminSuccessBanner message={success} />}

      {loading ? (
        <AdminLoading />
      ) : status ? (
        <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
          <InfoCard title="版本">
            <MetricGrid
              metrics={[
                { label: "应用版本", value: status.application.version || "dev" },
                { label: "Go 版本", value: status.runtime.goVersion },
              ]}
            />
          </InfoCard>

          <InfoCard title="关于与更新">
            {update ? (
              <div className="space-y-3 text-sm text-gray-700">
                <MetricGrid
                  metrics={[
                    { label: "当前版本", value: update.currentVersion || status.application.version || "—" },
                    {
                      label: "远端最新",
                      value: update.latest?.version
                        ? `${update.latest.version}${update.latest.newer ? "（可更新）" : ""}`
                        : "尚未检查",
                    },
                    { label: "通道", value: update.channel || "stable" },
                    {
                      label: "自更新",
                      value: update.enabled
                        ? update.capable
                          ? "已开启且可用"
                          : "已开启但不可用"
                        : "关闭",
                    },
                  ]}
                />
                {update.blockedReason ? (
                  <p className="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-800">
                    {update.blockedReason}
                  </p>
                ) : null}
                {update.latest?.notesUrl ? (
                  <p className="text-xs">
                    <a
                      href={update.latest.notesUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:underline"
                    >
                      Release 说明
                    </a>
                    {update.lastCheckAt ? ` · 上次检查 ${update.lastCheckAt}` : null}
                  </p>
                ) : update.lastCheckAt ? (
                  <p className="text-xs text-gray-500">上次检查 {update.lastCheckAt}</p>
                ) : null}
                <div className="flex flex-wrap gap-2 pt-1">
                  <AdminButton size="sm" variant="secondary" disabled={checking || applying} onClick={handleCheck}>
                    {checking ? "检查中…" : "检查更新"}
                  </AdminButton>
                  <AdminButton
                    size="sm"
                    variant="primary"
                    disabled={
                      applying ||
                      checking ||
                      !update.capable ||
                      !update.latest?.newer
                    }
                    onClick={handleApply}
                  >
                    {applying
                      ? "升级中…"
                      : update.latest?.newer
                        ? `升级到 ${update.latest.version}`
                        : "已是最新"}
                  </AdminButton>
                  <AdminButton
                    size="sm"
                    variant="secondary"
                    disabled={rollingBack || !update.capable || !update.hasPrevious}
                    onClick={handleRollback}
                  >
                    {rollingBack ? "回滚中…" : "回滚 previous"}
                  </AdminButton>
                </div>
                {!update.enabled ? (
                  <p className="text-xs text-gray-500">
                    默认关闭。在本实例 .env 设置 INKLESS_SELF_UPDATE_ENABLED=true、
                    INKLESS_RELEASE_ROOT、INKLESS_SYSTEMD_UNIT 后重启进程；仅升级本站代码树。
                  </p>
                ) : null}
                {(lastJob || update.lastJob) && (
                  <p className="font-mono text-[11px] text-gray-500">
                    最近任务：{(lastJob || update.lastJob)?.status}
                    {(lastJob || update.lastJob)?.toVersion
                      ? ` → ${(lastJob || update.lastJob)?.toVersion}`
                      : ""}
                    {(lastJob || update.lastJob)?.phase
                      ? ` · ${(lastJob || update.lastJob)?.phase}`
                      : ""}
                    {(lastJob || update.lastJob)?.error
                      ? ` · ${(lastJob || update.lastJob)?.error}`
                      : ""}
                  </p>
                )}
              </div>
            ) : (
              <p className="text-sm text-gray-500">无法加载更新状态（后端未部署或权限不足）。</p>
            )}
          </InfoCard>

          <InfoCard title="数据库" action={<StatusBadge healthy={status.database.healthy} status={status.database.status} />}>
            <MetricGrid
              metrics={[
                { label: "类型", value: status.database.type },
                { label: "打开连接", value: formatNumber(status.database.openConnections) },
                { label: "使用中", value: formatNumber(status.database.inUse) },
                { label: "空闲", value: formatNumber(status.database.idle) },
                { label: "最大打开连接", value: formatNumber(status.database.maxOpenConnections) },
              ]}
            />
            <HealthMessage message={status.database.error} />
          </InfoCard>

          <InfoCard title="存储" action={<StatusBadge healthy={status.storage.healthy} status={status.storage.status} />}>
            <MetricGrid
              metrics={[
                { label: "类型", value: status.storage.type === "local" ? "本地存储" : status.storage.type },
                { label: "媒体文件", value: formatNumber(status.storage.mediaCount) },
                { label: "使用空间", value: formatBytes(status.storage.uploadDirBytes) },
                { label: "使用空间 (MB)", value: formatMB(status.storage.uploadDirSizeMB) },
              ]}
            />
            <HealthMessage message={status.storage.error} />
          </InfoCard>

          <InfoCard title="运行时">
            <MetricGrid
              metrics={[
                { label: "系统", value: `${status.runtime.os}/${status.runtime.arch}` },
                { label: "CPU", value: formatNumber(status.runtime.cpuCount) },
                { label: "协程", value: formatNumber(status.runtime.goroutines) },
                { label: "运行时间", value: formatUptime(status.runtime.uptime) },
                { label: "当前内存", value: formatMB(status.memory.allocMB) },
                { label: "系统内存", value: formatMB(status.memory.sysMB) },
                { label: "累计分配", value: formatMB(status.memory.totalAllocMB) },
                { label: "最近 GC 暂停", value: `${status.memory.gcPauseMs.toFixed(2)} ms` },
              ]}
            />
          </InfoCard>

          <InfoCard title="内容">
            <MetricGrid
              metrics={[
                { label: "统一页面", value: formatNumber(status.content.pages) },
                { label: "文章", value: formatNumber(status.content.articles) },
                { label: "媒体", value: formatNumber(status.content.media) },
                { label: "用户", value: formatNumber(status.content.users) },
              ]}
            />
          </InfoCard>
        </div>
      ) : (
        <div className="bg-white rounded-lg border border-gray-200 p-8 text-center text-gray-500">暂无状态数据</div>
      )}
    </div>
  );
}
