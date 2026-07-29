/**
 * Slug conflict checks for unified (dynamic) pages vs Host system routes
 * and active theme declared pages (ADR-0002 appendix D).
 */

/** Mirrors backend reservedPublicPageSlugs (+ common static first segments). */
export const RESERVED_PUBLIC_PAGE_SLUGS = new Set([
  "admin",
  "setup",
  "blog",
  "categories",
  "tags",
  "search",
  "p",
  "public",
  "auth",
  "uploads",
  "assets",
  "images",
  "health",
  "version",
  "metrics",
  "api-docs",
  "sitemap",
  "feed",
  "robots",
  "login",
  "register",
  "api",
]);

const SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

export type PageSlugConflictKind =
  | "empty"
  | "format"
  | "reserved"
  | "theme"
  | "ok";

export interface PageSlugConflictResult {
  kind: PageSlugConflictKind;
  /** Normalized slug (trimmed, no leading/trailing slashes) */
  slug: string;
  /** Human message (zh) for admin UI */
  messageZh: string;
  /** Human message (en) */
  messageEn: string;
  /** Blocking: cannot save create/update */
  blocking: boolean;
}

export function normalizePageSlug(raw: string): string {
  return raw.trim().replace(/^\/+|\/+$/g, "").toLowerCase();
}

export interface CheckPageSlugOptions {
  /** Active theme pages[].slug values (and contentKey when different) */
  themeSlugs?: Iterable<string>;
  /** When editing, allow current page to keep its slug even if listed elsewhere */
  allowSlug?: string;
}

/**
 * Validate a unified-page slug for format, Host reserved paths, and theme IA slugs.
 * Theme conflicts are blocking: theme routes outrank `/p/:slug` for primary IA (ADR-0002 D).
 */
export function checkPageSlug(
  raw: string,
  opts: CheckPageSlugOptions = {},
): PageSlugConflictResult {
  const slug = normalizePageSlug(raw);
  const allow = opts.allowSlug ? normalizePageSlug(opts.allowSlug) : "";

  if (!slug) {
    return {
      kind: "empty",
      slug,
      messageZh: "请输入 URL 路径（slug）",
      messageEn: "Slug is required",
      blocking: true,
    };
  }

  if (!SLUG_PATTERN.test(slug)) {
    return {
      kind: "format",
      slug,
      messageZh: "slug 仅允许小写字母、数字与连字符",
      messageEn: "Slug must be lowercase letters, numbers, and hyphens only",
      blocking: true,
    };
  }

  if (RESERVED_PUBLIC_PAGE_SLUGS.has(slug)) {
    return {
      kind: "reserved",
      slug,
      messageZh: `「${slug}」为系统保留路径，不能用作动态页 slug`,
      messageEn: `"${slug}" is reserved by the application`,
      blocking: true,
    };
  }

  const themeSet = new Set(
    [...(opts.themeSlugs ?? [])].map((s) => normalizePageSlug(s)).filter(Boolean),
  );
  // Theme-declared IA outranks /p/* (ADR-0002 appendix D). When editing a page
  // that already uses this slug, allow keep (data preserved; public may still prefer theme route).
  const themeHit =
    themeSet.has(slug) || (slug === "home" && themeSet.has("home"));
  if (themeHit && !(allow && slug === allow)) {
    return {
      kind: "theme",
      slug,
      messageZh: `「${slug}」已被当前主题声明为页面路由，主题页优先于 /p/${slug}；请改用其他 slug，或把该页做成主题页`,
      messageEn: `"${slug}" is declared by the active theme (theme routes outrank /p/*)`,
      blocking: true,
    };
  }

  return {
    kind: "ok",
    slug,
    messageZh: "",
    messageEn: "",
    blocking: false,
  };
}

/** Collect slug keys from theme page definitions. */
export function themePageSlugsFromManifest(
  pages: Array<{ slug?: string; contentKey?: string }> | undefined | null,
): string[] {
  if (!pages?.length) return [];
  const out = new Set<string>();
  for (const p of pages) {
    if (p.slug) out.add(normalizePageSlug(p.slug));
    if (p.contentKey) out.add(normalizePageSlug(p.contentKey));
  }
  return [...out];
}
