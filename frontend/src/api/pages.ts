import { http } from "./http";

export interface PageItem {
  id: number;
  slug: string;
  parentId?: number;
  title: { zh?: string; en?: string };
  template: string;
  status: string;
  sortOrder: number;
  themeId?: string;
  contentKey?: string;
  renderMode?: string;
  isThemePage?: boolean;
  navConfig?: { showInHeader?: boolean; showInFooter?: boolean };
  coverImage?: string;
  autoSummary?: boolean;
  allowComments?: boolean;
  pinned?: boolean;
  visibility?: string;
  publishedAt?: string | null;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface PageDetail extends PageItem {
  config: unknown;
  seoTitle?: { zh?: string; en?: string };
  seoDescription?: { zh?: string; en?: string };
}

export interface CreatePageRequest {
  slug: string;
  parentId?: number;
  title: { zh?: string; en?: string };
  template?: string;
  config?: unknown;
  seoTitle?: { zh?: string; en?: string };
  seoDescription?: { zh?: string; en?: string };
  themeId?: string;
  contentKey?: string;
  renderMode?: string;
  isThemePage?: boolean;
  navConfig?: { showInHeader?: boolean; showInFooter?: boolean };
  coverImage?: string;
  autoSummary?: boolean;
  allowComments?: boolean;
  pinned?: boolean;
  visibility?: string;
  publishedAt?: string;
  metadata?: Record<string, unknown>;
}

export type UpdatePageRequest = Partial<CreatePageRequest>;

const token = () => localStorage.getItem("accessToken") || "";

const authHeaders = () => ({
  headers: { Authorization: `Bearer ${token()}` },
});

export async function listPages(status?: string, parentId?: number) {
  const params: Record<string, string> = {};
  if (status) params.status = status;
  if (parentId !== undefined) params.parentId = String(parentId);
  const res = await http.get<{ items: PageItem[] }>("/admin/pages", {
    params,
    ...authHeaders(),
  });
  return res.data.items || [];
}

export async function getPage(id: number) {
  const res = await http.get<PageDetail>(`/admin/pages/${id}`, authHeaders());
  return res.data;
}

export async function createPage(data: CreatePageRequest) {
  const res = await http.post<PageDetail>("/admin/pages", data, authHeaders());
  return res.data;
}

export async function updatePage(id: number, data: UpdatePageRequest) {
  const res = await http.put<PageDetail>(
    `/admin/pages/${id}`,
    data,
    authHeaders()
  );
  return res.data;
}

export async function deletePage(id: number) {
  await http.delete(`/admin/pages/${id}`, authHeaders());
}

export async function publishPage(id: number) {
  const res = await http.put<PageDetail>(
    `/admin/pages/${id}/publish`,
    {},
    authHeaders()
  );
  return res.data;
}

export async function unpublishPage(id: number) {
  const res = await http.put<PageDetail>(
    `/admin/pages/${id}/unpublish`,
    {},
    authHeaders()
  );
  return res.data;
}
