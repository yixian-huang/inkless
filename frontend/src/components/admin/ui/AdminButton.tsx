import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import { adminTheme } from "./adminTheme";

type Variant = "primary" | "secondary" | "danger" | "ghost" | "soft";
type Size = "sm" | "md" | "lg";

export interface AdminButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
  children: ReactNode;
}

/** Ink-press buttons: primary = solid ink, secondary = paper edge. */
const variantClass: Record<Variant, string> = {
  primary:
    "bg-[#1a1814] text-[#f7f3ec] border border-transparent shadow-[0_1px_2px_rgba(26,24,20,0.18)] hover:bg-[#2a2622] active:bg-[#0f0e0c]",
  secondary:
    "bg-[#fbfaf7] text-[#3d3832] border border-[#e4ddd2] shadow-[0_1px_0_rgba(26,24,20,0.04)] hover:bg-[#f5f1ea] hover:border-[#d4cbbf] active:bg-[#f0ebe3]",
  danger:
    "bg-[#9b3b2e] text-[#faf0ee] border border-transparent shadow-[0_1px_2px_rgba(155,59,46,0.2)] hover:bg-[#863227] active:bg-[#6f2a21]",
  ghost:
    "bg-transparent text-[#5c564f] border border-transparent hover:bg-[#f0ebe3]/90 hover:text-[#1a1814]",
  soft: "bg-[#f0ebe3] text-[#1a1814] border border-[#d4cbbf] hover:bg-[#e8e2d8] active:bg-[#e0d9cd]",
};

const sizeClass: Record<Size, string> = {
  sm: "h-8 px-3 text-xs rounded-lg tracking-wide",
  md: "h-9 px-3.5 text-sm rounded-lg tracking-wide",
  lg: "h-11 px-5 text-sm rounded-lg tracking-wide",
};

const AdminButton = forwardRef<HTMLButtonElement, AdminButtonProps>(function AdminButton(
  { variant = "primary", size = "md", className = "", disabled, children, type = "button", ...rest },
  ref,
) {
  return (
    <button
      ref={ref}
      type={type}
      disabled={disabled}
      className={`inline-flex items-center justify-center gap-1.5 font-medium ${adminTheme.transition} ${adminTheme.focusRing} ${variantClass[variant]} ${sizeClass[size]} ${
        disabled ? "opacity-50 cursor-not-allowed shadow-none hover:shadow-none" : ""
      } ${className}`}
      {...rest}
    >
      {children}
    </button>
  );
});

export default AdminButton;
