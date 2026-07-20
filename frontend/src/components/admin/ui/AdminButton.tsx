import type { ButtonHTMLAttributes, ReactNode } from "react";
import { adminTheme } from "./adminTheme";

type Variant = "primary" | "secondary" | "danger" | "ghost";
type Size = "sm" | "md";

export interface AdminButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
  children: ReactNode;
}

const variantClass: Record<Variant, string> = {
  primary: "bg-blue-600 text-white hover:bg-blue-700 border border-transparent shadow-sm",
  secondary: "bg-white text-slate-700 hover:bg-slate-50 border border-slate-200 shadow-sm",
  danger: "bg-red-600 text-white hover:bg-red-700 border border-transparent shadow-sm",
  ghost: "bg-transparent text-slate-600 hover:bg-slate-100 border border-transparent",
};

const sizeClass: Record<Size, string> = {
  sm: "px-3 py-1.5 text-sm",
  md: "px-4 py-2 text-sm",
};

export default function AdminButton({
  variant = "primary",
  size = "md",
  className = "",
  disabled,
  children,
  ...rest
}: AdminButtonProps) {
  return (
    <button
      type="button"
      disabled={disabled}
      className={`inline-flex items-center justify-center gap-1.5 rounded-lg font-medium transition-colors ${adminTheme.focusRing} ${variantClass[variant]} ${sizeClass[size]} ${
        disabled ? "opacity-50 cursor-not-allowed" : ""
      } ${className}`}
      {...rest}
    >
      {children}
    </button>
  );
}
