import { describe, expect, it } from "vitest";
import { resolveBlogHomeBrandMode } from "./resolveHomeBrand";
import type { BrandingView } from "@/hooks/useBranding";

const base: BrandingView = {
  siteName: "黄逸仙",
  tagline: "",
  logo: { light: "/logo.svg" },
  favicon: "",
  primaryColor: "#141310",
  author: { name: "黄逸仙", avatar: "/avatar.svg", bio: "", socials: [] },
  footer: { copyright: "", extraLinks: [] },
  localeMode: "mono-zh",
  defaultLocale: "zh",
  currentLocale: "zh",
};

describe("resolveBlogHomeBrandMode", () => {
  it("does not change brand mode off home", () => {
    expect(resolveBlogHomeBrandMode("logo", base, false)).toBe("logo");
    expect(resolveBlogHomeBrandMode("text", base, false)).toBe("text");
  });

  it("hides header brand on home when hero avatar is present", () => {
    expect(resolveBlogHomeBrandMode("logo", base, true)).toBe("none");
    expect(resolveBlogHomeBrandMode("avatar", base, true)).toBe("none");
    expect(resolveBlogHomeBrandMode("text", base, true)).toBe("none");
  });

  it("keeps logo on home when there is no avatar", () => {
    const noAvatar = { ...base, author: { ...base.author, avatar: "" } };
    expect(resolveBlogHomeBrandMode("logo", noAvatar, true)).toBe("logo");
  });
});
