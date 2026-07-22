import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { http } from "@/api/http";
import type { PageConfig } from "./types";
import { resolveLocale } from "@/utils/locale";
import { SectionRenderer } from "./sections";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

interface DynamicPageProps {
  slug?: string;
}

export default function DynamicPage({ slug: slugProp }: DynamicPageProps = {}) {
  const { "*": paramSlug } = useParams();
  const slug = slugProp || paramSlug;
  const { t, i18n } = useTranslation("common");
  const locale = resolveLocale(i18n.language);

  const [config, setConfig] = useState<PageConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [title, setTitle] = useState<string>("");
  useDocumentTitle(title);

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
        setConfig(raw);
        setTitle(res.data.title || res.data.metaTitle || "");
        setLoading(false);
      })
      .catch((e) => {
        setError(e.response?.data?.error || e.message);
        setLoading(false);
      });
  }, [slug, locale]);

  if (loading) {
    return (
      <div className="min-h-[60vh] flex items-center justify-center">
        <div className="text-gray-600">Loading...</div>
      </div>
    );
  }

  if (error || !config) {
    return (
      <div className="min-h-[60vh] flex items-center justify-center">
        <div className="text-red-600">{error || "Page not found"}</div>
      </div>
    );
  }

  const visibleSections = (config.sections ?? []).filter(
    (s) => !s.settings?.hidden,
  );

  if (visibleSections.length === 0) {
    return (
      <div className="min-h-[50vh] flex items-center justify-center px-4">
        <p className="text-center text-on-surface-muted text-sm md:text-base">
          {t("status.pageEmpty", {
            defaultValue:
              locale === "zh"
                ? "页面暂无内容"
                : "This page has no content yet",
          })}
        </p>
      </div>
    );
  }

  return (
    <>
      {visibleSections.map((section) => (
        <SectionRenderer key={section.id} section={section} />
      ))}
    </>
  );
}
