import { beforeEach, describe, expect, it } from "vitest";
import { recordAIMetaEvent, summarizeAIMetaStats } from "./aiMetaTelemetry";

describe("summarizeAIMetaStats", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("computes apply rate", () => {
    recordAIMetaEvent({ type: "open" });
    recordAIMetaEvent({ type: "generate_ok", model: "m" });
    recordAIMetaEvent({ type: "generate_ok", model: "m" });
    recordAIMetaEvent({ type: "apply", applied: 3, warnCount: 1 });
    recordAIMetaEvent({ type: "feedback", feedback: "needs_edit" });

    const s = summarizeAIMetaStats();
    expect(s.generateOk).toBe(2);
    expect(s.applies).toBe(1);
    expect(s.applyRate).toBe(0.5);
    expect(s.applyWithWarnRate).toBe(1);
    expect(s.feedback.needs_edit).toBe(1);
  });
});
