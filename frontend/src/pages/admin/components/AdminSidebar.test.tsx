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
});
