/** @type {import('tailwindcss').Config} */
export default {
    content: [
      "./index.html",
      "./src/**/*.{js,ts,jsx,tsx}",
      // Built-in themes ship utility classes outside ./src — must be scanned or layout collapses
      "./node_modules/@inkless/theme-product-first/src/**/*.{js,ts,jsx,tsx}",
      "./node_modules/@inkless/theme-blog-first/src/**/*.{js,ts,jsx,tsx}",
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
  }
