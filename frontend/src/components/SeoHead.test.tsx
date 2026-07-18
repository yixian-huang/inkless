// @vitest-environment jsdom
// @vitest-environment-options { "url": "https://alias.example/current" }

import { render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import SeoHead from "./SeoHead";

function canonicalLink(): HTMLLinkElement {
  const link = document.querySelector<HTMLLinkElement>('link[rel="canonical"]');
  if (!link) {
    throw new Error("canonical link is missing");
  }
  return link;
}

describe("SeoHead canonical URLs", () => {
  beforeEach(() => {
    document.head.innerHTML =
      '<link rel="canonical" href="https://primary.example/server-path">';
  });

  afterEach(() => {
    document.head.innerHTML = "";
  });

  it("keeps the server canonical origin across SPA updates without leaking paths", () => {
    expect(window.location.origin).toBe("https://alias.example");

    const firstPage = render(<SeoHead canonicalUrl="/blog" />);
    expect(canonicalLink()).toHaveAttribute("href", "https://primary.example/blog");

    firstPage.rerender(<SeoHead canonicalUrl="/tags" />);
    expect(canonicalLink()).toHaveAttribute("href", "https://primary.example/tags");

    firstPage.unmount();
    expect(document.querySelector('link[rel="canonical"]')).toBeNull();

    const nextPage = render(<SeoHead canonicalUrl="/categories" />);
    expect(canonicalLink()).toHaveAttribute(
      "href",
      "https://primary.example/categories",
    );
    nextPage.unmount();
    expect(document.querySelector('link[rel="canonical"]')).toBeNull();
  });

  it("honors absolute canonical URLs without replacing the server origin", () => {
    const page = render(
      <SeoHead canonicalUrl="https://external.example/explicit" />,
    );
    expect(canonicalLink()).toHaveAttribute(
      "href",
      "https://external.example/explicit",
    );

    page.rerender(<SeoHead canonicalUrl="/blog" />);
    expect(canonicalLink()).toHaveAttribute("href", "https://primary.example/blog");
  });
});
