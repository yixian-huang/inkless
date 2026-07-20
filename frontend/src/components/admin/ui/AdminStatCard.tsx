import type { ReactNode } from "react";
import { adminTheme } from "./adminTheme";

export interface AdminStatCardProps {
  label: string;
  value: string | number;
  icon: ReactNode;
  colorClass?: string;
  loading?: boolean;
}

export default function AdminStatCard({
  label,
  value,
  icon,
  colorClass = "bg-blue-500",
  loading = false,
}: AdminStatCardProps) {
  return (
    <div className={`${adminTheme.card} overflow-hidden`}>
      {loading ? (
        <div className="p-5">
          <div className="animate-pulse flex items-center gap-4">
            <div className="w-12 h-12 rounded-xl bg-slate-200" />
            <div className="flex-1">
              <div className="h-3 bg-slate-200 rounded w-16 mb-2" />
              <div className="h-6 bg-slate-200 rounded w-12" />
            </div>
          </div>
        </div>
      ) : (
        <div className="p-5 flex items-center gap-4">
          <div className={`${colorClass} rounded-xl p-3 shrink-0 text-white`}>{icon}</div>
          <div className="min-w-0">
            <p className="text-sm text-slate-500 truncate">{label}</p>
            <p className="text-2xl font-bold text-slate-900 tabular-nums">{value}</p>
          </div>
        </div>
      )}
    </div>
  );
}
