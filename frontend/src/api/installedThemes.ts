import { http } from "./http";

export interface InstalledThemeDTO {
  id: number;
  themeId: string;
  name: string;
  nameZh: string;
  description: string;
  author: string;
  version: string;
  source: "built-in" | "external" | "marketplace";
  externalUrl?: string;
  isActive: boolean;
  preview: string;
  config: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface ActiveThemeDTO {
  themeId: string;
  source: string;
  externalUrl?: string;
}

export async function getActiveTheme(): Promise<ActiveThemeDTO> {
  const res = await http.get<ActiveThemeDTO>("/public/active-theme");
  return res.data;
}

export async function listInstalledThemes(): Promise<InstalledThemeDTO[]> {
  const res = await http.get<InstalledThemeDTO[]>("/admin/themes", {

  });
  return res.data;
}

export async function getInstalledTheme(id: number): Promise<InstalledThemeDTO> {
  const res = await http.get<InstalledThemeDTO>(`/admin/themes/${id}`, {

  });
  return res.data;
}

export async function installTheme(data: {
  themeId: string;
  name: string;
  nameZh?: string;
  description?: string;
  author?: string;
  version?: string;
  source: "external" | "marketplace";
  externalUrl: string;
  preview?: string;
}): Promise<InstalledThemeDTO> {
  const res = await http.post<InstalledThemeDTO>("/admin/themes", data, {

  });
  return res.data;
}

export async function updateThemeConfig(id: number, config: Record<string, unknown>): Promise<InstalledThemeDTO> {
  const res = await http.put<InstalledThemeDTO>(`/admin/themes/${id}`, { config }, {

  });
  return res.data;
}

export async function activateTheme(id: number): Promise<void> {
  await http.put(`/admin/themes/${id}/activate`, null, {

  });
}

export async function uninstallTheme(id: number): Promise<void> {
  await http.delete(`/admin/themes/${id}`, {

  });
}
