import { describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";

function mock(seo: Record<string, unknown>, identityName: Record<string, unknown> = { zh: "My Site" }) {
  vi.doMock("@/contexts/GlobalConfigContext", () => ({
    useGlobalConfig: () => ({
      config: {
        siteConfig: {
          identity: { name: identityName, localeMode: "bilingual", defaultLocale: "zh" },
          brand: { logo: { light: "" }, favicon: "", ogImage: "https://example.com/og.png", primaryColor: "#000" },
          author: { name: "", socials: [] },
          footer: {},
          seo,
        },
      },
      locale: "zh",
      loading: false,
      features: {},
      refetch: vi.fn(),
    }),
  }));
}

describe("useSEODefaults", () => {
  it("falls back to identity.name for default title", async () => {
    vi.resetModules();
    mock({});
    const { useSEODefaults } = await import("./useSEODefaults");
    const { result } = renderHook(() => useSEODefaults());
    expect(result.current.defaultTitle).toBe("My Site");
  });

  it("uses titleTemplate '{page} | {site}' by default", async () => {
    vi.resetModules();
    mock({});
    const { useSEODefaults } = await import("./useSEODefaults");
    const { result } = renderHook(() => useSEODefaults());
    expect(result.current.buildTitle("Blog")).toBe("Blog | My Site");
  });

  it("respects configured titleTemplate", async () => {
    vi.resetModules();
    mock({ titleTemplate: "{site} — {page}" });
    const { useSEODefaults } = await import("./useSEODefaults");
    const { result } = renderHook(() => useSEODefaults());
    expect(result.current.buildTitle("Blog")).toBe("My Site — Blog");
  });

  it("buildTitle with empty page returns site only", async () => {
    vi.resetModules();
    mock({});
    const { useSEODefaults } = await import("./useSEODefaults");
    const { result } = renderHook(() => useSEODefaults());
    expect(result.current.buildTitle("")).toBe("My Site");
  });

  it("buildTitle does not double-append site when page equals site name", async () => {
    vi.resetModules();
    mock({ titleTemplate: "{page} · {site}" });
    const { useSEODefaults } = await import("./useSEODefaults");
    const { result } = renderHook(() => useSEODefaults());
    expect(result.current.buildTitle("My Site")).toBe("My Site");
  });
});
