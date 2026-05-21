import { lazy, type ReactNode } from "react";
import { useGlobalConfig } from "@/contexts/GlobalConfigContext";
import type { SiteConfigFeatures } from "@/types/siteConfig";

const NotFound = lazy(() => import("@/pages/NotFound"));

export interface FeatureGateProps {
  feature: keyof SiteConfigFeatures["publicPages"];
  children: ReactNode;
  fallback?: ReactNode;
}

export function FeatureGate({ feature, children, fallback }: FeatureGateProps) {
  const { features } = useGlobalConfig();
  // Old-deploy compat: missing record → render as enabled.
  // Missing key within an existing record → render as disabled.
  const enabled = !features || !features.publicPages
    ? true
    : features.publicPages[feature] === true;
  if (!enabled) return <>{fallback ?? <NotFound />}</>;
  return <>{children}</>;
}
