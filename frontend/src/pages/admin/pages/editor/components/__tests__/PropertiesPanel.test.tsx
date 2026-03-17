import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import PropertiesPanel from "../PropertiesPanel";

describe("PropertiesPanel", () => {
  it("renders DynamicForm when section has schema", () => {
    render(
      <PropertiesPanel
        section={{ id: "1", type: "hero", data: { title: { zh: "Hi", en: "" } }, settings: {} }}
        onDataChange={vi.fn()}
        onSettingsChange={vi.fn()}
      />
    );
    expect(screen.getByDisplayValue("Hi")).toBeInTheDocument();
    expect(screen.queryByText("数据 (JSON)")).not.toBeInTheDocument();
  });

  it("falls back to JSON editor when no schema exists", () => {
    render(
      <PropertiesPanel
        section={{ id: "1", type: "unknown-type", data: { foo: "bar" }, settings: {} }}
        onDataChange={vi.fn()}
        onSettingsChange={vi.fn()}
      />
    );
    expect(screen.getByText("数据 (JSON)")).toBeInTheDocument();
  });

  it("toggles between form and JSON mode", () => {
    render(
      <PropertiesPanel
        section={{ id: "1", type: "hero", data: { title: { zh: "Hi", en: "" } }, settings: {} }}
        onDataChange={vi.fn()}
        onSettingsChange={vi.fn()}
      />
    );
    fireEvent.click(screen.getByText("切换到 JSON 编辑"));
    expect(screen.getByText("切换到表单编辑")).toBeInTheDocument();
  });

  it("shows SectionSettings", () => {
    render(
      <PropertiesPanel
        section={{ id: "1", type: "hero", data: {}, settings: {} }}
        onDataChange={vi.fn()}
        onSettingsChange={vi.fn()}
      />
    );
    expect(screen.getByText("显示设置")).toBeInTheDocument();
  });
});
