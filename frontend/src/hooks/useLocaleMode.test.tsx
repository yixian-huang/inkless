import { describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";

function mockGlobalConfig(localeMode: string, defaultLocale: "zh" | "en", currentLocale: "zh" | "en") {
  vi.doMock("@/contexts/GlobalConfigContext", () => ({
    useGlobalConfig: () => ({
      config: {
        siteConfig: {
          identity: { name: { zh: "x" }, localeMode, defaultLocale },
          brand: { logo: { light: "" }, favicon: "", ogImage: "", primaryColor: "#000" },
          author: { name: "", socials: [] },
          footer: {},
          seo: {},
        },
      },
      locale: currentLocale,
      loading: false,
      features: {},
      refetch: vi.fn(),
    }),
  }));
}

describe("useLocaleMode", () => {
  it("mono-zh: available=['zh'], isMono=true", async () => {
    vi.resetModules();
    mockGlobalConfig("mono-zh", "zh", "zh");
    const { useLocaleMode: hook } = await import("./useLocaleMode");
    const { result } = renderHook(() => hook());
    expect(result.current.available).toEqual(["zh"]);
    expect(result.current.isMono).toBe(true);
  });

  it("bilingual: available=['zh','en'], isMono=false", async () => {
    vi.resetModules();
    mockGlobalConfig("bilingual", "zh", "en");
    const { useLocaleMode: hook } = await import("./useLocaleMode");
    const { result } = renderHook(() => hook());
    expect(result.current.available).toEqual(["zh", "en"]);
    expect(result.current.isMono).toBe(false);
    expect(result.current.currentLocale).toBe("en");
  });

  it("mono-en collapses currentLocale to en even if context says zh", async () => {
    vi.resetModules();
    mockGlobalConfig("mono-en", "en", "zh");
    const { useLocaleMode: hook } = await import("./useLocaleMode");
    const { result } = renderHook(() => hook());
    expect(result.current.currentLocale).toBe("en");
  });
});
