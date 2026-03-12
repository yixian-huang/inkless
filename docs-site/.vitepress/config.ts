import { defineConfig } from "vitepress";

export default defineConfig({
  title: "Impress CMS",
  description: "A bilingual CMS built with Go and React",
  lang: "en-US",

  themeConfig: {
    nav: [
      { text: "Guide", link: "/guide/getting-started" },
      { text: "API", link: "/api/" },
      { text: "GitHub", link: "https://github.com/your-org/impress" },
    ],

    sidebar: [
      {
        text: "Guide",
        items: [
          { text: "Getting Started", link: "/guide/getting-started" },
          { text: "Architecture", link: "/guide/architecture" },
          { text: "Extension Points", link: "/guide/extension-points" },
          { text: "Your First Plugin", link: "/guide/first-plugin" },
        ],
      },
    ],

    socialLinks: [
      { icon: "github", link: "https://github.com/your-org/impress" },
    ],
  },
});
