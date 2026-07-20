import { useState, useEffect, useCallback } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  ArticleList,
  AuthorIntro,
  BlogPageShell,
  getPublicArticles,
  pickLocaleValue,
  SeoHead,
  SITE_CONFIG_GLOBAL_DEFAULT,
  useGlobalConfig,
  useIsReadingLayout,
  useLocaleMode,
  useSEODefaults,
} from "@inkless/theme-host";

const HOME_RECENT_COUNT = 6;

/**
 * blog-first theme home — presentation lives in this package so Inkless host
 * only re-exports / registers it.
 */
export default function BlogFirstHomePage() {
  const { t } = useTranslation("common");
  const { config } = useGlobalConfig();
  const { defaultTitle, defaultDescription, defaultOgImage } = useSEODefaults();
  const { localeMode, defaultLocale, currentLocale } = useLocaleMode();
  const isReading = useIsReadingLayout();

  const siteConfig = config.siteConfig ?? SITE_CONFIG_GLOBAL_DEFAULT;
  const siteName = pickLocaleValue({
    value: siteConfig.identity.name,
    mode: localeMode,
    defaultLocale,
    currentLocale,
  });
  const authorName = siteConfig.author?.name?.trim() || siteName;
  const bio = pickLocaleValue({
    value: siteConfig.author?.bio,
    mode: localeMode,
    defaultLocale,
    currentLocale,
  });
  const tagline = pickLocaleValue({
    value: siteConfig.identity.tagline,
    mode: localeMode,
    defaultLocale,
    currentLocale,
  });
  const intro = bio || tagline || defaultDescription;

  const [articles, setArticles] = useState<Awaited<ReturnType<typeof getPublicArticles>>["items"]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadRecent = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getPublicArticles(1, HOME_RECENT_COUNT);
      setArticles(data.items || []);
      setTotal(data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load articles");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadRecent();
  }, [loadRecent]);

  // Prefer site SEO defaultTitle ("黄逸仙 · yx.ink") over templating the bare site name.
  const pageTitle = defaultTitle || siteName || t("blog.homeTitle");

  return (
    <>
      <SeoHead
        title={pageTitle}
        description={intro}
        ogTitle={siteName || pageTitle}
        ogDescription={intro}
        ogImage={siteConfig.brand.ogImage || defaultOgImage}
        ogType="website"
        canonicalUrl="/"
      />
      <BlogPageShell className={isReading ? "pt-1 md:pt-2" : undefined}>
        <AuthorIntro
          avatar={siteConfig.author?.avatar}
          name={authorName}
          tagline={tagline}
          bio={bio}
          intro={intro}
          showSocials={false}
          showBio={false}
          showSubtitle={false}
        />

        <section className={isReading ? "pt-1" : undefined}>
          <div
            className={
              isReading
                ? "flex items-baseline justify-between gap-4 mb-6"
                : "flex items-center justify-between mb-6"
            }
          >
            <h2
              className={
                isReading
                  ? "text-[11px] font-sans font-medium uppercase tracking-[0.22em] text-on-surface-muted"
                  : "text-xl font-semibold text-on-surface"
              }
            >
              {t("blog.recentPosts")}
            </h2>
            {total > HOME_RECENT_COUNT && (
              <Link
                to="/blog"
                className={
                  isReading
                    ? "text-xs font-sans uppercase tracking-[0.14em] text-on-surface-muted hover:text-on-surface transition-colors shrink-0"
                    : "text-sm text-primary hover:text-accent transition-colors"
                }
              >
                {t("blog.viewAll")} →
              </Link>
            )}
          </div>

          {error && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-800">
              {error}
            </div>
          )}

          <ArticleList
            articles={articles}
            localeMode={localeMode}
            defaultLocale={defaultLocale}
            currentLocale={currentLocale}
            loading={loading}
            loadingLabel={t("status.loading")}
            emptyLabel={t("blog.noPosts")}
          />

          {total > 0 && total <= HOME_RECENT_COUNT && (
            <p className={isReading ? "mt-12 text-center" : "mt-6"}>
              <Link
                to="/blog"
                className={
                  isReading
                    ? "text-xs font-sans uppercase tracking-[0.14em] text-on-surface-muted hover:text-on-surface transition-colors"
                    : "text-sm text-primary hover:text-accent transition-colors"
                }
              >
                {t("blog.archive")} →
              </Link>
            </p>
          )}
        </section>
      </BlogPageShell>
    </>
  );
}
