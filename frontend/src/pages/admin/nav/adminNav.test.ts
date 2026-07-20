import { describe, expect, it } from "vitest";
import {
  adminRouteAccess,
  getAdminRouteAccess,
  isAdminRouteVisibleInNavigation,
} from "@/router/adminAccess";
import {
  adminNavGroups,
  filterNavGroups,
  getNavTitle,
  getSettingsHubItems,
  isAdminEditorPath,
  isNavItemActive,
} from "./adminNav";

describe("adminNav registry", () => {
  it("covers production navigation paths that exist in adminRouteAccess", () => {
    const navPaths = adminNavGroups.flatMap((g) => g.items.map((i) => i.path));
    for (const path of navPaths) {
      const access = getAdminRouteAccess(path);
      expect(access, `missing adminRouteAccess for ${path}`).not.toBeNull();
    }
  });

  it("does not include multi-site routes", () => {
    const paths = adminNavGroups.flatMap((g) => g.items.map((i) => i.path));
    expect(paths).not.toContain("/admin/sites");
  });

  it("keeps site-config in appearance group", () => {
    const appearance = adminNavGroups.find((g) => g.id === "appearance");
    expect(appearance?.items.some((i) => i.path === "/admin/site-config")).toBe(true);
  });

  it("filters experimental QA from default navigation", () => {
    const groups = filterNavGroups(() => true);
    const paths = groups.flatMap((g) => g.items.map((i) => i.path));
    expect(isAdminRouteVisibleInNavigation("/admin/qa")).toBe(false);
    expect(paths).not.toContain("/admin/qa");
  });

  it("filters by permission", () => {
    const groups = filterNavGroups((p) => p === "dashboard:read" || p === "pages:read");
    const paths = groups.flatMap((g) => g.items.map((i) => i.path));
    expect(paths).toContain("/admin");
    expect(paths).toContain("/admin/pages");
    expect(paths).not.toContain("/admin/users");
  });

  it("filters by search query", () => {
    const groups = filterNavGroups(() => true, { query: "站点" });
    const labels = groups.flatMap((g) => g.items.map((i) => i.label));
    expect(labels).toContain("站点配置");
    expect(labels).not.toContain("仪表盘");
  });

  it("resolves titles and editor paths", () => {
    expect(getNavTitle("/admin/site-config")).toBe("站点配置");
    expect(getNavTitle("/admin/articles/categories")).toBe("分类");
    expect(getNavTitle("/admin/pages/edit/12")).toBe("页面编辑");
    expect(isAdminEditorPath("/admin/articles/new")).toBe(true);
    expect(isAdminEditorPath("/admin/media")).toBe(false);
  });

  it("matches nav active states without swallowing dashboard children", () => {
    const dashboard = adminNavGroups[0].items[0];
    expect(isNavItemActive("/admin", dashboard)).toBe(true);
    expect(isNavItemActive("/admin/pages", dashboard)).toBe(false);
  });

  it("settings hub items exclude hub itself and respect permissions", () => {
    const items = getSettingsHubItems((p) => p === "settings:manage");
    expect(items.every((i) => i.id !== "settings-hub")).toBe(true);
    expect(items.some((i) => i.path === "/admin/ai-settings")).toBe(true);
    expect(items.some((i) => i.path === "/admin/audit-logs")).toBe(false);
    expect(items.some((i) => i.path === "/admin/api-keys")).toBe(false);
  });

  it("exposes API keys for media:create", () => {
    const items = getSettingsHubItems((p) => p === "media:create");
    expect(items.some((i) => i.path === "/admin/api-keys")).toBe(true);
    expect(items.some((i) => i.path === "/admin/ai-settings")).toBe(false);
  });

  it("adminRouteAccess still lists known production capabilities", () => {
    expect(adminRouteAccess.some((r) => r.path === "/admin/settings")).toBe(true);
  });
});
