export interface AdminLoadingProps {
  label?: string;
  className?: string;
}

export default function AdminLoading({ label = "加载中…", className = "" }: AdminLoadingProps) {
  return (
    <div className={`flex flex-col items-center justify-center py-16 text-slate-500 ${className}`}>
      <div
        className="h-8 w-8 animate-spin rounded-full border-2 border-slate-200 border-t-blue-600"
        aria-hidden
      />
      <p className="mt-3 text-sm">{label}</p>
    </div>
  );
}

export function AdminRouteFallback() {
  return (
    <div className="animate-pulse space-y-4 p-1">
      <div className="h-8 w-48 rounded-lg bg-slate-200" />
      <div className="h-4 w-72 rounded bg-slate-200" />
      <div className="mt-6 h-40 rounded-xl bg-slate-200/80" />
      <div className="h-40 rounded-xl bg-slate-200/60" />
    </div>
  );
}
