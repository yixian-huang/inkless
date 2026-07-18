import { describe, expect, it } from "vitest";
import {
  ADMIN_DEFAULT_PATH,
  ADMIN_PAGES_PATH,
  adminRouteAccess,
  getAdminRouteAccess,
  getAdminRoutePermission,
  hasAdminRoutePermission,
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
    expect(getAdminRoutePermission("/admin/system-status")).toBe("system:manage");
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

  it("supports OR permission gates for shared publish workflows", () => {
    expect(getAdminRoutePermission("/admin/scheduled-publications")).toEqual([
      "pages:publish",
      "articles:publish",
    ]);
    expect(hasAdminRoutePermission(["pages:publish", "articles:publish"], (permission) => permission === "articles:publish")).toBe(true);
    expect(hasAdminRoutePermission(["pages:publish", "articles:publish"], (permission) => permission === "pages:read")).toBe(false);
  });

  it("removes multi-site access metadata while keeping site config production-ready", () => {
    expect(adminRouteAccess).not.toContainEqual(
      expect.objectContaining({ path: "/admin/sites" }),
    );
    expect(getAdminRouteAccess("/admin/sites")).toBeNull();
    expect(getAdminRouteAccess("/admin/site-config")).toEqual({
      path: "/admin/site-config",
      permission: "settings:manage",
      status: "production",
    });
    expect(isAdminRouteVisibleInNavigation("/admin/site-config")).toBe(true);
  });

  it("keeps supported capabilities visible in navigation", () => {
    expect(isAdminRouteVisibleInNavigation("/admin/storage")).toBe(true);
    expect(isAdminRouteVisibleInNavigation("/admin/wizard")).toBe(true);
    expect(isAdminRouteVisibleInNavigation("/admin/ai-settings")).toBe(true);
    expect(isAdminRouteVisibleInNavigation("/admin/migration")).toBe(true);
    expect(isAdminRouteVisibleInNavigation("/admin/system-status")).toBe(true);
  });
});
