import { describe, expect, it, vi } from "vitest";
import { ThemeManager } from "./ThemeManager";
import type { ThemePlugin } from "./types";

const theme: ThemePlugin = {
  manifest: {
    id: "external-test",
    name: "External Test",
    nameZh: "External Test",
    version: "1.0.0",
    description: "Test theme",
    descriptionZh: "Test theme",
    author: "Inkless CMS",
    type: "theme",
  },
  contractVersion: "1",
  defaultTokens: {} as ThemePlugin["defaultTokens"],
  pages: [],
};

describe("ThemeManager external registration globals", () => {
  it("registers through canonical Inkless global and cleans aliases", async () => {
    const manager = new ThemeManager();
    const appendSpy = vi.spyOn(document.head, "appendChild").mockImplementation((node: Node) => {
      queueMicrotask(() => {
        (window as typeof window & { __INKLESS_THEME_REGISTER__?: (plugin: ThemePlugin) => void })
          .__INKLESS_THEME_REGISTER__?.(theme);
      });
      return node as HTMLScriptElement;
    });

    await expect(manager.loadExternal("/theme.js")).resolves.toBe(theme);
    expect(manager.getTheme("external-test")).toBe(theme);
    expect((window as typeof window & { __INKLESS_THEME_REGISTER__?: unknown }).__INKLESS_THEME_REGISTER__).toBeUndefined();
    expect((window as typeof window & { __IMPRESS_THEME_REGISTER__?: unknown }).__IMPRESS_THEME_REGISTER__).toBeUndefined();

    appendSpy.mockRestore();
  });

  it("accepts legacy external bundles through a temporary alias", async () => {
    const manager = new ThemeManager();
    const appendSpy = vi.spyOn(document.head, "appendChild").mockImplementation((node: Node) => {
      queueMicrotask(() => {
        (window as typeof window & { __IMPRESS_THEME_REGISTER__?: (plugin: ThemePlugin) => void })
          .__IMPRESS_THEME_REGISTER__?.(theme);
      });
      return node as HTMLScriptElement;
    });

    await expect(manager.loadExternal("/legacy-theme.js")).resolves.toBe(theme);
    expect(manager.getTheme("external-test")).toBe(theme);

    appendSpy.mockRestore();
  });

  it("rejects themes with incompatible contractVersion", () => {
    const manager = new ThemeManager();
    expect(() =>
      manager.registerExternal({
        ...theme,
        manifest: { ...theme.manifest, id: "bad-contract" },
        contractVersion: "99",
      }),
    ).toThrow(/incompatible/);
    expect(manager.getTheme("bad-contract")).toBeUndefined();
  });
});
