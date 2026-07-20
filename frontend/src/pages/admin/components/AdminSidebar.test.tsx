import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import AdminSidebar from "./AdminSidebar";

vi.mock("@/contexts/AuthContext", () => ({
  useAuth: () => ({
    hasPermission: () => true,
  }),
}));

describe("AdminSidebar", () => {
  it("omits multi-site navigation while retaining site configuration", () => {
    render(
      <MemoryRouter initialEntries={["/admin/site-config"]}>
        <AdminSidebar
          collapsed={false}
          onToggle={vi.fn()}
          mobileOpen={false}
          onMobileClose={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(screen.queryByRole("link", { name: "站点管理" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /sites/i })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "站点配置" })).toHaveAttribute(
      "href",
      "/admin/site-config",
    );
  });

  it("groups settings under a dedicated section with hub entry", () => {
    render(
      <MemoryRouter initialEntries={["/admin"]}>
        <AdminSidebar
          collapsed={false}
          onToggle={vi.fn()}
          mobileOpen={false}
          onMobileClose={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(screen.getByRole("button", { name: /设置/i })).toBeInTheDocument();
    // Settings is default-collapsed when not on a settings route
    expect(screen.queryByRole("link", { name: "设置中心" })).not.toBeInTheDocument();
  });

  it("expands settings group when a settings route is active", () => {
    render(
      <MemoryRouter initialEntries={["/admin/settings"]}>
        <AdminSidebar
          collapsed={false}
          onToggle={vi.fn()}
          mobileOpen={false}
          onMobileClose={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "设置中心" })).toHaveAttribute(
      "href",
      "/admin/settings",
    );
    expect(screen.getByRole("link", { name: "AI 配置" })).toBeInTheDocument();
  });
});
