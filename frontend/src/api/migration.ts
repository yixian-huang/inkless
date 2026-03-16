import { http } from "@/api/http";

export type MigrationFormat = "wordpress" | "halo" | "markdown";
export type MigrationJobPhase = "parsing" | "importing" | "done" | "failed";

export interface MigrationJob {
  jobId: string;
  source: MigrationFormat;
  phase: MigrationJobPhase;
  total: number;
  processed: number;
  succeeded: number;
  failed: number;
  errors: string[];
  startedAt: string;
  finishedAt?: string;
}

export interface MigrationJobsResponse {
  jobs: MigrationJob[];
}

function getAuthHeaders() {
  const accessToken = localStorage.getItem("accessToken");
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
}

export async function importData(
  file: File,
  format: MigrationFormat
): Promise<{ jobId: string; message: string; totalArticles: number; parseErrors: string[] }> {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("source", format);
  const response = await http.post<{ jobId: string; message: string; totalArticles: number; parseErrors: string[] }>(
    "/admin/migration/import",
    formData,
    { headers: getAuthHeaders() }
  );
  return response.data;
}

export async function getMigrationJobs(): Promise<MigrationJob[]> {
  const response = await http.get<MigrationJobsResponse>("/admin/migration/jobs", {
    headers: getAuthHeaders(),
  });
  return response.data.jobs || [];
}

export async function getMigrationJob(jobId: string): Promise<MigrationJob> {
  const response = await http.get<MigrationJob>(`/admin/migration/jobs/${jobId}`, {
    headers: getAuthHeaders(),
  });
  return response.data;
}

export function createMigrationJobStream(jobId: string): EventSource {
  const accessToken = localStorage.getItem("accessToken");
  const url = `/admin/migration/jobs/${jobId}/stream${
    accessToken ? `?token=${encodeURIComponent(accessToken)}` : ""
  }`;
  return new EventSource(url);
}
