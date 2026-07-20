import AdminButton from "./AdminButton";

export interface AdminPaginationProps {
  page: number;
  totalPages: number;
  total?: number;
  onPageChange: (page: number) => void;
  className?: string;
}

export default function AdminPagination({
  page,
  totalPages,
  total,
  onPageChange,
  className = "",
}: AdminPaginationProps) {
  if (totalPages <= 1 && total === undefined) return null;

  return (
    <div className={`mt-4 flex flex-wrap items-center justify-between gap-3 ${className}`}>
      <p className="text-sm text-slate-500">
        {typeof total === "number" ? `共 ${total} 条` : null}
        {totalPages > 1 ? (
          <span className={typeof total === "number" ? " ml-2" : ""}>
            第 {page} / {totalPages} 页
          </span>
        ) : null}
      </p>
      {totalPages > 1 ? (
        <div className="flex items-center gap-2">
          <AdminButton
            variant="secondary"
            size="sm"
            disabled={page <= 1}
            onClick={() => onPageChange(page - 1)}
          >
            上一页
          </AdminButton>
          <AdminButton
            variant="secondary"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => onPageChange(page + 1)}
          >
            下一页
          </AdminButton>
        </div>
      ) : null}
    </div>
  );
}
