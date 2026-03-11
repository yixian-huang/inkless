import type { FieldDescriptor } from "@/types/schema";
import type { SectionMeta } from "@/theme/types";

// ---------------------------------------------------------------------------
// 1. hero
// ---------------------------------------------------------------------------
const heroSchema: Record<string, FieldDescriptor> = {
  title: { type: "string", label: "标题" },
  subtitle: { type: "string", label: "副标题" },
  label: { type: "string", label: "标签" },
  backgroundImage: { type: "mediaRef", label: "背景图片" },
  backgroundColor: { type: "color", label: "背景颜色" },
};

// ---------------------------------------------------------------------------
// 2. text-image
// ---------------------------------------------------------------------------
const textImageSchema: Record<string, FieldDescriptor> = {
  title: { type: "string", label: "标题" },
  description: { type: "string", label: "描述" },
  image: { type: "mediaRef", label: "图片" },
  imagePosition: {
    type: "enum",
    label: "图片位置",
    options: [
      { label: "左侧", value: "left" },
      { label: "右侧", value: "right" },
    ],
  },
};

// ---------------------------------------------------------------------------
// 3. card-grid
// ---------------------------------------------------------------------------
const cardGridSchema: Record<string, FieldDescriptor> = {
  title: { type: "string", label: "标题" },
  columns: {
    type: "enum",
    label: "列数",
    options: [
      { label: "2 列", value: "2" },
      { label: "3 列", value: "3" },
      { label: "4 列", value: "4" },
    ],
  },
  cards: {
    type: "array",
    label: "卡片",
    itemFields: [
      {
        type: "object",
        label: "卡片",
        fields: {
          title: { type: "string", label: "标题" },
          titleEn: { type: "string", label: "英文标题" },
          description: { type: "string", label: "描述" },
          image: { type: "mediaRef", label: "图片" },
        },
      },
    ],
  },
};

// ---------------------------------------------------------------------------
// 4. service-cards
// ---------------------------------------------------------------------------
const serviceCardsSchema: Record<string, FieldDescriptor> = {
  title: { type: "string", label: "标题" },
  services: {
    type: "array",
    label: "服务",
    itemFields: [
      {
        type: "object",
        label: "服务",
        fields: {
          title: { type: "string", label: "标题" },
          description: { type: "string", label: "描述" },
          image: { type: "mediaRef", label: "图片" },
          link: { type: "string", label: "链接" },
        },
      },
    ],
  },
};

// ---------------------------------------------------------------------------
// 5. team-grid
// ---------------------------------------------------------------------------
const teamGridSchema: Record<string, FieldDescriptor> = {
  sectionTitle: { type: "string", label: "板块标题" },
  experts: {
    type: "array",
    label: "专家",
    itemFields: [
      {
        type: "object",
        label: "专家",
        fields: {
          id: { type: "string", label: "ID", required: true },
          name: { type: "string", label: "姓名" },
          title: { type: "string", label: "头衔" },
          image: { type: "mediaRef", label: "头像" },
          bio: { type: "string", label: "简介" },
        },
      },
    ],
  },
};

// ---------------------------------------------------------------------------
// 6. checklist
// ---------------------------------------------------------------------------
const checklistSchema: Record<string, FieldDescriptor> = {
  categories: {
    type: "array",
    label: "分类",
    itemFields: [
      {
        type: "object",
        label: "分类",
        fields: {
          title: { type: "string", label: "分类标题" },
          items: {
            type: "array",
            label: "检查项",
            itemFields: [{ type: "string", label: "检查项" }],
          },
        },
      },
    ],
  },
};

// ---------------------------------------------------------------------------
// 7. contact-form
// ---------------------------------------------------------------------------
const contactFormSchema: Record<string, FieldDescriptor> = {
  title: { type: "string", label: "标题" },
  subtitle: { type: "string", label: "副标题" },
  nameLabel: { type: "string", label: "姓名标签" },
  namePlaceholder: { type: "string", label: "姓名占位符" },
  emailLabel: { type: "string", label: "邮箱标签" },
  emailPlaceholder: { type: "string", label: "邮箱占位符" },
  messageLabel: { type: "string", label: "留言标签" },
  messagePlaceholder: { type: "string", label: "留言占位符" },
  submit: { type: "string", label: "提交按钮文字" },
  phone: { type: "string", label: "电话" },
  address: { type: "string", label: "地址" },
  accentColor: { type: "color", label: "强调色" },
};

// ---------------------------------------------------------------------------
// 8. company-profile
// ---------------------------------------------------------------------------
const companyProfileSchema: Record<string, FieldDescriptor> = {
  title: { type: "string", label: "标题" },
  description: { type: "string", label: "描述" },
};

// ---------------------------------------------------------------------------
// 9. rich-text
// ---------------------------------------------------------------------------
const richTextSchema: Record<string, FieldDescriptor> = {
  content: { type: "string", label: "内容" },
  alignment: {
    type: "enum",
    label: "对齐方式",
    options: [
      { label: "左对齐", value: "left" },
      { label: "居中", value: "center" },
    ],
  },
};

// ---------------------------------------------------------------------------
// Exports
// ---------------------------------------------------------------------------

export const sectionSchemas: Record<string, Record<string, FieldDescriptor>> = {
  "hero": heroSchema,
  "text-image": textImageSchema,
  "card-grid": cardGridSchema,
  "service-cards": serviceCardsSchema,
  "team-grid": teamGridSchema,
  "checklist": checklistSchema,
  "contact-form": contactFormSchema,
  "company-profile": companyProfileSchema,
  "rich-text": richTextSchema,
};

/**
 * Convert theme sectionMeta fields to a basic FieldDescriptor schema.
 * This allows the admin editor to show fields for theme-registered sections.
 */
function sectionMetaToSchema(
  meta: SectionMeta & { fields?: Array<{ name: string; type: string; label: string }> },
): Record<string, FieldDescriptor> {
  if (!meta.fields) return {};
  const schema: Record<string, FieldDescriptor> = {};
  for (const f of meta.fields) {
    schema[f.name] = {
      type: (f.type === "number" ? "string" : f.type),
      label: f.label,
    } as FieldDescriptor;
  }
  return schema;
}

/**
 * Look up the FieldDescriptor schema for a given section type.
 * Falls back to converting theme sectionMetas if available.
 */
export function getSectionSchema(
  sectionType: string,
  themeMetas?: SectionMeta[],
): Record<string, FieldDescriptor> {
  if (sectionSchemas[sectionType]) return sectionSchemas[sectionType];

  if (themeMetas) {
    const meta = themeMetas.find((m) => m.type === sectionType) as
      SectionMeta & { fields?: Array<{ name: string; type: string; label: string }> } | undefined;
    if (meta) return sectionMetaToSchema(meta);
  }

  return {};
}

/**
 * Returns all available section types (base + theme).
 */
export function getAllSectionTypes(
  themeMetas?: SectionMeta[],
): string[] {
  const types = Object.keys(sectionSchemas);
  if (themeMetas) {
    for (const meta of themeMetas) {
      if (!types.includes(meta.type)) {
        types.push(meta.type);
      }
    }
  }
  return types;
}
