import type { ReactNode } from "react";
import { adminTheme } from "./adminTheme";

export interface AdminCardProps {
  children: ReactNode;
  className?: string;
  padded?: boolean;
  title?: string;
  description?: string;
  actions?: ReactNode;
}

export default function AdminCard({
  children,
  className = "",
  padded = true,
  title,
  description,
  actions,
}: AdminCardProps) {
  return (
    <div className={`${adminTheme.card} ${className}`}>
      {(title || actions) && (
        <div className="flex items-start justify-between gap-3 border-b border-slate-100 px-5 py-4 sm:px-6">
          <div className="min-w-0">
            {title ? <h2 className="text-base font-semibold text-slate-900">{title}</h2> : null}
            {description ? <p className="mt-0.5 text-sm text-slate-500">{description}</p> : null}
          </div>
          {actions ? <div className="shrink-0">{actions}</div> : null}
        </div>
      )}
      <div className={padded ? adminTheme.cardPad : ""}>{children}</div>
    </div>
  );
}
