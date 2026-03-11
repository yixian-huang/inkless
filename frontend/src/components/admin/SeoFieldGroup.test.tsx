import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import SeoFieldGroup from "./SeoFieldGroup";

describe("SeoFieldGroup", () => {
  it("renders SEO fields", () => {
    render(
      <SeoFieldGroup
        seoTitle="Test Title"
        onSeoTitleChange={vi.fn()}
        metaDescription="Test desc"
        onMetaDescriptionChange={vi.fn()}
      />
    );
    expect(screen.getByDisplayValue("Test Title")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Test desc")).toBeInTheDocument();
  });

  it("shows character count", () => {
    render(
      <SeoFieldGroup
        seoTitle="Hello"
        onSeoTitleChange={vi.fn()}
        metaDescription=""
        onMetaDescriptionChange={vi.fn()}
      />
    );
    expect(screen.getByText("5/70")).toBeInTheDocument();
  });
});
