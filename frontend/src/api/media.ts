import { http } from "@/api/http";

export interface MediaItem {
  id: number;
  url: string;
  filename: string;
  mimeType: string;
  size: number;
  width?: number;
  height?: number;
  createdAt: string;
}

interface MediaListResponse {
  items: MediaItem[];
  total: number;
  page: number;
  pageSize: number;
}

function getAuthHeaders() {
  const accessToken = localStorage.getItem("accessToken");
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
}

export async function uploadMedia(file: File | Blob, filename?: string): Promise<MediaItem> {
  const formData = new FormData();
  formData.append("file", file, filename || (file instanceof File ? file.name : "upload.jpg"));

  const response = await http.post<MediaItem>("/admin/media/upload", formData, {
    headers: getAuthHeaders(),
  });
  return response.data;
}

export async function listMedia(page: number = 1, pageSize: number = 20, type?: string): Promise<MediaListResponse> {
  const params: Record<string, any> = { page, pageSize };
  if (type) params.type = type;
  const response = await http.get<MediaListResponse>("/admin/media", {
    params,
    headers: getAuthHeaders(),
  });
  return response.data;
}

export async function deleteMedia(id: number): Promise<void> {
  await http.delete(`/admin/media/${id}`, {
    headers: getAuthHeaders(),
  });
}

export async function recropMedia(id: number, file: Blob): Promise<MediaItem> {
  const formData = new FormData();
  formData.append("file", file, "recropped.jpg");
  const response = await http.put<MediaItem>(`/admin/media/${id}/crop`, formData, {
    headers: getAuthHeaders(),
  });
  return response.data;
}

export interface MediaUsage {
  type: "article" | "page" | "content_document";
  id: string;
  title: string;
  field: string;
}

export async function getMediaUsages(id: number): Promise<MediaUsage[]> {
  const response = await http.get<{ usages: MediaUsage[] }>(`/admin/media/${id}/usages`, {
    headers: getAuthHeaders(),
  });
  return response.data.usages || [];
}

export async function renameMedia(id: number, filename: string): Promise<MediaItem> {
  const response = await http.put<MediaItem>(`/admin/media/${id}`, { filename }, {
    headers: getAuthHeaders(),
  });
  return response.data;
}
