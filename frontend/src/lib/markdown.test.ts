import { describe, it, expect } from "vitest";
import { markdownToHtml, htmlToMarkdown } from "@inkless/editor";

describe("markdown via @inkless/editor", () => {
  it("round-trips a simple paragraph", () => {
    const html = markdownToHtml("hello **world**");
    expect(html).toMatch(/hello/);
    expect(htmlToMarkdown(html)).toMatch(/hello/);
  });
});
