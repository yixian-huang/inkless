import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { ChevronRight } from "lucide-react";
import { adminTheme } from "./adminTheme";

export type AdminBreadcrumb = {
  label: string;
  to?: string;
};

export interface AdminPageHeaderProps {
  title: string;
  description?: string;
  actions?: ReactNode;
  breadcrumbs?: AdminBreadcrumb[];
  className?: string;
}

export default function AdminPageHeader({
  title,
  description,
  actions,
  breadcrumbs,
  className = "",
}: AdminPageHeaderProps) {
  return (
    <div className={`mb-6 ${className}`}>
      {breadcrumbs && breadcrumbs.length > 0 && (
        <nav className="mb-2 flex flex-wrap items-center gap-1 text-xs text-slate-500" aria-label="面包屑">
          {breadcrumbs.map((crumb, index) => {
            const isLast = index === breadcrumbs.length - 1;
            return (
              <span key={`${crumb.label}-${index}`} className="inline-flex items-center gap-1">
                {index > 0 && <ChevronRight className="h-3 w-3 text-slate-400" aria-hidden />}
                {crumb.to && !isLast ? (
                  <Link to={crumb.to} className="hover:text-slate-800 transition-colors">
                    {crumb.label}
                  </Link>
                ) : (
                  <span className={isLast ? "text-slate-700 font-medium" : undefined}>{crumb.label}</span>
                )}
              </span>
            );
          })}
        </nav>
      )}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <h1 className={adminTheme.pageTitle}>{title}</h1>
          {description ? <p className={adminTheme.pageDesc}>{description}</p> : null}
        </div>
        {actions ? (
          <div className="flex flex-wrap items-center gap-2 shrink-0">{actions}</div>
        ) : null}
      </div>
    </div>
  );
}
