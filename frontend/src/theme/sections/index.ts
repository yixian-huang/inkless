import type { ComponentType } from "react";
import type { SectionProps, SectionMeta } from "../types";

import HeroSection from "./HeroSection";
import TextImageSection from "./TextImageSection";
import CardGridSection from "./CardGridSection";
import ServiceCardsSection from "./ServiceCardsSection";
import TeamGridSection from "./TeamGridSection";
import ChecklistSection from "./ChecklistSection";
import ContactFormSection from "./ContactFormSection";
import CompanyProfileSection from "./CompanyProfileSection";
import RichTextSection from "./RichTextSection";

export const sectionRegistry: Record<string, ComponentType<SectionProps<any>>> = {
  "hero": HeroSection,
  "text-image": TextImageSection,
  "card-grid": CardGridSection,
  "service-cards": ServiceCardsSection,
  "team-grid": TeamGridSection,
  "checklist": ChecklistSection,
  "contact-form": ContactFormSection,
  "company-profile": CompanyProfileSection,
  "rich-text": RichTextSection,
};

export const sectionMetas: SectionMeta[] = [
  { type: "hero", label: "Hero Banner", labelZh: "横幅" },
  { type: "text-image", label: "Text & Image", labelZh: "图文区块" },
  { type: "card-grid", label: "Card Grid", labelZh: "卡片网格" },
  { type: "service-cards", label: "Service Cards", labelZh: "服务卡片" },
  { type: "team-grid", label: "Team Grid", labelZh: "团队展示" },
  { type: "checklist", label: "Checklist", labelZh: "清单列表" },
  { type: "contact-form", label: "Contact Form", labelZh: "联系表单" },
  { type: "company-profile", label: "Company Profile", labelZh: "公司简介" },
  { type: "rich-text", label: "Rich Text", labelZh: "富文本" },
];

export { default as SectionRenderer } from "./SectionRenderer";

export { sectionSchemas, settingsSchema } from "../sectionSchemas";
