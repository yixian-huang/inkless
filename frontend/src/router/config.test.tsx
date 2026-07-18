import { describe, expect, it } from "vitest";
import { matchRoutes } from "react-router-dom";
import { staticRoutes } from "./config";

describe("admin route configuration", () => {
  it("routes removed multi-site URLs to the standard fallback", () => {
    const matches = matchRoutes(staticRoutes, "/admin/sites");

    expect(matches?.at(-1)?.route.path).toBe("*");
  });

  it("keeps the single-site configuration page routable", () => {
    const matches = matchRoutes(staticRoutes, "/admin/site-config");

    expect(matches?.at(-1)?.route.path).toBe("site-config");
  });
});
