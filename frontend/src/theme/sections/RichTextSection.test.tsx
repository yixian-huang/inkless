import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import RichTextSection from "./RichTextSection";

describe("RichTextSection", () => {
  it("renders plain text as paragraphs (escaped)", () => {
    render(
      <RichTextSection
        data={{ content: "Hello world\n\nSecond para" }}
        settings={{}}
        variant="default"
      />,
    );
    expect(screen.getByText("Hello world")).toBeInTheDocument();
    expect(screen.getByText("Second para")).toBeInTheDocument();
  });

  it("renders HTML content as markup, not raw tags", () => {
    const { container } = render(
      <RichTextSection
        data={{
          content: "<h1>快速开始</h1><p>部署 <strong>Inkless</strong> 实例。</p>",
        }}
        settings={{}}
        variant="default"
      />,
    );
    const h1 = container.querySelector("h1");
    expect(h1).not.toBeNull();
    expect(h1?.textContent).toBe("快速开始");
    expect(container.querySelector("strong")?.textContent).toBe("Inkless");
    // Must not show literal tag strings as text-only content
    expect(container.textContent).not.toContain("<h1>");
  });

  it("strips script tags from HTML", () => {
    const { container } = render(
      <RichTextSection
        data={{ content: '<p>safe</p><script>alert(1)</script>' }}
        settings={{}}
        variant="default"
      />,
    );
    expect(container.querySelector("script")).toBeNull();
    expect(container.textContent).toContain("safe");
  });
});
