import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import FaqSection from "./FaqSection";

describe("FaqSection", () => {
  it("renders questions", () => {
    render(
      <FaqSection
        data={{
          title: "常见问题",
          items: [{ question: "如何部署？", answer: "使用 Docker 或二进制。" }],
        }}
        settings={{}}
        variant="default"
      />,
    );
    expect(screen.getByText("常见问题")).toBeInTheDocument();
    expect(screen.getByText("如何部署？")).toBeInTheDocument();
  });
});
