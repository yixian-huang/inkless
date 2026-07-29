import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import StepsSection from "./StepsSection";

describe("StepsSection", () => {
  it("renders numbered steps", () => {
    render(
      <StepsSection
        data={{
          title: "三步上手",
          steps: [
            { title: "安装", description: "部署实例" },
            { title: "配置", description: "填写品牌" },
          ],
        }}
        settings={{}}
        variant="default"
      />,
    );
    expect(screen.getByText("三步上手")).toBeInTheDocument();
    expect(screen.getByText("安装")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
  });
});
