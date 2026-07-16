import { http } from "./http";

// JSON config type — matches backend JSONMap
type JSONMap = Record<string, unknown>;

export interface UnifiedPageItem {
  id: number;
  slug: string;
  zhTitle: string;
  enTitle: string;
  mode: "template" | "composable";
  templateId?: number;
  status: string;
  sortOrder: number;
  showInNav: boolean;
  parentId?: number;
  publishedVersion: number;
  draftVersion: number;
  createdAt: string;
  updatedAt: string;
}

export interface PublicUnifiedPageItem {
  id: number;
  slug: string;
  title: { zh?: string; en?: string };
  description?: { zh?: string; en?: string };
  mode: "template" | "composable";
  sortOrder: number;
  showInNav: boolean;
  parentId?: number;
  status: string;
  publishedVersion: number;
}

export interface UnifiedPageDraft {
  id: number;
  slug: string;
  draftConfig: JSONMap;
  draftVersion: number;
  publishedVersion: number;
  updatedAt: string;
}

export interface CreateUnifiedPageRequest {
  slug: string;
  zhTitle?: string;
  enTitle?: string;
  mode: "template" | "composable";
  templateId?: number;
  draftConfig?: JSONMap;
  sortOrder?: number;
  showInNav?: boolean;
  parentId?: number | null;
}

export interface UpdateUnifiedPageRequest {
  slug: string;
  zhTitle?: string;
  enTitle?: string;
  zhDescription?: string;
  enDescription?: string;
  sortOrder?: number;
  showInNav?: boolean;
  parentId?: number;
  zhMetaTitle?: string;
  enMetaTitle?: string;
  zhMetaDescription?: string;
  enMetaDescription?: string;
  zhMetaKeywords?: string;
  enMetaKeywords?: string;
}

// Admin CRUD
export const listUnifiedPages = (status?: string, mode?: string) => {
  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (mode) params.set("mode", mode);
  return http.get<{ items: UnifiedPageItem[] }>(`/admin/pages?${params}`).then((r) => r.data.items ?? []);
};

export const getUnifiedPage = (id: number) =>
  http.get<UnifiedPageItem>(`/admin/pages/${id}`, {}).then((r) => r.data);

export const createUnifiedPage = (data: CreateUnifiedPageRequest) =>
  http.post<UnifiedPageItem>("/admin/pages", data, {}).then((r) => r.data);

export const updateUnifiedPage = (id: number, data: UpdateUnifiedPageRequest) =>
  http.put<UnifiedPageItem>(`/admin/pages/${id}`, data, {}).then((r) => r.data);

export const deleteUnifiedPage = (id: number) =>
  http.delete(`/admin/pages/${id}`, {});

// Draft
export const getUnifiedPageDraft = (id: number) =>
  http.get<UnifiedPageDraft>(`/admin/pages/${id}/draft`, {}).then((r) => r.data);

export const updateUnifiedPageDraft = (id: number, version: number, draftConfig: JSONMap) =>
  http.put(`/admin/pages/${id}/draft`, { draftConfig }, {
    headers: { "If-Match": String(version) },
  }).then((r) => r.data);

// Publish / Unpublish / Rollback
export const publishUnifiedPage = (id: number, expectedDraftVersion: number) =>
  http.post(`/admin/pages/${id}/publish`, { expectedDraftVersion }, {}).then((r) => r.data);

export const unpublishUnifiedPage = (id: number) =>
  http.post(`/admin/pages/${id}/unpublish`, {}, {}).then((r) => r.data);

export const rollbackUnifiedPage = (id: number, targetVersion: number) =>
  http.post(`/admin/pages/${id}/rollback`, { targetVersion }, {}).then((r) => r.data);

// Version history
export const listUnifiedPageVersions = (id: number, page = 1, pageSize = 20) =>
  http.get(`/admin/pages/${id}/versions?page=${page}&pageSize=${pageSize}`, {}).then((r) => r.data);

export const getUnifiedPageVersion = (id: number, version: number) =>
  http.get(`/admin/pages/${id}/versions/${version}`, {}).then((r) => r.data);
