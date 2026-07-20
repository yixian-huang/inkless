import { describe, expect, it } from "vitest";
import * as host from "./index";
import { THEME_HOST_VALUE_EXPORTS } from "./exports.inventory";
import { THEME_CONTRACT_VERSION } from "./contract";

describe("@inkless/theme-host export inventory", () => {
  it("exposes every inventory value export at runtime", () => {
    for (const name of THEME_HOST_VALUE_EXPORTS) {
      expect(host, `missing host export: ${name}`).toHaveProperty(name);
      expect((host as Record<string, unknown>)[name], name).not.toBeUndefined();
    }
  });

  it("does not expose unexpected runtime keys beyond the inventory", () => {
    const actual = Object.keys(host).sort();
    const expected = [...THEME_HOST_VALUE_EXPORTS].sort();
    expect(actual).toEqual(expected);
  });

  it("locks contract version to a supported value", () => {
    expect(THEME_CONTRACT_VERSION).toBe("1");
    expect(host.THEME_CONTRACT_SUPPORTED).toContain(THEME_CONTRACT_VERSION);
  });
});
