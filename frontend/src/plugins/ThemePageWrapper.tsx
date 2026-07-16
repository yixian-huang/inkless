import { lazy, Suspense, useMemo } from "react";
import type { ComponentType } from "react";
import type { ThemePageDefinition } from "./types";
import SiteLayout from "@/theme/layouts/PublicLayout";

const DynamicPage = lazy(() => import("@/theme/DynamicPage"));

function Loading() {
  return (
    <div className="min-h-[60vh] flex items-center justify-center flex-1">
      <div className="text-on-surface-muted animate-pulse">加载中...</div>
    </div>
  );
}

// Cache lazy components so they are only created once per lazyComponent function
const lazyCache = new WeakMap<() => Promise<{ default: ComponentType }>, ComponentType>();

function getLazyComponent(loader: () => Promise<{ default: ComponentType }>): ComponentType {
  let comp = lazyCache.get(loader);
  if (!comp) {
    comp = lazy(loader);
    lazyCache.set(loader, comp);
  }
  return comp;
}

interface ThemePageWrapperProps {
  pageDef: ThemePageDefinition;
}

export default function ThemePageWrapper({ pageDef }: ThemePageWrapperProps) {
  const Component = useMemo(() => {
    if (pageDef.renderMode === "hardcoded") {
      if (pageDef.lazyComponent) {
        return getLazyComponent(pageDef.lazyComponent);
      }
      if (pageDef.component) {
        return pageDef.component;
      }
    }
    return DynamicPage;
  }, [pageDef]);

  return (
    <SiteLayout>
      <Suspense fallback={<Loading />}>
        <Component slug={pageDef.slug} />
      </Suspense>
    </SiteLayout>
  );
}
