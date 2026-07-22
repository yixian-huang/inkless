import {
  SITE_CONFIG_GLOBAL_DEFAULT,
  type SiteConfigGlobal,
  type SiteConfigSocial,
} from "@/types/siteConfig";

type LocalizedLike = { zh?: string; en?: string };

function pickLoc(value: unknown): LocalizedLike | undefined {
  if (!value) return undefined;
  if (typeof value === "string") {
    const s = value.trim();
    return s ? { zh: s } : undefined;
  }
  if (typeof value === "object") {
    const o = value as LocalizedLike;
    const zh = typeof o.zh === "string" ? o.zh : undefined;
    const en = typeof o.en === "string" ? o.en : undefined;
    if (zh || en) return { zh, en };
  }
  return undefined;
}

function mediaUrl(value: unknown): string {
  if (typeof value === "string") return value.trim();
  if (value && typeof value === "object") {
    const url = (value as { url?: unknown }).url;
    if (typeof url === "string") return url.trim();
  }
  return "";
}

/**
 * Convert legacy impress global content
 * (`branding` / `header` / `footer` / `nav`) into SiteConfigGlobal fields.
 */
function legacyToPartial(raw: Record<string, unknown>): Partial<SiteConfigGlobal> {
  const branding = (raw.branding ?? {}) as {
    companyName?: unknown;
    logo?: unknown;
  };
  const header = (raw.header ?? {}) as { logo?: unknown };
  const footer = (raw.footer ?? {}) as {
    copyright?: unknown;
    address?: unknown;
    phone?: unknown;
  };

  const name = pickLoc(branding.companyName);
  const logoLight =
    mediaUrl(branding.logo) || mediaUrl(header.logo) || "";
  const copyright = pickLoc(footer.copyright);

  // Best-effort ICP extraction from Chinese copyright tails like "…京ICP备xxxx号"
  let icp: string | undefined;
  const zhCopy = copyright?.zh ?? "";
  const icpMatch = zhCopy.match(/(京ICP备[\dA-Za-z-]*号?(?:-\d+)?)/);
  if (icpMatch) icp = icpMatch[1];

  return {
    identity: name
      ? {
          name: { zh: name.zh ?? "", en: name.en },
          localeMode: "bilingual",
          defaultLocale: "zh",
        }
      : undefined,
    brand: logoLight
      ? {
          logo: { light: logoLight },
          favicon: "/brand/favicon.svg",
          ogImage: "",
          primaryColor: "#1a5f8f",
        }
      : undefined,
    footer: copyright
      ? {
          copyright: { zh: copyright.zh, en: copyright.en },
          icp,
        }
      : undefined,
    header: {
      brandMode: logoLight ? "logo" : "text",
      showRssLink: false,
      showSocials: false,
    },
  };
}

/** Deep-merge API payload with defaults so partial drafts never crash the form. */
export function normalizeSiteConfig(raw: unknown): SiteConfigGlobal {
  const d = SITE_CONFIG_GLOBAL_DEFAULT;
  if (!raw || typeof raw !== "object") return structuredClone(d);

  const input = raw as Record<string, unknown>;
  // If payload is legacy impress shape (no identity), project into new fields first.
  let projected: Record<string, unknown> = input;
  if (!("identity" in input) && ("branding" in input || "header" in input)) {
    const legacy = legacyToPartial(input) as Record<string, unknown>;
    projected = {
      ...legacy,
      ...input,
      // Keep converted siteConfig fields; do not let raw header/footer objects clobber them entirely.
      identity: input.identity ?? legacy.identity,
      brand: input.brand ?? legacy.brand,
      footer:
        input.footer && typeof input.footer === "object" && "copyright" in (input.footer as object)
          ? input.footer
          : (legacy.footer ?? input.footer),
      header: {
        ...((legacy.header as object) || {}),
        ...((typeof input.header === "object" && input.header) || {}),
      },
    };
  }

  const r = projected as Partial<SiteConfigGlobal> & Record<string, unknown>;
  const identity = (r.identity ?? {}) as Partial<SiteConfigGlobal["identity"]>;
  const brand = (r.brand ?? {}) as Partial<SiteConfigGlobal["brand"]>;
  const logo = (brand.logo ?? {}) as Partial<SiteConfigGlobal["brand"]["logo"]>;
  const author = (r.author ?? {}) as Partial<SiteConfigGlobal["author"]>;
  const footer = (r.footer ?? {}) as Partial<SiteConfigGlobal["footer"]>;
  const seo = (r.seo ?? {}) as Partial<SiteConfigGlobal["seo"]>;
  const header = (r.header ?? {}) as Partial<NonNullable<SiteConfigGlobal["header"]>>;

  const socials: SiteConfigSocial[] = Array.isArray(author.socials)
    ? author.socials.filter((s): s is SiteConfigSocial => !!s && typeof s === "object")
    : [];

  const copyright = footer.copyright
    ? { zh: footer.copyright.zh, en: footer.copyright.en }
    : pickLoc((input.footer as { copyright?: unknown } | undefined)?.copyright);

  let icp = footer.icp;
  if (!icp && copyright?.zh) {
    const m = copyright.zh.match(/(京ICP备[\dA-Za-z-]*号?(?:-\d+)?)/);
    if (m) icp = m[1];
  }

  return {
    identity: {
      name: { zh: identity.name?.zh ?? d.identity.name.zh, en: identity.name?.en },
      tagline: identity.tagline
        ? { zh: identity.tagline.zh, en: identity.tagline.en }
        : undefined,
      localeMode: identity.localeMode ?? d.identity.localeMode,
      defaultLocale: identity.defaultLocale ?? d.identity.defaultLocale,
    },
    brand: {
      logo: {
        light: logo.light ?? d.brand.logo.light,
        dark: logo.dark,
      },
      favicon: brand.favicon ?? d.brand.favicon,
      ogImage: brand.ogImage ?? d.brand.ogImage,
      primaryColor: brand.primaryColor ?? d.brand.primaryColor,
      accentColor: brand.accentColor,
    },
    author: {
      name: author.name ?? d.author.name,
      avatar: author.avatar,
      bio: author.bio ? { zh: author.bio.zh, en: author.bio.en } : undefined,
      location: author.location,
      socials,
    },
    footer: {
      copyright: copyright
        ? { zh: copyright.zh, en: copyright.en }
        : undefined,
      icp,
      extraLinks: footer.extraLinks,
    },
    seo: {
      defaultTitle: seo.defaultTitle
        ? { zh: seo.defaultTitle.zh, en: seo.defaultTitle.en }
        : undefined,
      titleTemplate: seo.titleTemplate,
      defaultDescription: seo.defaultDescription
        ? { zh: seo.defaultDescription.zh, en: seo.defaultDescription.en }
        : undefined,
      twitterHandle: seo.twitterHandle,
    },
    header: {
      brandMode: header.brandMode,
      showRssLink: header.showRssLink,
      showSocials: header.showSocials,
    },
  };
}
