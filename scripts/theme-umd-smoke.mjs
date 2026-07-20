#!/usr/bin/env node
/**
 * Smoke-load packages/theme-blog-first/dist/theme.umd.js in a browser-like
 * page with host/peer globals stubbed. Asserts register + contractVersion.
 *
 * Usage (from repo root, after build):
 *   node scripts/theme-umd-smoke.mjs
 *   pnpm theme:umd:smoke
 */
import { chromium } from "playwright";
import { existsSync, readFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const umdCandidates = [
  resolve(root, "frontend/node_modules/@inkless/theme-blog-first/dist/theme.umd.js"),
  resolve(root, "node_modules/@inkless/theme-blog-first/dist/theme.umd.js"),
  resolve(root, "packages/theme-blog-first/dist/theme.umd.js"),
];
const manifestCandidates = [
  resolve(root, "frontend/node_modules/@inkless/theme-blog-first/inkless.theme.json"),
  resolve(root, "node_modules/@inkless/theme-blog-first/inkless.theme.json"),
  resolve(root, "packages/theme-blog-first/inkless.theme.json"),
];
const umdPath = umdCandidates.find((p) => existsSync(p));
const manifestPath = manifestCandidates.find((p) => existsSync(p));
if (!umdPath) {
  // fall through to old error after assignment below
}


function fail(msg) {
  console.error(`theme-umd-smoke: ${msg}`);
  process.exit(1);
}

if (!umdPath || !existsSync(umdPath)) {
  fail(
    `missing UMD bundle — run: pnpm --filter @inkless/theme-blog-first build (looked in node_modules and packages/)`,
  );
}
if (!manifestPath || !existsSync(manifestPath)) {
  fail("missing inkless.theme.json for theme-blog-first");
}

const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
const expectedContract = String(manifest.contractVersion ?? "");
if (!expectedContract) {
  fail("inkless.theme.json missing contractVersion");
}

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage();

page.on("pageerror", (err) => {
  console.error("pageerror:", err.message);
});
page.on("console", (msg) => {
  if (msg.type() === "error") console.error("console.error:", msg.text());
});

await page.setContent("<!doctype html><html><head></head><body></body></html>");

// Install peer/host globals on the live document (more reliable than initScript
// alone for Playwright setContent in this environment).
await page.evaluate(() => {
  window.process = window.process || { env: { NODE_ENV: "production" } };

  const noop = () => null;
  const passthrough = new Proxy(function () {}, {
    get: () => passthrough,
    apply: () => null,
  });

  window.React = {
    createElement: noop,
    Fragment: "Fragment",
    useState: (v) => [v, noop],
    useEffect: noop,
    useMemo: (fn) => fn(),
    useCallback: (fn) => fn,
    useRef: (v) => ({ current: v }),
    useContext: () => ({}),
    lazy: (fn) => fn,
    Suspense: passthrough,
    forwardRef: (fn) => fn,
  };
  window.ReactDOM = { createRoot: () => ({ render: noop }) };
  window.ReactRouterDOM = {
    Link: passthrough,
    useNavigate: () => noop,
    useLocation: () => ({ pathname: "/" }),
    useParams: () => ({}),
    useSearchParams: () => [new URLSearchParams(), noop],
  };
  window.ReactI18next = {
    useTranslation: () => ({ t: (k) => k, i18n: { language: "zh" } }),
    Trans: passthrough,
  };
  window.InklessThemeHost = {
    THEME_CONTRACT_VERSION: "1",
    THEME_CONTRACT_SUPPORTED: ["1"],
    BLOG_DEFAULT_LAYOUT: {
      type: "default",
      contentProfile: "reading",
      header: { style: "sticky" },
      footer: { style: "minimal" },
    },
    BaseSiteHeader: passthrough,
    BrandMark: passthrough,
    HeaderUtilities: passthrough,
    useHeaderSettings: () => ({ brandMode: "text", showRssLink: true, showSocials: false }),
    useBranding: () => ({
      siteName: "Smoke",
      tagline: "",
      logo: { light: "" },
      favicon: "",
      primaryColor: "#000",
      author: { name: "Smoke", avatar: "", bio: "", socials: [] },
      footer: { copyright: "", extraLinks: [] },
      localeMode: "mono-zh",
      defaultLocale: "zh",
      currentLocale: "zh",
    }),
    useContentMaxWidth: () => "40rem",
    useIsReadingLayout: () => true,
    useIsThemeHomePath: () => true,
    useGlobalConfig: () => ({ config: {}, features: {}, locale: "zh" }),
    useSEODefaults: () => ({
      defaultTitle: "Smoke",
      defaultDescription: "",
      defaultOgImage: "",
      buildTitle: (t) => t,
    }),
    useLocaleMode: () => ({
      localeMode: "mono-zh",
      defaultLocale: "zh",
      currentLocale: "zh",
    }),
    SeoHead: passthrough,
    BlogPageShell: passthrough,
    AuthorIntro: passthrough,
    ArticleList: passthrough,
    AuthorSocialLinks: passthrough,
    ArticleAdjacentNav: passthrough,
    getPublicArticles: async () => ({ items: [], total: 0, page: 1, pageSize: 10 }),
    pickLocaleValue: ({ value }) =>
      typeof value === "string" ? value : value?.zh || value?.en || "",
    SITE_CONFIG_GLOBAL_DEFAULT: {},
  };
  window.__INKLESS_SHARED__ = {
    React: window.React,
    ReactDOM: window.ReactDOM,
    ReactRouterDOM: window.ReactRouterDOM,
    ReactI18next: window.ReactI18next,
    host: window.InklessThemeHost,
  };
  window.__REGISTERED_THEME__ = null;
  window.__INKLESS_THEME_REGISTER__ = (theme) => {
    window.__REGISTERED_THEME__ = theme;
  };
});

const hostReady = await page.evaluate(() => !!window.InklessThemeHost?.THEME_CONTRACT_VERSION);
if (!hostReady) {
  fail("failed to install InklessThemeHost stub before UMD load");
}

await page.addScriptTag({ path: umdPath });

// Give the UMD IIFE a tick to run
await page.waitForTimeout(100);

const result = await page.evaluate(() => {
  const theme = window.__REGISTERED_THEME__;
  if (!theme) return { ok: false, error: "register callback was not invoked" };
  return {
    ok: true,
    id: theme.manifest?.id,
    contractVersion: theme.contractVersion,
    hasPages: Array.isArray(theme.pages) && theme.pages.length > 0,
    pageSlugs: (theme.pages || []).map((p) => p.slug),
  };
});

await browser.close();

if (!result.ok) {
  fail(result.error || "unknown failure");
}

if (result.id !== "blog-first") {
  fail(`expected theme id blog-first, got ${result.id}`);
}
if (String(result.contractVersion) !== expectedContract) {
  fail(
    `contractVersion mismatch: UMD=${result.contractVersion} manifest=${expectedContract}`,
  );
}
if (!result.hasPages) {
  fail("theme.pages is empty");
}

console.log(
  JSON.stringify(
    {
      status: "ok",
      themeId: result.id,
      contractVersion: result.contractVersion,
      pages: result.pageSlugs,
      umd: umdPath,
    },
    null,
    2,
  ),
);
