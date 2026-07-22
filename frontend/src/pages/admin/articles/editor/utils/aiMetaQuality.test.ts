import { describe, expect, it } from "vitest";
import {
  detectScriptLang,
  evaluateAIMetaQuality,
  extractSignificantTokens,
  mergeQualityIssues,
  tokenOverlapRatio,
} from "./aiMetaQuality";

describe("detectScriptLang", () => {
  it("detects zh and en", () => {
    expect(detectScriptLang("这是中文标题")).toBe("zh");
    expect(detectScriptLang("This is English")).toBe("en");
  });
});

describe("evaluateAIMetaQuality", () => {
  const body = "Rust 异步运行时与 Tokio 调度模型深入解析。".repeat(6);

  it("flags placeholder", () => {
    const issues = evaluateAIMetaQuality({
      values: { zhTitle: "未命名" },
      bodyPlain: body,
    });
    expect(issues.some((i) => i.code === "placeholder")).toBe(true);
  });

  it("flags language mismatch", () => {
    const issues = evaluateAIMetaQuality({
      values: { zhTitle: "Production Kubernetes Guide" },
      bodyPlain: body,
    });
    expect(issues.some((i) => i.code === "language_mismatch")).toBe(true);
  });

  it("flags short meta", () => {
    const issues = evaluateAIMetaQuality({
      values: { zhMetaDescription: "太短了" },
      bodyPlain: body,
    });
    expect(issues.some((i) => i.code === "length_short")).toBe(true);
  });

  it("flags low relevance", () => {
    const issues = evaluateAIMetaQuality({
      values: {
        zhTitle: "今日天气预报",
        zhMetaDescription: "完全无关的内容描述完全无关的内容描述完全无关的内容描述",
      },
      bodyPlain: body,
    });
    expect(issues.some((i) => i.code === "low_relevance")).toBe(true);
  });

  it("accepts relevant English meta", () => {
    const enBody =
      "GraphQL schema design patterns for large APIs. ".repeat(8);
    const issues = evaluateAIMetaQuality({
      values: {
        enTitle: "GraphQL Schema Design Patterns",
        enMetaDescription:
          "Learn GraphQL schema design patterns for large APIs and production type systems.",
      },
      bodyPlain: enBody,
    });
    expect(issues.some((i) => i.code === "low_relevance")).toBe(false);
    expect(issues.some((i) => i.code === "language_mismatch")).toBe(false);
  });
});

describe("tokenOverlapRatio", () => {
  it("computes positive overlap", () => {
    const body = extractSignificantTokens("GraphQL schema design patterns");
    expect(tokenOverlapRatio("GraphQL schema", body)).toBeGreaterThan(0);
  });
});

describe("mergeQualityIssues", () => {
  it("keeps client apply-key issues and server candidate notes", () => {
    const merged = mergeQualityIssues(
      [
        { code: "length_long", field: "zhTitles[1]", message: "候选偏长", severity: "info" },
        { code: "placeholder", field: "zhTitle", message: "server placeholder", severity: "warn" },
      ],
      [
        {
          code: "placeholder",
          field: "zhTitle",
          message: "中文标题 疑似占位/模板文案",
          severity: "warn",
        },
      ],
    );
    expect(merged.some((i) => i.field === "zhTitles[1]")).toBe(true);
    expect(merged.filter((i) => i.field === "zhTitle").length).toBe(1);
    expect(merged.find((i) => i.field === "zhTitle")?.message).toContain("占位");
  });
});
