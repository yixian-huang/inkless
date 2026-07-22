import type { ThemePlugin } from "@inkless/theme-host";
import { editorialFirmTokens, noirGalleryTokens } from "./tokens";

/** Theme id — keep in sync with host `BUILTIN_THEME_IDS.EDITORIAL_FIRM` and DB. */
export const EDITORIAL_FIRM_THEME_ID = "editorial-firm";

/**
 * Host contract this package targets.
 * Keep in lockstep with host THEME_CONTRACT_VERSION and inkless.theme.json.
 */
export const EDITORIAL_FIRM_CONTRACT_VERSION = "1";

/** Content keys for the four editorial-firm pages. */
export const EDITORIAL_PAGE_CONTENT_KEYS = [
  "home",
  "about",
  "services",
  "contact",
] as const;

export type EditorialPageContentKey = (typeof EDITORIAL_PAGE_CONTENT_KEYS)[number];

const PAGE_NAV = {
  home: { slug: "home", label: "Home", labelZh: "首页", order: 0 },
  about: { slug: "about", label: "About", labelZh: "关于", order: 1 },
  services: { slug: "services", label: "Services", labelZh: "服务", order: 2 },
  contact: { slug: "contact", label: "Contact", labelZh: "联系", order: 3 },
} as const;

/** Wide editorial layout (magazine / atelier firm site). */
export const EDITORIAL_DEFAULT_LAYOUT = {
  type: "default" as const,
  contentProfile: "wide" as const,
  header: { style: "sticky" as const },
  footer: { style: "minimal" as const },
};

/**
 * Build the editorial-firm theme plugin.
 *
 * Chrome (layoutChrome) and custom sections land in later tasks.
 * Pages are CMS section-driven (`renderMode: "dynamic"`).
 */
export function createEditorialFirmTheme(): ThemePlugin {
  return {
    manifest: {
      id: EDITORIAL_FIRM_THEME_ID,
      name: "Editorial Firm",
      nameZh: "编辑机构",
      description:
        "Magazine-style firm site: home, about, services, contact — section-driven",
      descriptionZh: "杂志气质机构官网：首页、关于、服务、联系 — 区块配置驱动",
      author: "Inkless CMS",
      version: "0.1.0",
      type: "theme",
      preview: "linear-gradient(135deg, #111111 0%, #C45C26 100%)",
      tags: ["corporate", "editorial", "bilingual", "dynamic"],
    },
    contractVersion: EDITORIAL_FIRM_CONTRACT_VERSION,
    defaultTokens: editorialFirmTokens,
    tokenPresets: [
      {
        id: "ink-editorial",
        name: "Ink Editorial",
        nameZh: "墨色编辑",
        preview: "linear-gradient(135deg, #111111 0%, #C45C26 100%)",
        tokens: editorialFirmTokens,
      },
      {
        id: "noir-gallery",
        name: "Noir Gallery",
        nameZh: "黑白画廊",
        preview: "linear-gradient(135deg, #0A0A0A 0%, #E8B86D 100%)",
        tokens: noirGalleryTokens,
      },
    ],
    pages: EDITORIAL_PAGE_CONTENT_KEYS.map((key) => ({
      slug: PAGE_NAV[key].slug,
      renderMode: "dynamic" as const,
      contentKey: key,
      nav: {
        label: PAGE_NAV[key].label,
        labelZh: PAGE_NAV[key].labelZh,
        order: PAGE_NAV[key].order,
        showInHeader: true,
        showInFooter: true,
      },
    })),
    defaultLayout: EDITORIAL_DEFAULT_LAYOUT,
    sections: {},
    sectionMetas: [],
    // layoutChrome filled in Task 2
  };
}

/**
 * Theme shell without host chrome/sections (scaffold).
 * Prefer registering via host after Task 2+ wires chrome.
 */
export const editorialFirmTheme: ThemePlugin = createEditorialFirmTheme();

export { editorialFirmTokens, noirGalleryTokens } from "./tokens";
