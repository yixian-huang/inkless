import { describe, expect, it } from "vitest";
import { markdownToHtml, htmlToMarkdown } from "./markdown";

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
    expect(html).toContain('data-type="mermaid"');
    expect(html).toContain("graph TD");
    expect(html).not.toContain("language-mermaid");
  });

  it("converts GFM tables", () => {
    const md = "| A | B |\n| --- | --- |\n| 1 | 2 |\n";
    const html = markdownToHtml(md);
    expect(html).toContain("<table");
    expect(html).toContain("<th");
    expect(html).toContain("A");
    expect(html).toContain("1");
  });
});

describe("htmlToMarkdown", () => {
  it("round-trips mermaid blocks", () => {
    const md = "Intro\n\n```mermaid\ngraph TD; A-->B\n```\n\nOutro\n";
    const html = markdownToHtml(md);
    const back = htmlToMarkdown(html);
    expect(back).toContain("```mermaid");
    expect(back).toContain("graph TD");
    expect(back).toContain("A-->B");
  });

  it("round-trips GFM tables", () => {
    const md = "| Name | Age |\n| --- | --- |\n| Ada | 36 |\n";
    const html = markdownToHtml(md);
    const back = htmlToMarkdown(html);
    expect(back).toContain("| Name | Age |");
    expect(back).toMatch(/\| --- \| --- \|/);
    expect(back).toContain("| Ada | 36 |");
  });

  it("round-trips headings and emphasis", () => {
    const md = "## Title\n\nThis is **bold** and *italic*.\n";
    const html = markdownToHtml(md);
    const back = htmlToMarkdown(html);
    expect(back).toContain("## Title");
    expect(back).toContain("**bold**");
    expect(back).toMatch(/\*italic\*/);
  });

  it("round-trips fenced code blocks with language", () => {
    const md = "```js\nconsole.log(1)\n```\n";
    const html = markdownToHtml(md);
    const back = htmlToMarkdown(html);
    expect(back).toContain("```js");
    expect(back).toContain("console.log(1)");
  });
});
