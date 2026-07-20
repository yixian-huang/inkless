import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import { resolve } from "node:path";

/**
 * Library build for remote install (UMD) and standalone ESM dist.
 * Host built-in path imports package source via workspace (no prebuild).
 */
export default defineConfig({
  plugins: [react()],
  // Browser UMD must not reference Node `process` (register smoke / remote install).
  define: {
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
  build: {
    lib: {
      entry: resolve(__dirname, "src/register.ts"),
      name: "InklessThemeBlogFirst",
      formats: ["umd", "es"],
      fileName: (format) => (format === "umd" ? "theme.umd.js" : "theme.es.js"),
    },
    outDir: "dist",
    emptyOutDir: true,
    sourcemap: true,
    rollupOptions: {
      // Single-file remote install: no async chunks for home page.
      output: [
        {
          format: "umd",
          name: "InklessThemeBlogFirst",
          entryFileNames: "theme.umd.js",
          inlineDynamicImports: true,
          globals: {
            react: "React",
            "react-dom": "ReactDOM",
            "react-router-dom": "ReactRouterDOM",
            "react-i18next": "ReactI18next",
            // Host publishes window.InklessThemeHost via plugins/externals.ts
            "@inkless/theme-host": "InklessThemeHost",
          },
        },
        {
          format: "es",
          entryFileNames: "theme.es.js",
          inlineDynamicImports: true,
          // Preserve external host + peers for tree-shaking consumers.
        },
      ],
      external: [
        "react",
        "react-dom",
        "react-router-dom",
        "react-i18next",
        "@inkless/theme-host",
      ],
    },
  },
  resolve: {
    alias: {
      // When building in isolation without host sources, map to a stub.
      // In monorepo, prefer real host if available (UMD still externalizes it).
      "@inkless/theme-host": resolve(__dirname, "../../frontend/src/theme-host/index.ts"),
    },
  },
});
