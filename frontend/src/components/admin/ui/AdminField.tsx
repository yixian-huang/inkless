import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from "react";
import { adminTheme } from "./adminTheme";

function cx(...parts: Array<string | false | undefined | null>) {
  return parts.filter(Boolean).join(" ");
}

export function AdminLabel({
  children,
  htmlFor,
  className = "",
}: {
  children: ReactNode;
  htmlFor?: string;
  className?: string;
}) {
  return (
    <label htmlFor={htmlFor} className={cx("mb-1.5 block", adminTheme.label, className)}>
      {children}
    </label>
  );
}

export function AdminHint({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <p className={cx("mt-1 text-xs text-slate-500", className)}>{children}</p>;
}

export function AdminField({
  label,
  htmlFor,
  hint,
  children,
  className = "",
}: {
  label?: ReactNode;
  htmlFor?: string;
  hint?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={className}>
      {label != null && label !== "" ? (
        typeof label === "string" ? (
          <AdminLabel htmlFor={htmlFor}>{label}</AdminLabel>
        ) : (
          <div className={cx("mb-1.5", adminTheme.label)}>{label}</div>
        )
      ) : null}
      {children}
      {hint ? <AdminHint>{hint}</AdminHint> : null}
    </div>
  );
}

export function AdminInput({ className = "", ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cx(adminTheme.input, className)} {...rest} />;
}

export function AdminSelect({
  className = "",
  children,
  ...rest
}: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select className={cx(adminTheme.select, className)} {...rest}>
      {children}
    </select>
  );
}

export function AdminTextarea({
  className = "",
  ...rest
}: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cx(adminTheme.textarea, className)} {...rest} />;
}

export function AdminCheckbox({
  className = "",
  label,
  ...rest
}: InputHTMLAttributes<HTMLInputElement> & { label?: ReactNode }) {
  const input = (
    <input type="checkbox" className={cx(adminTheme.checkbox, className)} {...rest} />
  );
  if (!label) return input;
  return (
    <label className="inline-flex cursor-pointer items-center gap-2 text-sm text-slate-700">
      {input}
      <span>{label}</span>
    </label>
  );
}

export function AdminToolbar({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cx(adminTheme.toolbar, className)}>{children}</div>;
}

export function AdminFilterChip({
  active = false,
  children,
  className = "",
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & { active?: boolean }) {
  return (
    <button
      type="button"
      className={cx(
        "inline-flex items-center rounded-full border px-3 py-1.5 text-xs font-medium transition-all",
        active
          ? "border-[#1a1814] bg-[#1a1814] text-[#f7f3ec] shadow-sm"
          : "border-[#e4ddd2] bg-[#fbfaf7] text-[#5c564f] hover:bg-[#f5f1ea] hover:border-[#d4cbbf]",
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

/** Inline text action used in table rows / card footers. */
export function AdminTextButton({
  tone = "primary",
  className = "",
  children,
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  tone?: "primary" | "danger" | "muted";
}) {
  const tones = {
    primary: "text-[#1a1814] hover:text-[#3d3832] underline-offset-2 hover:underline decoration-[#d4cbbf]",
    danger: "text-[#9b3b2e] hover:text-[#6f2a21]",
    muted: "text-[#8a8378] hover:text-[#1a1814]",
  } as const;
  return (
    <button
      type="button"
      className={cx(
        "text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50",
        tones[tone],
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

export function AdminSuccessBanner({
  message,
  className = "",
  onDismiss,
}: {
  message: string;
  className?: string;
  onDismiss?: () => void;
}) {
  return (
    <div
      className={cx(
        "mb-4 flex items-start justify-between gap-3 rounded-2xl border border-emerald-200/80 bg-emerald-50/90 px-4 py-3 text-sm text-emerald-800",
        className,
      )}
      role="status"
    >
      <span className="min-w-0 break-words">{message}</span>
      {onDismiss ? (
        <button
          type="button"
          onClick={onDismiss}
          className="shrink-0 font-medium text-emerald-700 hover:text-emerald-900"
        >
          关闭
        </button>
      ) : null}
    </div>
  );
}

export function AdminInfoBanner({
  message,
  className = "",
}: {
  message: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cx(
        "mb-4 flex items-center gap-2 rounded-2xl border border-blue-200/80 bg-blue-50/90 px-4 py-3 text-sm text-blue-800",
        className,
      )}
    >
      {message}
    </div>
  );
}

export function AdminModal({
  open,
  title,
  children,
  onClose,
  widthClass = "max-w-lg",
  footer,
}: {
  open: boolean;
  title: string;
  children: ReactNode;
  onClose: () => void;
  widthClass?: string;
  footer?: ReactNode;
}) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4" role="presentation">
      <div className="absolute inset-0 bg-slate-900/45 backdrop-blur-[2px]" onClick={onClose} aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        className={cx(
          "relative w-full overflow-hidden rounded-2xl border border-slate-200/80 bg-white shadow-[0_24px_64px_rgba(15,23,42,0.18)]",
          widthClass,
        )}
      >
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-4 sm:px-6">
          <h2 className="text-base font-semibold tracking-tight text-slate-900">{title}</h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg px-2 py-1 text-lg leading-none text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
            aria-label="关闭"
          >
            ×
          </button>
        </div>
        <div className="max-h-[min(70vh,32rem)] overflow-y-auto px-5 py-4 sm:px-6">{children}</div>
        {footer ? (
          <div className="flex flex-wrap justify-end gap-2 border-t border-slate-100 px-5 py-4 sm:px-6">
            {footer}
          </div>
        ) : null}
      </div>
    </div>
  );
}
