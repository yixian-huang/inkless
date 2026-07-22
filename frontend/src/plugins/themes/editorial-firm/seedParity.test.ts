import { describe, expect, it } from "vitest";
import { editorialFirmPageConfigs } from "@inkless/theme-editorial-firm";
// Host Go embed — keep in lockstep with pinned theme package seeds.
import hostSeeds from "../../../../../backend/internal/builtinthemes/editorial_firm_seeds.json";

/**
 * When changing defaults: update theme repo first, bump pin, then update
 * backend/internal/builtinthemes/editorial_firm_seeds.json.
 */
function sectionFingerprint(
  pages: Record<string, { sections: Array<{ id: string; type: string }> }>,
): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const key of Object.keys(pages).sort()) {
    out[key] = (pages[key]?.sections ?? []).map((s) => `${s.type}:${s.id}`);
  }
  return out;
}

describe("editorial-firm host seed parity (pinned package)", () => {
  it("package pageConfigs match backend editorial_firm_seeds.json fingerprints", () => {
    expect(
      sectionFingerprint(
        hostSeeds as Record<string, { sections: Array<{ id: string; type: string }> }>,
      ),
    ).toEqual(
      sectionFingerprint(
        editorialFirmPageConfigs as Record<
          string,
          { sections: Array<{ id: string; type: string }> }
        >,
      ),
    );
  });
});
