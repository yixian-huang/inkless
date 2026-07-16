import { describe, expect, it } from "vitest";
import { hasGrantedPermission } from "./permissions";

describe("hasGrantedPermission", () => {
  it("supports exact and wildcard RBAC grants", () => {
    expect(hasGrantedPermission(["pages:read"], "pages:read")).toBe(true);
    expect(hasGrantedPermission(["pages:*"], "pages:publish")).toBe(true);
    expect(hasGrantedPermission(["*:*"], "system:manage")).toBe(true);
  });

  it("supports resource checks and legacy aliases", () => {
    expect(hasGrantedPermission(["articles:update"], "articles")).toBe(true);
    expect(hasGrantedPermission(["form-submissions"], "form_submissions:read")).toBe(true);
    expect(hasGrantedPermission(["theme"], "themes:read")).toBe(true);
    expect(hasGrantedPermission(["audit-logs"], "audit_logs:read")).toBe(true);
  });

  it("does not treat an unrelated grant as access", () => {
    expect(hasGrantedPermission(["articles:read"], "settings:manage")).toBe(false);
    expect(hasGrantedPermission([], "pages:read")).toBe(false);
  });
});
