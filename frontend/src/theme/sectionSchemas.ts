import type { FieldSchema } from "@/theme/types";

export const sectionSchemas: Record<string, FieldSchema[]> = {
  hero: [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "subtitle", type: "bilingual", label: "副标题" },
    { key: "label", type: "bilingual", label: "标签文字" },
    { key: "backgroundImage", type: "media", label: "背景图" },
    { key: "backgroundColor", type: "color", label: "背景色" },
  ],
  "text-image": [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "description", type: "bilingual-textarea", label: "描述" },
    { key: "image", type: "media", label: "图片" },
    { key: "imagePosition", type: "select", label: "图片位置", options: [
      { label: "左侧", value: "left" },
      { label: "右侧", value: "right" },
    ]},
  ],
  "card-grid": [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "columns", type: "select", label: "列数", options: [
      { label: "2列", value: 2 },
      { label: "3列", value: 3 },
      { label: "4列", value: 4 },
    ]},
    { key: "cards", type: "array", label: "卡片", itemSchema: [
      { key: "title", type: "bilingual", label: "标题" },
      { key: "titleEn", type: "text", label: "英文标题 (legacy)" },
      { key: "description", type: "bilingual-textarea", label: "描述" },
      { key: "image", type: "media", label: "图片" },
    ]},
  ],
  "service-cards": [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "items", type: "array", label: "服务项", itemSchema: [
      { key: "title", type: "bilingual", label: "标题" },
      { key: "description", type: "bilingual-textarea", label: "描述" },
      { key: "image", type: "media", label: "图片" },
      { key: "link", type: "text", label: "链接" },
    ]},
  ],
  "team-grid": [
    { key: "sectionTitle", type: "bilingual", label: "区块标题" },
    { key: "experts", type: "array", label: "专家列表", itemSchema: [
      { key: "id", type: "text", label: "ID", hidden: true },
      { key: "name", type: "bilingual", label: "姓名" },
      { key: "title", type: "bilingual", label: "职位" },
      { key: "image", type: "media", label: "头像" },
      { key: "bio", type: "bilingual-textarea", label: "简介" },
    ]},
  ],
  checklist: [
    { key: "categories", type: "array", label: "分类列表", itemSchema: [
      { key: "title", type: "bilingual", label: "分类标题" },
      { key: "items", type: "string-array", label: "检查项" },
    ]},
  ],
  "contact-form": [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "subtitle", type: "bilingual", label: "副标题" },
    { key: "nameLabel", type: "bilingual", label: "姓名标签" },
    { key: "namePlaceholder", type: "bilingual", label: "姓名占位符" },
    { key: "emailLabel", type: "bilingual", label: "邮箱标签" },
    { key: "emailPlaceholder", type: "bilingual", label: "邮箱占位符" },
    { key: "messageLabel", type: "bilingual", label: "留言标签" },
    { key: "messagePlaceholder", type: "bilingual", label: "留言占位符" },
    { key: "submit", type: "bilingual", label: "提交按钮文字" },
    { key: "phone", type: "bilingual", label: "电话" },
    { key: "email", type: "bilingual", label: "联系邮箱" },
    { key: "address", type: "bilingual", label: "地址" },
    { key: "accentColor", type: "color", label: "主题色" },
  ],
  "company-profile": [
    { key: "title", type: "bilingual", label: "标题" },
    { key: "description", type: "bilingual-textarea", label: "描述" },
    { key: "descriptions", type: "array", label: "段落列表", itemSchema: [
      { key: "zh", type: "textarea", label: "中文" },
      { key: "en", type: "textarea", label: "英文" },
    ]},
    { key: "image", type: "media", label: "图片" },
  ],
  "rich-text": [
    { key: "content", type: "bilingual-textarea", label: "内容" },
    { key: "alignment", type: "select", label: "对齐方式", options: [
      { label: "左对齐", value: "left" },
      { label: "居中", value: "center" },
    ]},
  ],
};

export const settingsSchema: FieldSchema[] = [
  { key: "background", type: "select", label: "背景", options: [
    { label: "默认", value: "surface" },
    { label: "交替", value: "surface-alt" },
    { label: "主题色", value: "primary" },
  ]},
  { key: "padding", type: "select", label: "内边距", options: [
    { label: "无", value: "none" },
    { label: "小", value: "sm" },
    { label: "中", value: "md" },
    { label: "大", value: "lg" },
  ]},
  { key: "maxWidth", type: "select", label: "最大宽度", options: [
    { label: "标准", value: "layout" },
    { label: "全宽", value: "full" },
  ]},
  { key: "hidden", type: "boolean", label: "隐藏" },
];
