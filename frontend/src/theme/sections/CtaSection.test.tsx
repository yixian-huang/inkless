import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import CtaSection from "./CtaSection";

describe("CtaSection", () => {
  it("renders title and primary link", () => {
    render(
      <CtaSection
        data={{
          title: "开始使用",
          primaryLabel: "快速开始",
          primaryHref: "/p/get-started",
        }}
        settings={{}}
        variant="default"
      />,
    );
    expect(screen.getByText("开始使用")).toBeInTheDocument();
    const link = screen.getByRole("link", { name: "快速开始" });
    expect(link).toHaveAttribute("href", "/p/get-started");
  });
});
