export interface AdminLoadingProps {
  label?: string;
  className?: string;
}

export default function AdminLoading({ label = "加载中…", className = "" }: AdminLoadingProps) {
  return (
    <div className={`flex flex-col items-center justify-center py-16 text-[#8a8378] ${className}`}>
      <div
        className="h-9 w-9 animate-spin rounded-full border-2 border-[#e4ddd2] border-t-[#1a1814]"
        aria-hidden
      />
      <p className="mt-3.5 text-sm font-medium tracking-wide text-[#8a8378]">{label}</p>
    </div>
  );
}

export function AdminRouteFallback() {
  return (
    <div className="animate-pulse space-y-4 p-1" aria-busy="true" aria-label="页面加载中">
      <div className="h-8 w-48 rounded-lg bg-[#e4ddd2]/80" />
      <div className="h-4 w-72 rounded-md bg-[#e4ddd2]/55" />
      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="h-24 rounded-xl bg-[#e4ddd2]/70" />
        <div className="h-24 rounded-xl bg-[#e4ddd2]/55" />
        <div className="h-24 rounded-xl bg-[#e4ddd2]/45" />
        <div className="h-24 rounded-xl bg-[#e4ddd2]/35" />
      </div>
      <div className="h-52 rounded-xl bg-[#e4ddd2]/45" />
    </div>
  );
}
