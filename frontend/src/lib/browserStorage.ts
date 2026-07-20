type StorageArea = Pick<Storage, "getItem" | "setItem" | "removeItem">;

const LEGACY_TO_CANONICAL_LOCAL_KEYS = {
  accessToken: "inkless.auth.accessToken",
  refreshToken: "inkless.auth.refreshToken",
  admin_sidebar_collapsed: "inkless.admin.sidebarCollapsed",
  "impress.comment.guest": "inkless.comment.guest",
} as const;

const LEGACY_TO_CANONICAL_SESSION_KEYS = {
  "impress.setup.step": "inkless.setup.step",
  "impress.setup.draft": "inkless.setup.draft",
} as const;

export const BROWSER_STORAGE_KEYS = {
  authAccessToken: "inkless.auth.accessToken",
  authRefreshToken: "inkless.auth.refreshToken",
  adminSidebarCollapsed: "inkless.admin.sidebarCollapsed",
  adminNavGroupCollapsed: "inkless.admin.navGroupCollapsed",
  commentGuest: "inkless.comment.guest",
  setupStep: "inkless.setup.step",
  setupDraft: "inkless.setup.draft",
} as const;

function isValidToken(value: string | null): value is string {
  return typeof value === "string" && value.trim().length > 0;
}

function isValidBooleanText(value: string | null): value is "true" | "false" {
  return value === "true" || value === "false";
}

function isValidJsonObject(value: string | null): value is string {
  if (!value) return false;
  try {
    const parsed = JSON.parse(value) as unknown;
    return typeof parsed === "object" && parsed !== null && !Array.isArray(parsed);
  } catch {
    return false;
  }
}

function isValidSetupStep(value: string | null): value is string {
  return (
    value === "welcome" ||
    value === "database" ||
    value === "restart" ||
    value === "site" ||
    value === "admin" ||
    value === "content" ||
    value === "done"
  );
}

const localValidators: Record<string, (value: string | null) => value is string> = {
  [BROWSER_STORAGE_KEYS.authAccessToken]: isValidToken,
  [BROWSER_STORAGE_KEYS.authRefreshToken]: isValidToken,
  [BROWSER_STORAGE_KEYS.adminSidebarCollapsed]: isValidBooleanText,
  [BROWSER_STORAGE_KEYS.commentGuest]: isValidJsonObject,
};

const sessionValidators: Record<string, (value: string | null) => value is string> = {
  [BROWSER_STORAGE_KEYS.setupStep]: isValidSetupStep,
  [BROWSER_STORAGE_KEYS.setupDraft]: isValidJsonObject,
};

function migrateKeys(
  storage: StorageArea,
  mapping: Record<string, string>,
  validators: Record<string, (value: string | null) => value is string>,
) {
  for (const [legacyKey, canonicalKey] of Object.entries(mapping)) {
    const canonicalValue = storage.getItem(canonicalKey);
    if (validators[canonicalKey]?.(canonicalValue)) {
      storage.removeItem(legacyKey);
      continue;
    }

    const legacyValue = storage.getItem(legacyKey);
    if (validators[canonicalKey]?.(legacyValue)) {
      storage.setItem(canonicalKey, legacyValue);
    }
    storage.removeItem(legacyKey);
  }
}

export function migrateLegacyBrowserStorage() {
  if (typeof window === "undefined") return;
  migrateKeys(window.localStorage, LEGACY_TO_CANONICAL_LOCAL_KEYS, localValidators);
  migrateKeys(window.sessionStorage, LEGACY_TO_CANONICAL_SESSION_KEYS, sessionValidators);
}

export function getStoredAccessToken(): string | null {
  return localStorage.getItem(BROWSER_STORAGE_KEYS.authAccessToken);
}

export function setStoredAccessToken(token: string) {
  localStorage.setItem(BROWSER_STORAGE_KEYS.authAccessToken, token);
}

export function removeStoredAccessToken() {
  localStorage.removeItem(BROWSER_STORAGE_KEYS.authAccessToken);
}

export function getStoredRefreshToken(): string | null {
  return localStorage.getItem(BROWSER_STORAGE_KEYS.authRefreshToken);
}

export function setStoredRefreshToken(token: string) {
  localStorage.setItem(BROWSER_STORAGE_KEYS.authRefreshToken, token);
}

export function removeStoredRefreshToken() {
  localStorage.removeItem(BROWSER_STORAGE_KEYS.authRefreshToken);
}
