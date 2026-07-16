import { describe, expect, it } from "vitest";
import {
  ADMIN_DEFAULT_PATH,
  ADMIN_PAGES_PATH,
  getAdminRouteAccess,
  getAdminRoutePermission,
  isAdminRouteVisibleInNavigation,
} from "./adminAccess";

describe("admin route access metadata", () => {
  it("uses valid default admin destinations", () => {
    expect(ADMIN_DEFAULT_PATH).toBe("/admin");
    expect(ADMIN_PAGES_PATH).toBe("/admin/pages");
    expect(ADMIN_DEFAULT_PATH).not.toContain("/admin/content");
    expect(ADMIN_PAGES_PATH).not.toContain("/admin/content");
  });

  it("matches nested routes using canonical backend permissions", () => {
    expect(getAdminRoutePermission("/admin/pages/edit/1")).toBe("pages:read");
    expect(getAdminRoutePermission("/admin/form-submissions/12")).toBe("form_submissions:read");
    expect(getAdminRoutePermission("/admin/audit-logs")).toBe("audit_logs:read");
    expect(getAdminRoutePermission("/admin/migration")).toBe("system:manage");
  });

  it("uses the most specific capability for nested article taxonomies", () => {
    expect(getAdminRoutePermission("/admin/articles/categories")).toBe("categories:read");
    expect(getAdminRoutePermission("/admin/articles/tags")).toBe("tags:read");
  });

  it("does not let the dashboard prefix swallow child routes", () => {
    expect(getAdminRouteAccess("/admin")).toEqual({
      path: "/admin",
      permission: "dashboard:read",
      status: "production",
    });
    expect(getAdminRoutePermission("/admin/unknown")).toBeNull();
  });

  it("keeps incomplete capabilities out of navigation", () => {
    expect(isAdminRouteVisibleInNavigation("/admin/sites")).toBe(false);
    expect(isAdminRouteVisibleInNavigation("/admin/storage")).toBe(false);
    expect(isAdminRouteVisibleInNavigation("/admin/wizard")).toBe(false);
    expect(isAdminRouteVisibleInNavigation("/admin/migration")).toBe(true);
  });
});
