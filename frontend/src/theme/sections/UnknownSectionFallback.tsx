interface UnknownSectionFallbackProps {
  type: string;
  /** When true, show slightly more detail (admin / dev) */
  detailed?: boolean;
}

/**
 * Public-safe placeholder when a section type is not registered
 * (inactive theme `ef-*`/`pf-*`, removed plugin, or typo).
 * Must not wipe draft data — only affects render.
 */
export default function UnknownSectionFallback({
  type,
  detailed = false,
}: UnknownSectionFallbackProps) {
  const isThemePrefixed = /^(ef|pf|bf)-/i.test(type);

  return (
    <div
      className="mx-4 my-3 md:mx-auto md:max-w-layout rounded-lg border border-dashed border-border bg-surface-alt/60 px-4 py-6 text-center"
      data-testid="unknown-section-fallback"
      data-section-type={type}
      role="status"
    >
      <p className="text-sm font-medium text-on-surface">
        {isThemePrefixed
          ? "此区块需要对应主题才能完整显示"
          : "未知页面区块"}
      </p>
      <p className="mt-1 text-xs text-on-surface-muted">
        {isThemePrefixed
          ? "当前激活主题未提供该组件；内容仍保留在草稿/已发布配置中，切换回原主题或更新主题后可恢复。"
          : "页面配置引用了未注册的区块类型，数据未删除。"}
      </p>
      {(detailed || import.meta.env.DEV) && (
        <p className="mt-2 font-mono text-[11px] text-on-surface-muted break-all">
          type: {type}
        </p>
      )}
    </div>
  );
}
