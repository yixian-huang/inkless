import { describe, expect, it } from "vitest";
import {
  assertThemeContractCompatible,
  isThemeContractCompatible,
  resolveThemeContractVersion,
  THEME_CONTRACT_VERSION,
} from "./contract";

describe("theme contract version", () => {
  it("accepts current contract version", () => {
    expect(isThemeContractCompatible(THEME_CONTRACT_VERSION)).toBe(true);
    expect(isThemeContractCompatible("1")).toBe(true);
  });

  it("rejects unknown and empty versions", () => {
    expect(isThemeContractCompatible("99")).toBe(false);
    expect(isThemeContractCompatible("")).toBe(false);
    expect(isThemeContractCompatible(null)).toBe(false);
  });

  it("defaults missing version to 1 while host supports 1", () => {
    expect(resolveThemeContractVersion(undefined)).toBe("1");
    expect(resolveThemeContractVersion(null)).toBe("1");
  });

  it("assertThemeContractCompatible throws on incompatible", () => {
    expect(() => assertThemeContractCompatible("99", "bad-theme")).toThrow(
      /bad-theme.*incompatible/,
    );
  });

  it("assertThemeContractCompatible accepts missing as legacy 1", () => {
    expect(() => assertThemeContractCompatible(undefined, "old-umd")).not.toThrow();
  });
});
