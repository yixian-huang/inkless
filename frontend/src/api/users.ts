import { http } from "./http";

export interface UserDTO {
  id: number;
  username: string;
  role: string;
  isSuperAdmin: boolean;
  permissions: string[];
  createdAt: string;
  updatedAt: string;
}

export interface UserListResponse {
  items: UserDTO[];
  total: number;
  page: number;
}

export interface CreateUserRequest {
  username: string;
  password: string;
  role: string;
  permissions: string[];
}

export interface UpdateUserRequest {
  username?: string;
  password?: string;
  role?: string;
  permissions?: string[];
}

function getAuthHeaders() {
  const accessToken = localStorage.getItem("accessToken");
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
}

export async function listUsers(page = 1, pageSize = 20) {
  const res = await http.get<UserListResponse>("/admin/users", {
    params: { page, pageSize },
    headers: getAuthHeaders(),
  });
  return res.data;
}

export async function createUser(data: CreateUserRequest) {
  const res = await http.post<UserDTO>("/admin/users", data, {
    headers: getAuthHeaders(),
  });
  return res.data;
}

export async function updateUser(id: number, data: UpdateUserRequest) {
  const res = await http.put<UserDTO>(`/admin/users/${id}`, data, {
    headers: getAuthHeaders(),
  });
  return res.data;
}

export async function deleteUser(id: number) {
  await http.delete(`/admin/users/${id}`, {
    headers: getAuthHeaders(),
  });
}
