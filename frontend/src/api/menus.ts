import { http } from "@/api/http";

export interface MenuItem {
  id: number;
  groupId: number;
  parentId?: number | null;
  zhName: string;
  enName: string;
  type: "custom_link" | "article" | "page" | "category" | "tag";
  target: "_self" | "_parent" | "_blank" | "_top";
  url: string;
  refId?: number | null;
  refSlug: string;
  visible?: boolean | null;
  metadata: Record<string, unknown>;
  sortOrder: number;
  children?: MenuItem[];
  createdAt: string;
  updatedAt: string;
}

export interface MenuGroup {
  id: number;
  name: string;
  slug: string;
  isPrimary: boolean;
  items?: MenuItem[];
  createdAt: string;
  updatedAt: string;
}

function getAuthHeaders() {
  const accessToken = localStorage.getItem("accessToken");
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
}

// Admin APIs
export async function listMenuGroups(): Promise<MenuGroup[]> {
  const res = await http.get<{ items: MenuGroup[] }>("/admin/menus", {
    headers: getAuthHeaders(),
  });
  return res.data.items || [];
}

export async function createMenuGroup(data: { name: string; slug: string }): Promise<MenuGroup> {
  const res = await http.post<MenuGroup>("/admin/menus", data, {
    headers: getAuthHeaders(),
  });
  return res.data;
}

export async function getMenuGroup(id: number): Promise<MenuGroup> {
  const res = await http.get<MenuGroup>(`/admin/menus/${id}`, {
    headers: getAuthHeaders(),
  });
  return res.data;
}

export async function updateMenuGroup(id: number, data: { name?: string; slug?: string }): Promise<MenuGroup> {
  const res = await http.put<MenuGroup>(`/admin/menus/${id}`, data, {
    headers: getAuthHeaders(),
  });
  return res.data;
}

export async function deleteMenuGroup(id: number): Promise<void> {
  await http.delete(`/admin/menus/${id}`, {
    headers: getAuthHeaders(),
  });
}

export async function setMenuGroupPrimary(id: number): Promise<void> {
  await http.put(`/admin/menus/${id}/primary`, {}, {
    headers: getAuthHeaders(),
  });
}

export async function createMenuItem(groupId: number, data: Partial<MenuItem>): Promise<MenuItem> {
  const res = await http.post<MenuItem>(`/admin/menus/${groupId}/items`, data, {
    headers: getAuthHeaders(),
  });
  return res.data;
}

export async function updateMenuItem(groupId: number, itemId: number, data: Partial<MenuItem>): Promise<MenuItem> {
  const res = await http.put<MenuItem>(`/admin/menus/${groupId}/items/${itemId}`, data, {
    headers: getAuthHeaders(),
  });
  return res.data;
}

export async function deleteMenuItem(groupId: number, itemId: number): Promise<void> {
  await http.delete(`/admin/menus/${groupId}/items/${itemId}`, {
    headers: getAuthHeaders(),
  });
}

export async function reorderMenuItems(groupId: number, itemIds: number[]): Promise<void> {
  await http.put(`/admin/menus/${groupId}/items/reorder`, { itemIds }, {
    headers: getAuthHeaders(),
  });
}

// Public API
export async function getPublicMenu(): Promise<MenuGroup | null> {
  const res = await http.get<MenuGroup>("/public/menu");
  return res.data;
}
