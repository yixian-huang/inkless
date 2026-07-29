import { useMemo, type CSSProperties } from "react";
import type { DynamicPageLayout, PageConfig, SectionData } from "./types";
import { SectionRenderer } from "./sections";
import DynamicPageHeader from "./DynamicPageHeader";
import {
  resolveDynamicPageLayout,
  shouldShowPageHeader,
} from "./dynamicPageLayout";
import { useContentMaxWidth } from "@/plugins/hooks";

export interface DynamicPagePreviewProps {
  title?: string;
  description?: string;
  layout?: DynamicPageLayout | string;
  showPageHeader?: boolean;
  sections: SectionData[];
  /** Highlight selected section index in admin */
  selectedIndex?: number | null;
  onSelectSection?: (index: number) => void;
  className?: string;
}

/**
 * Admin / story preview that mirrors public DynamicPage shell (layout + header + sections).
 * Does not fetch or set document title / SEO.
 */
export default function DynamicPagePreview({
  title = "",
  description = "",
  layout: layoutProp,
  showPageHeader,
  sections,
  selectedIndex = null,
  onSelectSection,
  className = "",
}: DynamicPagePreviewProps) {
  const contentMaxWidth = useContentMaxWidth();

  const config: PageConfig = useMemo(
    () => ({
      layout: layoutProp,
      showPageHeader,
      sections,
    }),
    [layoutProp, showPageHeader, sections],
  );

  const layout = resolveDynamicPageLayout(config);
  const showHeader = shouldShowPageHeader(config, layout, title);
  const visible = sections.filter((s) => !s.settings?.hidden);

  const column =
    layout === "reading"
      ? {
          className: "w-full mx-auto px-4 md:px-content xl:px-8 pb-section-lg",
          style: { maxWidth: contentMaxWidth } as CSSProperties,
        }
      : {
          className: "w-full pb-section-lg",
          style: undefined as CSSProperties | undefined,
        };

  if (visible.length === 0) {
    return (
      <div
        className={`flex flex-col min-h-[200px] bg-surface ${className}`}
        data-testid="dynamic-page-preview"
        data-layout={layout}
      >
        {showHeader && (
          <DynamicPageHeader
            title={title || "页面标题"}
            description={description}
            narrow={layout === "reading"}
            maxWidth={contentMaxWidth}
          />
        )}
        <div className="flex-1 flex items-center justify-center text-sm text-on-surface-muted px-4 py-12">
          暂无可见区块
        </div>
      </div>
    );
  }

  return (
    <div
      className={`flex flex-col min-w-0 bg-surface ${className}`}
      data-testid="dynamic-page-preview"
      data-layout={layout}
    >
      {showHeader && (
        <DynamicPageHeader
          title={title || "页面标题"}
          description={description}
          narrow={layout === "reading"}
          maxWidth={contentMaxWidth}
        />
      )}
      <div className={column.className} style={column.style}>
        {sections.map((section, i) => {
          if (section.settings?.hidden) return null;
          const selected = selectedIndex === i;
          return (
            <div
              key={section.id}
              role={onSelectSection ? "button" : undefined}
              tabIndex={onSelectSection ? 0 : undefined}
              onClick={() => onSelectSection?.(i)}
              onKeyDown={(e) => {
                if (!onSelectSection) return;
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onSelectSection(i);
                }
              }}
              className={[
                "relative transition-all",
                onSelectSection ? "cursor-pointer" : "",
                selected
                  ? "ring-2 ring-blue-400 ring-inset"
                  : onSelectSection
                    ? "hover:ring-1 hover:ring-gray-300 hover:ring-inset"
                    : "",
              ]
                .filter(Boolean)
                .join(" ")}
            >
              <SectionRenderer section={section} pageLayout={layout} />
            </div>
          );
        })}
      </div>
    </div>
  );
}
