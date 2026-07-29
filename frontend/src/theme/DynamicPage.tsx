import { useState, useEffect, useMemo, type CSSProperties } from "react";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { http } from "@/api/http";
import type { PageConfig } from "./types";
import { resolveLocale } from "@/utils/locale";
import { SectionRenderer } from "./sections";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useContentMaxWidth } from "@/plugins/hooks";
import SeoHead from "@/components/SeoHead";
import DynamicPageHeader from "./DynamicPageHeader";
import {
  resolveDynamicPageLayout,
  shouldShowPageHeader,
} from "./dynamicPageLayout";

interface DynamicPageProps {
  slug?: string;
}

function asDisplayString(value: unknown, locale: string): string {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "object") {
    const obj = value as Record<string, unknown>;
    const localized = obj[locale] ?? obj.zh ?? obj.en;
    if (typeof localized === "string") return localized;
  }
  return "";
}

export default function DynamicPage({ slug: slugProp }: DynamicPageProps = {}) {
  const { "*": paramSlug } = useParams();
  const slug = slugProp || paramSlug;
  const { t, i18n } = useTranslation("common");
  const locale = resolveLocale(i18n.language);
  const contentMaxWidth = useContentMaxWidth();

  const [config, setConfig] = useState<PageConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [metaTitle, setMetaTitle] = useState("");
  const [metaDescription, setMetaDescription] = useState("");
  useDocumentTitle(metaTitle || title);

  useEffect(() => {
    if (!slug) {
      setError("No page slug provided");
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    http
      .get(`/public/pages/${slug}`, { params: { locale } })
      .then((res) => {
        const raw = res.data.publishedConfig ?? res.data.config ?? res.data;
        // Normalize sections: backend uses "props", frontend SectionData uses "data"
        if (raw?.sections) {
          raw.sections = raw.sections.map((s: any) => ({
            ...s,
            data: s.data || s.props || {},
          }));
        }
        setConfig(raw as PageConfig);
        setTitle(asDisplayString(res.data.title, locale));
        setDescription(asDisplayString(res.data.description, locale));
        setMetaTitle(asDisplayString(res.data.metaTitle, locale));
        setMetaDescription(asDisplayString(res.data.metaDescription, locale));
        setLoading(false);
      })
      .catch((e) => {
        setError(e.response?.data?.error || e.message);
        setLoading(false);
      });
  }, [slug, locale]);

  const layout = useMemo(
    () => (config ? resolveDynamicPageLayout(config) : "reading"),
    [config],
  );
  const showHeader = useMemo(
    () => (config ? shouldShowPageHeader(config, layout, title) : false),
    [config, layout, title],
  );

  const visibleSections = useMemo(
    () => (config?.sections ?? []).filter((s) => !s.settings?.hidden),
    [config],
  );

  if (loading) {
    return (
      <div
        className="min-h-[60vh] flex items-center justify-center"
        data-testid="dynamic-page-loading"
      >
        <div className="text-on-surface-muted">
          {t("status.loading", { defaultValue: "Loading..." })}
        </div>
      </div>
    );
  }

  if (error || !config) {
    return (
      <div
        className="min-h-[60vh] flex items-center justify-center px-4"
        data-testid="dynamic-page-error"
      >
        <div className="text-red-600 text-center">
          {error || t("status.pageNotFound", { defaultValue: "Page not found" })}
        </div>
      </div>
    );
  }

  const seoTitle = metaTitle || title;
  const seoDesc = metaDescription || description;
  const canonical = slug ? `/p/${String(slug).replace(/^\/+/, "")}` : undefined;

  if (visibleSections.length === 0) {
    return (
      <div className="flex-1 flex flex-col min-h-[50vh]" data-testid="dynamic-page-empty">
        <SeoHead
          title={seoTitle}
          description={seoDesc}
          canonicalUrl={canonical}
          locale={locale}
        />
        {showHeader && (
          <DynamicPageHeader
            title={title}
            description={description}
            narrow={layout === "reading"}
            maxWidth={contentMaxWidth}
          />
        )}
        <div className="flex-1 flex items-center justify-center px-4 py-section">
          <p className="text-center text-on-surface-muted text-sm md:text-base">
            {t("status.pageEmpty", {
              defaultValue:
                locale === "zh"
                  ? "页面暂无内容"
                  : "This page has no content yet",
            })}
          </p>
        </div>
      </div>
    );
  }

  const readingColumn =
    layout === "reading"
      ? {
          className: "w-full mx-auto px-4 md:px-content xl:px-8 pb-section-lg",
          style: { maxWidth: contentMaxWidth } as CSSProperties,
        }
      : {
          className: "w-full pb-section-lg",
          style: undefined as CSSProperties | undefined,
        };

  return (
    <div className="flex-1 flex flex-col min-w-0" data-testid="dynamic-page" data-layout={layout}>
      <SeoHead
        title={seoTitle}
        description={seoDesc}
        ogTitle={seoTitle}
        ogDescription={seoDesc}
        ogType="website"
        canonicalUrl={canonical}
        locale={locale}
      />
      {showHeader && (
        <DynamicPageHeader
          title={title}
          description={description}
          narrow={layout === "reading"}
          maxWidth={contentMaxWidth}
        />
      )}
      <div className={readingColumn.className} style={readingColumn.style}>
        {visibleSections.map((section) => (
          <SectionRenderer
            key={section.id}
            section={section}
            pageLayout={layout}
          />
        ))}
      </div>
    </div>
  );
}
