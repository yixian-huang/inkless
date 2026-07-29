import { describe, expect, it } from "vitest";
import {
  buildHostPagePreset,
  isHostPagePresetId,
  listHostPagePresets,
} from "./pagePresets";

describe("pagePresets", () => {
  it("lists three host presets", () => {
    expect(listHostPagePresets()).toHaveLength(3);
    expect(isHostPagePresetId("doc-simple")).toBe(true);
    expect(isHostPagePresetId("nope")).toBe(false);
  });

  it("builds doc-simple with reading layout", () => {
    const cfg = buildHostPagePreset("doc-simple", {
      zhTitle: "政策",
      enTitle: "Policy",
    });
    expect(cfg.layout).toBe("reading");
    expect(cfg.showPageHeader).toBe(true);
    expect(cfg.sections).toHaveLength(1);
    expect(cfg.sections[0].type).toBe("rich-text");
    expect(cfg.sections[0].settings?.maxWidth).toBe("reading");
  });

  it("builds doc-guide with compact hero", () => {
    const cfg = buildHostPagePreset("doc-guide", { zhTitle: "上手" });
    expect(cfg.layout).toBe("landing");
    expect(cfg.sections[0].type).toBe("hero");
    expect(cfg.sections[0].variant).toBe("compact");
    expect(cfg.sections.some((s) => s.type === "checklist")).toBe(true);
  });

  it("builds landing-use-cases with cards", () => {
    const cfg = buildHostPagePreset("landing-use-cases", {
      zhTitle: "用例",
    });
    expect(cfg.sections.some((s) => s.type === "card-grid")).toBe(true);
  });
});
