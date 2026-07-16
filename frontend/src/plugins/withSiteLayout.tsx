import type { ReactNode } from "react";
import SiteLayout from "@/theme/layouts/PublicLayout";

/** Wrap arbitrary public route elements in SiteLayout (static routes). */
export function withSiteLayout(element: ReactNode) {
  return <SiteLayout>{element}</SiteLayout>;
}
