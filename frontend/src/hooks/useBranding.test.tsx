import { describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";

vi.mock("@/contexts/GlobalConfigContext", () => ({
  useGlobalConfig: () => ({
    config: {
      siteConfig: {
        identity: {
          name: { zh: "我的博客", en: "My Blog" },
          localeMode: "bilingual",
          defaultLocale: "zh",
        },
        brand: { logo: { light: "/logo.png" }, favicon: "/fav.ico", ogImage: "", primaryColor: "#000" },
        author: { name: "Isian", socials: [{ kind: "github", url: "https://github.com/isian" }] },
        footer: { copyright: { zh: "© 2026 我的博客" }, icp: "京 ICP 备 123" },
        seo: {},
      },
    },
    loading: false,
    locale: "zh",
    features: {},
    refetch: vi.fn(),
  }),
}));

import { useBranding } from "./useBranding";

const Wrapper = ({ children }: { children: ReactNode }) => <>{children}</>;

describe("useBranding", () => {
  it("returns localized site name based on current locale", () => {
    const { result } = renderHook(() => useBranding(), { wrapper: Wrapper });
    expect(result.current.siteName).toBe("我的博客");
  });

  it("exposes logo and ICP", () => {
    const { result } = renderHook(() => useBranding(), { wrapper: Wrapper });
    expect(result.current.logo.light).toBe("/logo.png");
    expect(result.current.footer.icp).toBe("京 ICP 备 123");
  });
});
