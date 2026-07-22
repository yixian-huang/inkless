import type { ComponentType } from "react";
import EfHeroEditorial from "./EfHeroEditorial";
import EfPullQuote from "./EfPullQuote";
import EfFeatureSplit from "./EfFeatureSplit";
import EfServiceIndex from "./EfServiceIndex";
import EfMosaic from "./EfMosaic";
import EfCtaBand from "./EfCtaBand";
import EfContactSplit from "./EfContactSplit";
import EfRichText from "./EfRichText";
import type { SectionProps } from "./types";

export { EfHeroEditorial, EfPullQuote, EfFeatureSplit, EfServiceIndex };
export { EfMosaic, EfCtaBand, EfContactSplit, EfRichText };
export { editorialFirmSectionSchemas } from "./schemas";
export type { FieldSchema } from "./schemas";

// Loosely typed map — host ThemePlugin expects SectionProps from @/theme/types;
// components use a structurally compatible local SectionProps.
export const editorialFirmSections = {
  "ef-hero-editorial": EfHeroEditorial,
  "ef-pull-quote": EfPullQuote,
  "ef-feature-split": EfFeatureSplit,
  "ef-service-index": EfServiceIndex,
  "ef-mosaic": EfMosaic,
  "ef-cta-band": EfCtaBand,
  "ef-contact-split": EfContactSplit,
  "ef-rich-text": EfRichText,
} as Record<string, ComponentType<SectionProps<any>>>;

export const editorialFirmSectionMetas = [
  { type: "ef-hero-editorial", label: "Editorial Hero", labelZh: "编辑主视觉" },
  { type: "ef-pull-quote", label: "Pull Quote", labelZh: "大引文" },
  { type: "ef-feature-split", label: "Feature Split", labelZh: "图文分栏" },
  { type: "ef-service-index", label: "Service Index", labelZh: "服务目录" },
  { type: "ef-mosaic", label: "Image Mosaic", labelZh: "图片拼贴" },
  { type: "ef-cta-band", label: "CTA Band", labelZh: "行动号召" },
  { type: "ef-contact-split", label: "Contact Split", labelZh: "联系分栏" },
  { type: "ef-rich-text", label: "Rich Text", labelZh: "长文阅读" },
];
