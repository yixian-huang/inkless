/**
 * Local SectionProps-compatible shape.
 * Host SectionRenderer already locale-resolves `data` to plain strings.
 * Avoid deep `@/` imports from the theme package.
 * Keep settings union aligned with host `SectionSettings` for ThemePlugin assignability.
 */
export interface SectionSettings {
  background?: "surface" | "surface-alt" | "primary" | string;
  padding?: "none" | "sm" | "md" | "lg";
  maxWidth?: "layout" | "full" | string;
  hidden?: boolean;
}

export interface SectionProps<T = Record<string, unknown>> {
  data: T;
  settings?: SectionSettings;
  variant?: string;
}

/** Coerce unknown values to display strings; never pass objects to React children. */
export function asString(value: unknown, fallback = ""): string {
  if (value == null) return fallback;
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return fallback;
}

export function asBool(value: unknown, fallback = false): boolean {
  if (typeof value === "boolean") return value;
  if (value === "true") return true;
  if (value === "false") return false;
  return fallback;
}

export function asArray<T = Record<string, unknown>>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}
