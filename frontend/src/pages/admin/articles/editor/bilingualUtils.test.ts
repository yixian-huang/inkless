import { describe, expect, it } from "vitest";
import { countWords, plainTextToHtml } from "./bilingualUtils";

describe("countWords", () => {
  it("counts empty as zero", () => {
    expect(countWords("")).toEqual({ chars: 0, words: 0 });
  });

  it("counts CJK and latin", () => {
    const r = countWords("你好 world");
    expect(r.chars).toBeGreaterThan(0);
    expect(r.words).toBe(3); // 你 好 world — CJK each + 1 latin
  });
});

describe("plainTextToHtml", () => {
  it("wraps paragraphs", () => {
    expect(plainTextToHtml("a\n\nb")).toContain("<p>a</p>");
    expect(plainTextToHtml("a\n\nb")).toContain("<p>b</p>");
  });
});
