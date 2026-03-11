import { http } from "./http";
import type { ThemeTokens } from "@/theme";

const token = () => localStorage.getItem("accessToken") || "";

const authHeaders = () => ({
  headers: { Authorization: `Bearer ${token()}` },
});

interface ThemeGetResponse {
  draftConfig: ThemeTokens;
  draftVersion: number;
  publishedConfig: ThemeTokens;
  publishedVersion: number;
}

interface ThemeUpdateResponse {
  draftConfig: ThemeTokens;
  draftVersion: number;
  message: string;
}

export async function getThemeSettings() {
  const res = await http.get<ThemeGetResponse>("/admin/theme", authHeaders());
  return res.data;
}

export async function updateThemeSettings(
  config: ThemeTokens,
  draftVersion: number
) {
  const res = await http.put<ThemeUpdateResponse>(
    "/admin/theme",
    { config, draftVersion },
    authHeaders()
  );
  return res.data;
}
