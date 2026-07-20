import type { ThemeTokens } from "@/theme/tokens";
import {
  DEFAULT_MONO_PRESET_ID,
  DEFAULT_SERIF_PRESET_ID,
  getFontPreset,
} from "./presets";
import type {
  ArticleTypographyConfig,
  ArticleTypographyOverride,
  BodyFontRole,
  CustomFontRef,
} from "./types";

const DEFAULT_BODY_SIZE = "1.0625rem";
const DEFAULT_BODY_LINE_HEIGHT = 1.8;

function mergeCustomFont(base: string, upload?: CustomFontRef): { stack: string; custom: CustomFontRef[] } {
  if (!upload?.family || !upload.url) {
    return { stack: base, custom: [] };
  }
  const quoted = upload.family.includes(" ") ? `"${upload.family}"` : upload.family;
  return {
    stack: `${quoted}, ${base}`,
    custom: [upload],
  };
}

function normalizeStack(stack: string): string {
  return stack
    .replace(/['"]/g, "")
    .replace(/\s+/g, " ")
    .trim()
    .toLowerCase();
}

/** Heuristic: stack already names a serif face (not merely "system-ui"). */
export function looksLikeSerifStack(stack: string): boolean {
  // Strip "sans-serif" so generic `serif` does not false-positive on UI stacks.
  const withoutSansSerif = stack.replace(/sans-serif/gi, "SANS");
  return /(?:georgia|palatino|times|iowan|book antiqua|songti|noto serif|source han serif|kaiti|lxgw|\bserif\b)/i.test(
    withoutSansSerif,
  );
}

/**
 * Resolve the editorial/serif heading stack used for article titles and
 * `bodyFontRole: "serif"` body text.
 *
 * Published theme tokens sometimes copy sans into `fonts.heading` (color-only
 * admin saves). Recover from `fontSources.headingPresetId` or the built-in
 * editorial Georgia preset when heading is empty or identical to sans.
 */
export function resolveSerifHeadingStack(tokens: ThemeTokens): string {
  const sources = tokens.fontSources ?? {};
  const heading = tokens.fonts.heading?.trim() ?? "";
  const sans = tokens.fonts.sans?.trim() ?? "";
  const preset = getFontPreset(sources.headingPresetId);
  const defaultSerif =
    getFontPreset(DEFAULT_SERIF_PRESET_ID)?.stack ??
    "Georgia, 'Iowan Old Style', 'Palatino Linotype', 'Book Antiqua', Palatino, serif";

  if (heading && looksLikeSerifStack(heading)) {
    return heading;
  }

  const polluted =
    !heading || (sans.length > 0 && normalizeStack(heading) === normalizeStack(sans));

  if (polluted) {
    if (preset && (preset.role === "serif" || looksLikeSerifStack(preset.stack))) {
      return preset.stack;
    }
    return defaultSerif;
  }

  // Explicit non-serif heading chosen by admin — respect it.
  return heading;
}

export interface ResolveTypographyInput {
  tokens: ThemeTokens;
  /** Installed-theme config flattened keys, e.g. `article.bodyFontRole`. */
  themeSettings?: Record<string, unknown>;
  articleOverride?: ArticleTypographyOverride | null;
}

/**
 * Resolve article typography from design tokens + `article.bodyFontRole` only.
 * Font stacks, size, and line height come from tokens; per-article override may change body role only.
 */
export function resolveArticleTypography(input: ResolveTypographyInput): ArticleTypographyConfig {
  const { tokens, themeSettings = {}, articleOverride } = input;
  const sources = tokens.fontSources ?? {};

  const themeBodyRole =
    (themeSettings["article.bodyFontRole"] as BodyFontRole | undefined) ?? "serif";

  const bodySize = tokens.typography?.article?.bodySize ?? DEFAULT_BODY_SIZE;
  const bodyLineHeight = tokens.typography?.article?.bodyLineHeight ?? DEFAULT_BODY_LINE_HEIGHT;

  const overrideActive = articleOverride?.enabled === true;
  const bodyFontRole = overrideActive
    ? (articleOverride?.bodyFontRole ?? themeBodyRole)
    : themeBodyRole;

  const serifStack = resolveSerifHeadingStack(tokens);
  const serifResolved = mergeCustomFont(serifStack, sources.headingUpload);
  const sansResolved = mergeCustomFont(tokens.fonts.sans, sources.sansUpload);
  const monoStack =
    tokens.fonts.mono ?? getFontPreset(DEFAULT_MONO_PRESET_ID)?.stack ?? "ui-monospace, monospace";

  const bodyMerge = bodyFontRole === "serif" ? serifResolved : sansResolved;

  const customFonts = [...serifResolved.custom, ...sansResolved.custom].filter(
    (f, i, arr) => arr.findIndex((x) => x.url === f.url && x.family === f.family) === i,
  );

  return {
    bodyFontRole,
    bodyFontStack: bodyMerge.stack,
    titleFontStack: serifResolved.stack,
    uiFontStack: sansResolved.stack,
    monoFontStack: monoStack,
    bodySize,
    bodyLineHeight,
    customFonts,
  };
}

export function parseArticleTypographyOverride(metadata: Record<string, unknown> | undefined): ArticleTypographyOverride | null {
  if (!metadata?.typography || typeof metadata.typography !== "object") {
    return null;
  }
  return metadata.typography as ArticleTypographyOverride;
}

export function typographyToCssVars(config: ArticleTypographyConfig): Record<string, string> {
  return {
    "--article-font-body": config.bodyFontStack,
    "--article-font-title": config.titleFontStack,
    "--article-font-ui": config.uiFontStack,
    "--article-font-mono": config.monoFontStack,
    "--article-size-body": config.bodySize,
    "--article-leading-body": String(config.bodyLineHeight),
  };
}
