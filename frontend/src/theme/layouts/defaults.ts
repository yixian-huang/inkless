import type { LayoutConfig } from "./types";

export const CORPORATE_DEFAULT_LAYOUT: LayoutConfig = {
  type: "default",
  contentProfile: "wide",
  header: { style: "sticky" },
  footer: { style: "full" },
};

export const BLOG_DEFAULT_LAYOUT: LayoutConfig = {
  type: "default",
  contentProfile: "reading",
  header: { style: "sticky" },
  footer: { style: "minimal" },
};
