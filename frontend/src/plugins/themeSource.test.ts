import { describe, expect, it } from "vitest";
import {
  isRemoteThemeSource,
  isUninstallableThemeSource,
  themeSourceBadgeLabel,
} from "./themeSource";

describe("themeSource", () => {
  it("treats external and marketplace as remote loadable", () => {
    expect(isRemoteThemeSource("external")).toBe(true);
    expect(isRemoteThemeSource("marketplace")).toBe(true);
    expect(isRemoteThemeSource("MARKETPLACE")).toBe(true);
    expect(isRemoteThemeSource("built-in")).toBe(false);
    expect(isRemoteThemeSource("")).toBe(false);
    expect(isRemoteThemeSource(undefined)).toBe(false);
  });

  it("allows uninstall for remote sources only", () => {
    expect(isUninstallableThemeSource("marketplace")).toBe(true);
    expect(isUninstallableThemeSource("external")).toBe(true);
    expect(isUninstallableThemeSource("built-in")).toBe(false);
  });

  it("returns gallery badge labels", () => {
    expect(themeSourceBadgeLabel("marketplace")).toBe("市场");
    expect(themeSourceBadgeLabel("external")).toBe("外部");
    expect(themeSourceBadgeLabel("built-in")).toBeNull();
  });
});
