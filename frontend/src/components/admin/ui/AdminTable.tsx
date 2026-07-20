import type { ReactNode } from "react";
import { adminTheme } from "./adminTheme";

export interface AdminTableProps {
  children: ReactNode;
  className?: string;
}

export function AdminTable({ children, className = "" }: AdminTableProps) {
  return (
    <div className={`${adminTheme.card} overflow-hidden ${className}`}>
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-slate-200 text-sm">{children}</table>
      </div>
    </div>
  );
}

export function AdminTableHead({ children }: { children: ReactNode }) {
  return <thead className="bg-slate-50/90 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">{children}</thead>;
}

export function AdminTableBody({ children }: { children: ReactNode }) {
  return <tbody className="divide-y divide-slate-100 bg-white">{children}</tbody>;
}

export function AdminTh({
  children,
  className = "",
  colSpan,
}: {
  children: ReactNode;
  className?: string;
  colSpan?: number;
}) {
  return (
    <th colSpan={colSpan} className={`px-4 py-3 whitespace-nowrap ${className}`}>
      {children}
    </th>
  );
}

export function AdminTd({
  children,
  className = "",
  colSpan,
  title,
}: {
  children?: ReactNode;
  className?: string;
  colSpan?: number;
  title?: string;
}) {
  return (
    <td colSpan={colSpan} title={title} className={`px-4 py-3 text-slate-700 ${className}`}>
      {children}
    </td>
  );
}
