import type { HeaderBrandMode } from "@/types/siteConfig";
import type { BrandingView } from "@/hooks/useBranding";

/**
 * On theme home, keep header chrome quiet so AuthorIntro owns identity.
 * Prefer no brand mark when a hero avatar is present (avoids double monogram).
 * Otherwise keep a compact logo-only mark for navigation home-link affordance.
 */
export function resolveBlogHomeBrandMode(
  brandMode: HeaderBrandMode,
  branding: BrandingView,
  compactHome: boolean,
): HeaderBrandMode {
  if (!compactHome || brandMode === "none") return brandMode;

  // Hero already shows avatar / identity — header stays minimal.
  if (branding.author.avatar?.trim()) return "none";

  if (brandMode === "logo" && branding.logo.light?.trim()) return "logo";
  if (brandMode === "avatar" && branding.author.avatar?.trim()) return "avatar";
  // Text name on home competes with the hero title; hide it.
  if (brandMode === "text") return "none";
  return "none";
}
