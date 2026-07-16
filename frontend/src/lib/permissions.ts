const canonicalLegacyAliases: Record<string, string[]> = {
  form_submissions: ["form-submissions"],
  themes: ["theme"],
  audit_logs: ["audit-logs"],
  settings: ["site-config", "features", "email-settings", "storage", "translation", "qa"],
  system: ["migration"],
};

export function hasGrantedPermission(grants: string[] | undefined, requested: string): boolean {
  if (!grants?.length) return false;
  if (grants.includes("*:*") || grants.includes(requested)) return true;

  const [resource, action] = requested.includes(":")
    ? requested.split(":", 2)
    : [requested, ""];

  if (grants.includes(`${resource}:*`)) return true;

  const legacyAliases = canonicalLegacyAliases[resource] ?? [];
  if (grants.includes(resource) || legacyAliases.some((alias) => grants.includes(alias))) {
    return true;
  }

  if (!action) {
    return grants.some((grant) => grant.startsWith(`${resource}:`));
  }

  return false;
}
