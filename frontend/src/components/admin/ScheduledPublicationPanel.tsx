import { memo, useCallback, useMemo, useState } from "react";
import type { ScheduledPublication } from "@/api/scheduledPublications";
import {
  dateTimeLocalToISOString,
  toDateTimeLocalValue,
} from "@/api/scheduledPublications";
import SchedulePublishModal from "@/components/admin/SchedulePublishModal";

interface ScheduledPublicationPanelProps {
  item: ScheduledPublication | null;
  loading?: boolean;
  busy?: boolean;
  canPublish: boolean;
  disabledReason?: string;
  title?: string;
  /** Compact single-line toolbar style (default true for article editor). */
  compact?: boolean;
  onSchedule: (scheduledAt: string) => Promise<void> | void;
  onCancel: () => Promise<void> | void;
  onRetry?: () => Promise<void> | void;
  onRefresh?: () => Promise<void> | void;
}

const statusClasses: Record<string, string> = {
  pending: "bg-blue-50 text-blue-700 border-blue-200",
  running: "bg-indigo-50 text-indigo-700 border-indigo-200",
  succeeded: "bg-green-50 text-green-700 border-green-200",
  failed: "bg-red-50 text-red-700 border-red-200",
  cancelled: "bg-gray-50 text-gray-600 border-gray-200",
};

const statusLabels: Record<string, string> = {
  pending: "等待发布",
  running: "发布中",
  succeeded: "已发布",
  failed: "发布失败",
  cancelled: "已取消",
};

function formatDateTime(value: string | null | undefined) {
  if (!value) return "—";
  try {
    return new Date(value).toLocaleString("zh-CN", {
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return value;
  }
}

export const ScheduledPublicationPanel = memo(function ScheduledPublicationPanel({
  item,
  loading = false,
  busy = false,
  canPublish,
  disabledReason,
  title = "定时发布",
  compact = false,
  onSchedule,
  onCancel,
  onRetry,
  onRefresh,
}: ScheduledPublicationPanelProps) {
  const [modalOpen, setModalOpen] = useState(false);
  const [localError, setLocalError] = useState("");

  const currentSchedule = useMemo(
    () => toDateTimeLocalValue(item?.status === "pending" ? item.scheduledAt : null),
    [item?.scheduledAt, item?.status],
  );

  const handleSchedule = useCallback(
    async (dateValue: string) => {
      setLocalError("");
      try {
        if (!dateValue) {
          await onCancel();
          setModalOpen(false);
          return;
        }
        await onSchedule(dateTimeLocalToISOString(dateValue));
        setModalOpen(false);
      } catch (error) {
        setLocalError(error instanceof Error ? error.message : "定时发布失败");
      }
    },
    [onCancel, onSchedule],
  );

  const isRunning = item?.status === "running";
  const canAct = canPublish && !busy && !isRunning;
  const hasPending = item?.status === "pending";
  const hasFailed = item?.status === "failed";

  const scheduleLabel = isRunning ? "发布中" : hasPending ? "改期" : compact ? "定时" : "安排发布";

  const actions = (
    <>
      {onRefresh && (
        <button
          type="button"
          onClick={onRefresh}
          disabled={busy}
          className="rounded border border-gray-300 px-2 py-1 text-gray-600 hover:bg-gray-50 disabled:opacity-50"
          title="刷新定时发布状态"
        >
          刷新
        </button>
      )}
      {canPublish && (
        <button
          type="button"
          onClick={() => setModalOpen(true)}
          disabled={!canAct}
          className="rounded bg-blue-600 px-2.5 py-1 text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {scheduleLabel}
        </button>
      )}
      {canPublish && hasPending && (
        <button
          type="button"
          onClick={onCancel}
          disabled={!canAct}
          className="rounded border border-red-200 px-2 py-1 text-red-600 hover:bg-red-50 disabled:opacity-50"
        >
          取消
        </button>
      )}
      {canPublish && hasFailed && onRetry && (
        <button
          type="button"
          onClick={onRetry}
          disabled={!canAct}
          className="rounded border border-orange-300 px-2 py-1 text-orange-700 hover:bg-orange-50 disabled:opacity-50"
        >
          重试
        </button>
      )}
    </>
  );

  const modal = (
    <SchedulePublishModal
      open={modalOpen}
      onClose={() => setModalOpen(false)}
      onSchedule={handleSchedule}
      currentSchedule={currentSchedule}
      submitting={busy}
    />
  );

  if (compact) {
    return (
      <div className="flex items-center gap-2 text-xs min-w-0">
        <span className="text-gray-500 flex-shrink-0">{title}</span>
        {loading ? (
          <span className="text-gray-400">…</span>
        ) : item ? (
          <>
            <span className={`rounded-full border px-2 py-0.5 flex-shrink-0 ${statusClasses[item.status] ?? statusClasses.cancelled}`}>
              {statusLabels[item.status] ?? item.status}
            </span>
            <span className="text-gray-700 truncate" title={formatDateTime(item.scheduledAt)}>
              {formatDateTime(item.scheduledAt)}
            </span>
            {hasFailed && item.lastError && (
              <span className="text-red-600 truncate max-w-[10rem]" title={item.lastError}>
                {item.lastError}
              </span>
            )}
          </>
        ) : (
          <span className="text-gray-400">未安排</span>
        )}
        <div className="flex items-center gap-1.5 flex-shrink-0">{actions}</div>
        {localError && <span className="text-red-600 truncate max-w-[12rem]" title={localError}>{localError}</span>}
        {!canPublish && disabledReason && (
          <span className="text-yellow-700 truncate max-w-[12rem]" title={disabledReason}>无权限</span>
        )}
        {modal}
      </div>
    );
  }

  return (
    <section className="rounded-md border border-gray-200 bg-white p-3 text-xs">
      <div className="mb-2 flex items-center justify-between gap-2">
        <h3 className="font-semibold text-gray-800">{title}</h3>
        {item && (
          <span className={`rounded-full border px-2 py-0.5 ${statusClasses[item.status] ?? statusClasses.cancelled}`}>
            {statusLabels[item.status] ?? item.status}
          </span>
        )}
      </div>

      {loading ? (
        <p className="text-gray-500">加载定时发布状态...</p>
      ) : item ? (
        <div className="space-y-1.5 text-gray-600">
          <div className="flex justify-between gap-3">
            <span className="text-gray-400">计划时间</span>
            <span className="text-right font-medium text-gray-800">{formatDateTime(item.scheduledAt)}</span>
          </div>
          {typeof item.expectedVersion === "number" && (
            <div className="flex justify-between gap-3">
              <span className="text-gray-400">目标版本</span>
              <span className="font-medium text-gray-800">{item.expectedVersion}</span>
            </div>
          )}
          {hasFailed && item.lastError && (
            <p className="rounded border border-red-200 bg-red-50 p-2 text-red-700">
              {item.lastError}
            </p>
          )}
        </div>
      ) : (
        <p className="text-gray-500">当前没有待发布任务。</p>
      )}

      {localError && (
        <p className="mt-2 rounded border border-red-200 bg-red-50 p-2 text-red-700">{localError}</p>
      )}
      {!canPublish && disabledReason && (
        <p className="mt-2 rounded border border-yellow-200 bg-yellow-50 p-2 text-yellow-700">
          {disabledReason}
        </p>
      )}

      <div className="mt-3 flex flex-wrap items-center gap-2">{actions}</div>
      {modal}
    </section>
  );
});
