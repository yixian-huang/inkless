/**
 * Admin field schemas for editorial-firm sections.
 * Uses host FieldType vocabulary without importing host types
 * (package type-check isolation).
 */

export type FieldType =
  | "text"
  | "textarea"
  | "bilingual"
  | "bilingual-textarea"
  | "media"
  | "color"
  | "select"
  | "number"
  | "boolean"
  | "array"
  | "string-array";

export interface FieldSchema {
  key: string;
  type: FieldType;
  label: string;
  placeholder?: string;
  defaultValue?: unknown;
  hidden?: boolean;
  options?: { label: string; value: string | number }[];
  itemSchema?: FieldSchema[];
}

export const editorialFirmSectionSchemas: Record<string, FieldSchema[]> = {
  "ef-hero-editorial": [
    { key: "kicker", type: "bilingual", label: "引语 / Kicker" },
    { key: "title", type: "bilingual", label: "标题" },
    { key: "deck", type: "bilingual-textarea", label: "导语 / Deck" },
    { key: "image", type: "media", label: "主图" },
    { key: "ctaLabel", type: "bilingual", label: "按钮文字" },
    { key: "ctaHref", type: "text", label: "按钮链接" },
    {
      key: "layout",
      type: "select",
      label: "布局",
      options: [
        { label: "全幅", value: "full" },
        { label: "分栏", value: "split" },
      ],
      defaultValue: "full",
    },
  ],
  "ef-pull-quote": [
    { key: "quote", type: "bilingual-textarea", label: "引文" },
    { key: "attribution", type: "bilingual", label: "出处" },
  ],
  "ef-feature-split": [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "body", type: "bilingual-textarea", label: "正文" },
    { key: "image", type: "media", label: "图片" },
    {
      key: "imageSide",
      type: "select",
      label: "图片位置",
      options: [
        { label: "左侧", value: "left" },
        { label: "右侧", value: "right" },
      ],
      defaultValue: "left",
    },
    { key: "caption", type: "bilingual", label: "图注" },
  ],
  "ef-service-index": [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "intro", type: "bilingual-textarea", label: "引言" },
    {
      key: "items",
      type: "array",
      label: "服务项",
      itemSchema: [
        { key: "title", type: "bilingual", label: "标题" },
        { key: "summary", type: "bilingual-textarea", label: "摘要" },
        { key: "href", type: "text", label: "链接" },
      ],
    },
  ],
  "ef-mosaic": [
    { key: "title", type: "bilingual", label: "标题" },
    {
      key: "tiles",
      type: "array",
      label: "图块",
      itemSchema: [
        { key: "image", type: "media", label: "图片" },
        { key: "label", type: "bilingual", label: "标签" },
        { key: "href", type: "text", label: "链接" },
      ],
    },
  ],
  "ef-cta-band": [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "body", type: "bilingual-textarea", label: "说明" },
    { key: "ctaLabel", type: "bilingual", label: "按钮文字" },
    { key: "ctaHref", type: "text", label: "按钮链接" },
  ],
  "ef-contact-split": [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "intro", type: "bilingual-textarea", label: "引言" },
    { key: "phone", type: "bilingual", label: "电话" },
    { key: "email", type: "bilingual", label: "邮箱" },
    { key: "address", type: "bilingual-textarea", label: "地址" },
    { key: "showForm", type: "boolean", label: "显示表单", defaultValue: true },
    { key: "nameLabel", type: "bilingual", label: "姓名标签" },
    { key: "emailLabel", type: "bilingual", label: "邮箱标签" },
    { key: "messageLabel", type: "bilingual", label: "留言标签" },
    { key: "submitLabel", type: "bilingual", label: "提交按钮" },
  ],
  "ef-rich-text": [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "body", type: "bilingual-textarea", label: "正文" },
  ],
};
