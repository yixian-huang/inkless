import type { ReactNode } from "react";

type Tone = "success" | "warning" | "neutral" | "info" | "danger";

export interface AdminBadgeProps {
  children: ReactNode;
  tone?: Tone;
  className?: string;
}

const toneClass: Record<Tone, string> = {
  success: "bg-[#eef5ef] text-[#2f5d3a] ring-[#d5e5d8]",
  warning: "bg-[#faf4e8] text-[#7a5b22] ring-[#eadfc4]",
  neutral: "bg-[#f0ebe3] text-[#5c564f] ring-[#e4ddd2]",
  info: "bg-[#eef2f4] text-[#2f4a5c] ring-[#d5e0e6]",
  danger: "bg-[#faf0ee] text-[#8b3a32] ring-[#ebd4cf]",
};

export default function AdminBadge({ children, tone = "neutral", className = "" }: AdminBadgeProps) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium tracking-wide ring-1 ring-inset ${toneClass[tone]} ${className}`}
    >
      {children}
    </span>
  );
}
