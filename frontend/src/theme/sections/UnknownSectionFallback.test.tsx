import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import UnknownSectionFallback from "./UnknownSectionFallback";

describe("UnknownSectionFallback", () => {
  it("renders theme-prefixed message for pf-* types", () => {
    render(<UnknownSectionFallback type="pf-steps" detailed />);
    expect(screen.getByTestId("unknown-section-fallback")).toHaveAttribute(
      "data-section-type",
      "pf-steps",
    );
    expect(screen.getByText(/对应主题/)).toBeInTheDocument();
    expect(screen.getByText(/type: pf-steps/)).toBeInTheDocument();
  });

  it("renders generic unknown message for host-like typos", () => {
    render(<UnknownSectionFallback type="not-a-real-section" />);
    expect(screen.getByText("未知页面区块")).toBeInTheDocument();
  });
});
