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
