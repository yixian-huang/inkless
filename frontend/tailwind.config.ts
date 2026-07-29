import { resolve } from "node:path";
import { existsSync } from "node:fs";

/** Same env keys as vite.config.ts — keep Tailwind scanning live-linked theme src. */
function localThemeSrcGlobs(): string[] {
  const keys = [
    "THEME_PRODUCT_FIRST_PATH",
    "THEME_BLOG_FIRST_PATH",
    "THEME_EDITORIAL_FIRM_PATH",
  ] as const;
  const monorepoRoot = resolve(__dirname, "..");
  const globs: string[] = [];
  for (const key of keys) {
    const raw = (process.env[key] || "").trim();
    if (!raw) continue;
    const candidates = [resolve(raw), resolve(monorepoRoot, raw), resolve(__dirname, raw)];
    for (const root of candidates) {
      if (existsSync(resolve(root, "package.json"))) {
        globs.push(resolve(root, "src/**/*.{js,ts,jsx,tsx}"));
        break;
      }
    }
  }
  return globs;
}

/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
    // Built-in themes ship utility classes outside ./src — must be scanned or layout collapses
    "./node_modules/@inkless/theme-product-first/src/**/*.{js,ts,jsx,tsx}",
    "./node_modules/@inkless/theme-blog-first/src/**/*.{js,ts,jsx,tsx}",
    "./node_modules/@inkless/theme-editorial-firm/src/**/*.{js,ts,jsx,tsx}",
    // Live-linked theme repos (THEME_*_PATH)
    ...localThemeSrcGlobs(),
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: "var(--color-primary)",
          dark: "var(--color-primary-dark)",
        },
        accent: {
          DEFAULT: "var(--color-accent)",
          hover: "var(--color-accent-hover)",
        },
        surface: {
          DEFAULT: "var(--color-surface)",
          alt: "var(--color-surface-alt)",
        },
        "on-primary": "var(--color-on-primary)",
        "on-surface": {
          DEFAULT: "var(--color-on-surface)",
          muted: "var(--color-on-surface-muted)",
        },
        border: "var(--color-border)",
      },
      fontFamily: {
        sans: "var(--font-sans)",
        heading: "var(--font-heading)",
        mono: "var(--font-mono, ui-monospace, monospace)",
      },
      maxWidth: {
        layout: "var(--layout-max-width)",
      },
      borderRadius: {
        card: "var(--radius-card)",
        button: "var(--radius-button)",
      },
      padding: {
        content: "var(--layout-content-padding)",
        section: "var(--layout-section-spacing)",
        "section-sm": "calc(var(--layout-section-spacing) * 0.4)",
        "section-lg": "calc(var(--layout-section-spacing) * 1.2)",
      },
      gap: {
        content: "var(--layout-content-gap)",
      },
    },
  },
  plugins: [],
};
