import type { ReactNode } from "react";

export interface AdminEmptyStateProps {
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}

export default function AdminEmptyState({
  title,
  description,
  action,
  className = "",
}: AdminEmptyStateProps) {
  return (
    <div
      className={`flex flex-col items-center justify-center rounded-xl border border-dashed border-slate-200 bg-slate-50/80 px-6 py-12 text-center ${className}`}
    >
      <p className="text-sm font-medium text-slate-700">{title}</p>
      {description ? <p className="mt-1 max-w-sm text-sm text-slate-500">{description}</p> : null}
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  );
}
