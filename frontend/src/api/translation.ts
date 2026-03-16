import { http } from "@/api/http";

export interface TranslateRequest {
  text: string;
  sourceLang: string;
  targetLang: string;
  glossary?: Record<string, string>;
}

export interface TranslateResponse {
  translatedText: string;
  sourceLang: string;
  targetLang: string;
}

export interface GlossaryTerm {
  id: number;
  sourceTerm: string;
  targetTerm: string;
  sourceLang: string;
  targetLang: string;
  context?: string;
}

export interface GlossaryListResponse {
  items: GlossaryTerm[];
  page: number;
  pageSize: number;
  total: number;
}

function getAuthHeaders() {
  const accessToken = localStorage.getItem("accessToken");
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
}

export async function translateText(req: TranslateRequest): Promise<TranslateResponse> {
  const response = await http.post<TranslateResponse>("/admin/translate", req, {
    headers: getAuthHeaders(),
  });
  return response.data;
}

export async function getGlossary(page = 1, pageSize = 20): Promise<GlossaryListResponse> {
  const response = await http.get<GlossaryListResponse>("/admin/glossary", {
    params: { page, pageSize },
    headers: getAuthHeaders(),
  });
  return response.data;
}

export async function addGlossaryTerm(term: Omit<GlossaryTerm, "id">): Promise<GlossaryTerm> {
  const response = await http.post<GlossaryTerm>("/admin/glossary", term, {
    headers: getAuthHeaders(),
  });
  return response.data;
}

export async function deleteGlossaryTerm(id: number): Promise<void> {
  await http.delete(`/admin/glossary/${id}`, {
    headers: getAuthHeaders(),
  });
}
