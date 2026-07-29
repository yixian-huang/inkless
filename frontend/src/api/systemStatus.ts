import { http } from "@/api/http";

export interface SystemApplicationInfo {
  version: string;
  updateCapable?: boolean;
  updateBlockedReason?: string;
  selfUpdateEnabled?: boolean;
}

export interface SystemRuntimeInfo {
  goVersion: string;
  os: string;
  arch: string;
  cpuCount: number;
  goroutines: number;
  uptime: number;
}

export interface SystemMemoryInfo {
  allocMB: number;
  totalAllocMB: number;
  sysMB: number;
  gcPauseMs: number;
}

export interface SystemDatabaseInfo {
  type: string;
  healthy: boolean;
  status: string;
  error?: string;
  openConnections: number;
  maxOpenConnections: number;
  inUse: number;
  idle: number;
}

export interface SystemStorageInfo {
  type: string;
  healthy: boolean;
  status: string;
  error?: string;
  uploadDirSizeMB: number;
  uploadDirBytes: number;
  mediaCount: number;
}

export interface SystemContentCounts {
  articles: number;
  pages: number;
  media: number;
  users: number;
}

export interface SystemStatusResponse {
  application: SystemApplicationInfo;
  runtime: SystemRuntimeInfo;
  memory: SystemMemoryInfo;
  database: SystemDatabaseInfo;
  storage: SystemStorageInfo;
  content: SystemContentCounts;
}

export async function getSystemStatus(): Promise<SystemStatusResponse> {
  const response = await http.get<SystemStatusResponse>("/admin/system/status");
  return response.data;
}

// ── Host self-update (H0/H1) ──

export interface HostReleaseAsset {
  name: string;
  url: string;
  sha256?: string;
  size?: number;
}

export interface HostReleaseInfo {
  version: string;
  publishedAt?: string;
  notesUrl?: string;
  newer?: boolean;
  assets?: HostReleaseAsset[];
  prerelease?: boolean;
}

export interface HostUpdateJob {
  id: string;
  kind: string;
  status: string;
  fromVersion?: string;
  toVersion?: string;
  startedAt: string;
  finishedAt?: string;
  error?: string;
  phase?: string;
  message?: string;
}

export interface HostUpdateStatus {
  enabled: boolean;
  capable: boolean;
  blockedReason?: string;
  currentVersion: string;
  channel: string;
  releaseRoot?: string;
  systemdUnit?: string;
  repo?: string;
  latest?: HostReleaseInfo | null;
  lastCheckAt?: string;
  lastJob?: HostUpdateJob | null;
  localVersions?: string[];
  hasPrevious?: boolean;
  checks?: Record<string, boolean>;
}

export interface HostUpdateProbeResult {
  checkedAt: string;
  channel: string;
  source?: string;
  latest?: HostReleaseInfo | null;
  error?: string;
}

export async function getHostUpdateStatus(): Promise<HostUpdateStatus> {
  const res = await http.get<HostUpdateStatus>("/admin/system/update");
  return res.data;
}

export async function checkHostUpdate(): Promise<HostUpdateProbeResult> {
  const res = await http.post<HostUpdateProbeResult>("/admin/system/update/check");
  return res.data;
}

export async function applyHostUpdate(version?: string): Promise<HostUpdateJob> {
  const res = await http.post<HostUpdateJob>("/admin/system/update/apply", {
    version: version || undefined,
  });
  return res.data;
}

export async function rollbackHostUpdate(to?: string): Promise<HostUpdateJob> {
  const res = await http.post<HostUpdateJob>("/admin/system/update/rollback", {
    to: to || "previous",
  });
  return res.data;
}
