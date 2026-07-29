import { http } from "./http";

export interface APIKey {
  id: number;
  userId: number;
  name: string;
  tokenPrefix: string;
  scopes: string[];
  lastUsedAt?: string | null;
  createdAt: string;
}

export interface CreateAPIKeyRequest {
  name: string;
  scopes?: string[];
}

export interface CreateAPIKeyResponse {
  token: string;
  key: APIKey;
}

/** Mirrors backend AllowedAPIKeyScopes (keep in sync). */
export const API_KEY_SCOPE_OPTIONS: {
  value: string;
  label: string;
  group: "media" | "articles" | "pages" | "taxonomy";
  caution?: boolean;
}[] = [
  { value: "media:create", label: "上传媒体", group: "media" },
  { value: "media:read", label: "读取媒体库", group: "media" },
  { value: "articles:read", label: "读取文章", group: "articles" },
  { value: "articles:create", label: "创建文章", group: "articles" },
  { value: "articles:update", label: "更新文章 / AI 元数据", group: "articles" },
  { value: "articles:publish", label: "发布文章", group: "articles", caution: true },
  { value: "articles:delete", label: "删除文章", group: "articles", caution: true },
  { value: "pages:read", label: "读取页面", group: "pages" },
  { value: "pages:create", label: "创建页面", group: "pages" },
  { value: "pages:update", label: "更新页面草稿", group: "pages" },
  { value: "pages:publish", label: "发布页面", group: "pages", caution: true },
  { value: "pages:delete", label: "删除页面", group: "pages", caution: true },
  { value: "categories:read", label: "读取分类", group: "taxonomy" },
  { value: "categories:create", label: "创建分类", group: "taxonomy" },
  { value: "categories:update", label: "更新分类", group: "taxonomy" },
  { value: "tags:read", label: "读取标签", group: "taxonomy" },
  { value: "tags:create", label: "创建标签", group: "taxonomy" },
  { value: "tags:update", label: "更新标签", group: "taxonomy" },
];

/** Recommended presets for common clients. */
export const API_KEY_SCOPE_PRESETS: {
  id: string;
  label: string;
  description: string;
  scopes: string[];
}[] = [
  {
    id: "picgo",
    label: "PicGo 上传",
    description: "仅媒体上传",
    scopes: ["media:create"],
  },
  {
    id: "content-agent",
    label: "内容 Agent（推荐）",
    description: "读写文章/页面 SEO 与正文，不含发布",
    scopes: [
      "articles:read",
      "articles:update",
      "pages:read",
      "pages:update",
      "media:create",
      "categories:read",
      "tags:read",
    ],
  },
  {
    id: "content-agent-publish",
    label: "内容 Agent + 发布",
    description: "在推荐基础上允许 create/publish（请慎用）",
    scopes: [
      "articles:read",
      "articles:create",
      "articles:update",
      "articles:publish",
      "pages:read",
      "pages:create",
      "pages:update",
      "pages:publish",
      "media:create",
      "categories:read",
      "tags:read",
      "tags:create",
    ],
  },
];

export async function listAPIKeys(): Promise<APIKey[]> {
  const res = await http.get<{ items: APIKey[] }>("/admin/api-keys");
  return res.data.items ?? [];
}

export async function createAPIKey(body: CreateAPIKeyRequest): Promise<CreateAPIKeyResponse> {
  const res = await http.post<CreateAPIKeyResponse>("/admin/api-keys", body);
  return res.data;
}

export async function revokeAPIKey(id: number): Promise<void> {
  await http.delete(`/admin/api-keys/${id}`);
}
