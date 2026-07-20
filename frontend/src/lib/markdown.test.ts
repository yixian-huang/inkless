import { describe, expect, it } from "vitest";
import { markdownToHtml } from "./markdown";

describe("markdownToHtml", () => {
  it("renders basic markdown", () => {
    const html = markdownToHtml("# Hello\n\n**bold**");
    expect(html).toContain("<h1");
    expect(html).toContain("Hello");
    expect(html).toContain("<strong>bold</strong>");
  });

  it("converts mermaid fences to renderable divs", () => {
    const source = "```mermaid\ngraph TD; A-->B\n```";
    const html = markdownToHtml(source);
    expect(html).toContain('class="mermaid"');
    expect(html).toContain("graph TD");
    expect(html).not.toContain("language-mermaid");
  });
});
