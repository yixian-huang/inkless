import type { DynamicPageLayout, PageConfig, SectionData } from "./types";

const LEADING_HERO_TYPES = new Set(["hero"]);

export function hasLeadingHero(sections: SectionData[]): boolean {
  const first = sections.find((s) => !s.settings?.hidden);
  return Boolean(first && LEADING_HERO_TYPES.has(first.type));
}

/** True when every visible section is document-ish (rich text / checklist). */
export function isDocOnlySections(sections: SectionData[]): boolean {
  const visible = sections.filter((s) => !s.settings?.hidden);
  if (visible.length === 0) return false;
  return visible.every((s) => s.type === "rich-text" || s.type === "checklist");
}

export function resolveDynamicPageLayout(config: PageConfig): DynamicPageLayout {
  const raw = (config.layout || "auto").toLowerCase();
  if (raw === "reading" || raw === "doc") return "reading";
  if (raw === "landing" || raw === "marketing") return "landing";

  const sections = config.sections ?? [];
  if (hasLeadingHero(sections)) return "landing";
  if (isDocOnlySections(sections)) return "reading";
  // Mixed section stacks (cards + text) behave as landing for full-bleed rhythm
  if (sections.some((s) => !s.settings?.hidden && s.type !== "rich-text")) {
    return "landing";
  }
  return "reading";
}

export function shouldShowPageHeader(
  config: PageConfig,
  layout: DynamicPageLayout,
  title: string,
): boolean {
  if (!title.trim()) return false;
  if (config.showPageHeader === false) return false;
  if (config.showPageHeader === true) return true;
  // Reading pages without a leading hero get a host title
  if (layout === "reading" && !hasLeadingHero(config.sections ?? [])) return true;
  return false;
}
