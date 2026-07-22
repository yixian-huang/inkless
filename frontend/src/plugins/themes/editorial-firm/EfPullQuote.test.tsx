import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { EfPullQuote } from "@inkless/theme-editorial-firm";

describe("EfPullQuote", () => {
  it("renders quote and attribution", () => {
    render(<EfPullQuote data={{ quote: "Hello", attribution: "A" }} />);
    expect(screen.getByText(/Hello/)).toBeTruthy();
    expect(screen.getByText(/A/)).toBeTruthy();
  });

  it("returns null when quote is empty", () => {
    const { container } = render(<EfPullQuote data={{ quote: "" }} />);
    expect(container.firstChild).toBeNull();
  });
});
