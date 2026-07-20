import type { ReactNode } from "react";

type Tone = "success" | "warning" | "neutral" | "info" | "danger";

export interface AdminBadgeProps {
  children: ReactNode;
  tone?: Tone;
  className?: string;
}

const toneClass: Record<Tone, string> = {
  success: "bg-emerald-50 text-emerald-800 ring-emerald-600/15",
  warning: "bg-amber-50 text-amber-800 ring-amber-600/15",
  neutral: "bg-slate-100 text-slate-600 ring-slate-500/10",
  info: "bg-blue-50 text-blue-800 ring-blue-600/15",
  danger: "bg-red-50 text-red-800 ring-red-600/15",
};

export default function AdminBadge({ children, tone = "neutral", className = "" }: AdminBadgeProps) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${toneClass[tone]} ${className}`}
    >
      {children}
    </span>
  );
}
