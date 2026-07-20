export interface AdminErrorBannerProps {
  message: string;
  className?: string;
  onDismiss?: () => void;
}

export default function AdminErrorBanner({ message, className = "", onDismiss }: AdminErrorBannerProps) {
  return (
    <div
      role="alert"
      className={`mb-4 flex items-start justify-between gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 ${className}`}
    >
      <span className="min-w-0 break-words">{message}</span>
      {onDismiss ? (
        <button
          type="button"
          onClick={onDismiss}
          className="shrink-0 text-red-600 hover:text-red-900 font-medium"
        >
          关闭
        </button>
      ) : null}
    </div>
  );
}
