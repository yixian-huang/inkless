import { defineConfig } from "vitepress";

export default defineConfig({
  title: "Inkless CMS",
  description: "A bilingual CMS built with Go and React",
  lang: "en-US",

  themeConfig: {
    nav: [
      { text: "Guide", link: "/guide/getting-started" },
      { text: "API", link: "/api/" },
      { text: "GitHub", link: "https://github.com/yixian-huang/inkless" },
    ],

    sidebar: [
      {
        text: "Guide",
        items: [
          { text: "Getting Started", link: "/guide/getting-started" },
          { text: "Blog-first Mode", link: "/guide/blog-first" },
          { text: "Theme Layout", link: "/guide/theme-layout" },
          { text: "Theme Market", link: "/guide/theme-market" },
          { text: "Host Updates", link: "/guide/host-updates" },
          { text: "Architecture", link: "/guide/architecture" },
          { text: "Extension Points", link: "/guide/extension-points" },
          { text: "Your First Plugin", link: "/guide/first-plugin" },
        ],
      },
    ],

    socialLinks: [
      { icon: "github", link: "https://github.com/yixian-huang/inkless" },
    ],
  },
});
