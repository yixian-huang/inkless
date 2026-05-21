import { describe, expect, it } from "vitest";
import { pickLocaleValue } from "./locale";

describe("pickLocaleValue", () => {
  it("returns empty string for undefined", () => {
    expect(
      pickLocaleValue({
        value: undefined,
        mode: "bilingual",
        defaultLocale: "zh",
        currentLocale: "zh",
      })
    ).toBe("");
  });

  it("mono-zh ignores en", () => {
    expect(
      pickLocaleValue({
        value: { zh: "中文", en: "English" },
        mode: "mono-zh",
        defaultLocale: "zh",
        currentLocale: "zh",
      })
    ).toBe("中文");
  });

  it("mono-zh returns empty if zh missing", () => {
    expect(
      pickLocaleValue({
        value: { en: "English" },
        mode: "mono-zh",
        defaultLocale: "zh",
        currentLocale: "zh",
      })
    ).toBe("");
  });

  it("bilingual prefers currentLocale", () => {
    expect(
      pickLocaleValue({
        value: { zh: "中文", en: "English" },
        mode: "bilingual",
        defaultLocale: "zh",
        currentLocale: "en",
      })
    ).toBe("English");
  });

  it("bilingual falls back to defaultLocale", () => {
    expect(
      pickLocaleValue({
        value: { zh: "中文" },
        mode: "bilingual",
        defaultLocale: "zh",
        currentLocale: "en",
      })
    ).toBe("中文");
  });

  it("bilingual final fallback is the other language", () => {
    expect(
      pickLocaleValue({
        value: { en: "Only English" },
        mode: "bilingual",
        defaultLocale: "zh",
        currentLocale: "zh",
      })
    ).toBe("Only English");
  });
});
