import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react-swc";
import { existsSync } from "node:fs";
import { resolve } from "node:path";
import AutoImport from "unplugin-auto-import/vite";

const base = process.env.BASE_PATH || "/";
const isPreview = process.env.IS_PREVIEW ? true : false;

/**
 * Local theme live-dev: point at a cloned theme repo (source entry, not dist).
 *
 *   THEME_PRODUCT_FIRST_PATH=../inkless-theme-product-first pnpm dev
 *
 * Path resolution order:
 *   1. absolute path
 *   2. relative to monorepo root (parent of frontend/)
 *   3. relative to frontend/
 * Leave unset to use the pinned github: dependency in node_modules.
 */
function resolveLocalThemeRoot(envKey: string): string | null {
  const raw = (process.env[envKey] || "").trim();
  if (!raw) return null;
  const monorepoRoot = resolve(__dirname, "..");
  const candidates = [
    resolve(raw), // absolute or cwd-relative
    resolve(monorepoRoot, raw),
    resolve(__dirname, raw),
  ];
  for (const root of candidates) {
    if (existsSync(resolve(root, "package.json"))) {
      console.info(`[vite] live theme: ${envKey} → ${root}`);
      return root;
    }
  }
  console.warn(
    `[vite] ${envKey}=${raw} not found (tried monorepo root + frontend/); ignoring`,
  );
  return null;
}

const localProductFirst = resolveLocalThemeRoot("THEME_PRODUCT_FIRST_PATH");
const localBlogFirst = resolveLocalThemeRoot("THEME_BLOG_FIRST_PATH");
const localEditorialFirm = resolveLocalThemeRoot("THEME_EDITORIAL_FIRM_PATH");

const localThemeRoots = [localProductFirst, localBlogFirst, localEditorialFirm].filter(
  (p): p is string => Boolean(p),
);

const themeAliases: Record<string, string> = {};
if (localProductFirst) {
  themeAliases["@inkless/theme-product-first"] = localProductFirst;
}
if (localBlogFirst) {
  themeAliases["@inkless/theme-blog-first"] = localBlogFirst;
}
if (localEditorialFirm) {
  themeAliases["@inkless/theme-editorial-firm"] = localEditorialFirm;
}

// https://vite.dev/config/
export default defineConfig({
  define: {
    __BASE_PATH__: JSON.stringify(base),
    __IS_PREVIEW__: JSON.stringify(isPreview),
    __READDY_PROJECT_ID__: JSON.stringify(process.env.PROJECT_ID || ""),
    __READDY_VERSION_ID__: JSON.stringify(process.env.VERSION_ID || ""),
    __READDY_AI_DOMAIN__: JSON.stringify(process.env.READDY_AI_DOMAIN || ""),
  },
  plugins: [
    react(),
    AutoImport({
      imports: [
        {
          react: [
            "React",
            "useState",
            "useEffect",
            "useContext",
            "useReducer",
            "useCallback",
            "useMemo",
            "useRef",
            "useImperativeHandle",
            "useLayoutEffect",
            "useDebugValue",
            "useDeferredValue",
            "useId",
            "useInsertionEffect",
            "useSyncExternalStore",
            "useTransition",
            "startTransition",
            "lazy",
            "memo",
            "forwardRef",
            "createContext",
            "createElement",
            "cloneElement",
            "isValidElement",
          ],
        },
        {
          "react-router-dom": [
            "useNavigate",
            "useLocation",
            "useParams",
            "useSearchParams",
            "Link",
            "NavLink",
            "Navigate",
            "Outlet",
          ],
        },
        // React i18n
        {
          "react-i18next": ["useTranslation", "Trans"],
        },
      ],
      dts: true,
    }),
  ],
  base,
  build: {
    outDir: "out",
    emptyOutDir: true,
    sourcemap: process.env.NODE_ENV !== "production",
    assetsDir: "assets",
    rollupOptions: {
      output: {
        chunkFileNames: "assets/[name]-[hash].js",
        entryFileNames: "assets/[name]-[hash].js",
        assetFileNames: "assets/[name]-[hash][extname]",
      },
    },
    chunkSizeWarningLimit: 1000,
  },
  resolve: {
    alias: {
      "@": resolve(__dirname, "./src"),
      // Editor kit package (workspace source).
      "@inkless/editor": resolve(__dirname, "../packages/editor/src/index.ts"),
      "@inkless/editor/markdown": resolve(__dirname, "../packages/editor/src/markdown.ts"),
      "@inkless/editor/extensions": resolve(
        __dirname,
        "../packages/editor/src/extensions/index.ts",
      ),
      // Theme packages import host APIs only through this facade.
      "@inkless/theme-host": resolve(__dirname, "./src/theme-host/index.ts"),
      // Optional: live-link cloned theme repos (see THEME_*_PATH above).
      ...themeAliases,
    },
    // Single React copy when theme package is outside the monorepo.
    dedupe: ["react", "react-dom", "react-router-dom", "react-i18next", "i18next"],
  },
  server: {
    port: 3000,
    host: "0.0.0.0",
    fs: {
      // Allow reading theme sources outside frontend/ when live-linked.
      allow: [resolve(__dirname, ".."), ...localThemeRoots],
    },
    watch: {
      // Ensure HMR picks up edits in linked theme repos.
      ignored: localThemeRoots.length
        ? ["**/node_modules/**", "**/.git/**"]
        : undefined,
    },
    proxy: {
      // Theme packages may call relative /public/* (not axios baseURL).
      "/public": {
        target: "http://localhost:8088",
        changeOrigin: true,
      },
      "/auth": {
        target: "http://localhost:8088",
        changeOrigin: true,
      },
      "/uploads": {
        target: "http://localhost:8088",
        changeOrigin: true,
      },
    },
  },
  test: {
    globals: true,
    environment: "happy-dom",
    setupFiles: "./src/test/setup.ts",
    include: [
      "src/**/*.test.{ts,tsx}",
      // Workspace package tests co-run with the web suite
      "../packages/editor/src/**/*.test.{ts,tsx}",
    ],
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "html"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: [
        "src/**/*.test.{ts,tsx}",
        "src/test/**",
        "src/main.tsx",
        "src/vite-env.d.ts",
      ],
    },
  },
});
