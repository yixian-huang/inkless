import { http } from "@/api/http";

export interface QALog {
  id: number;
  question: string;
  answer: string;
  sources: unknown[];
  locale: string;
  ip_address: string;
  rating: string;
  created_at: string;
}

export interface QALogsResponse {
  items: QALog[];
  page: number;
  pageSize: number;
  total: number;
}

export interface IndexResponse {
  chunksStored: number;
  message: string;
}

function getAuthHeaders() {
  const accessToken = localStorage.getItem("accessToken");
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
}

export async function triggerQAIndex(): Promise<IndexResponse> {
  const response = await http.post<IndexResponse>("/admin/qa/index", {}, {
    headers: getAuthHeaders(),
  });
  return response.data;
}

export async function getQALogs(page = 1, pageSize = 20): Promise<QALogsResponse> {
  const response = await http.get<QALogsResponse>("/admin/qa/logs", {
    params: { page, pageSize },
    headers: getAuthHeaders(),
  });
  return response.data;
}

export async function submitQAFeedback(id: number, rating: "positive" | "negative"): Promise<void> {
  await http.post(`/admin/qa/logs/${id}/feedback`, { rating }, {
    headers: getAuthHeaders(),
  });
}
