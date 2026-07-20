/** Read a string field from a version/draft snapshot map. */
export function snapStr(snap: Record<string, unknown> | undefined, key: string): string {
  if (!snap) return "";
  const v = snap[key];
  return typeof v === "string" ? v : v == null ? "" : String(v);
}
