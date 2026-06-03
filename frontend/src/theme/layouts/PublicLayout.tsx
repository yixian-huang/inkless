import type { ReactNode } from "react";
import Sidebar from "./Sidebar";
import type { LayoutConfig } from "./types";
import { CORPORATE_DEFAULT_LAYOUT } from "./defaults";
import { QAWidget } from "@/modules/qa";
import { useGlobalConfig } from "@/contexts/GlobalConfigContext";
import { useThemeManager } from "@/plugins/hooks";
import { getFallbackLayoutChrome } from "@/plugins/builtinThemes";
import RssHeadLink from "@/components/feature/RssHeadLink";

interface SiteLayoutProps {
  layout?: LayoutConfig;
  children: ReactNode;
}

/** Public site shell: active theme layoutChrome + defaultLayout, with sticky footer column. */
export default function SiteLayout({ layout: layoutProp, children }: SiteLayoutProps) {
  const { activeTheme } = useThemeManager();
  const { features } = useGlobalConfig();

  const layout = layoutProp ?? activeTheme?.defaultLayout ?? CORPORATE_DEFAULT_LAYOUT;
  const layoutType = layout.type ?? "default";
  const hasHeader = layoutType !== "blank";
  const mainClassName = "flex-1 min-w-0 flex flex-col";

  const fallbackChrome = getFallbackLayoutChrome();
  const HeaderComponent = activeTheme?.layoutChrome?.Header ?? fallbackChrome?.Header;
  const FooterComponent = activeTheme?.layoutChrome?.Footer ?? fallbackChrome?.Footer;

  const showRss = features?.blog?.rss === true;

  return (
    <div className="min-h-screen bg-surface flex flex-col">
      {showRss && <RssHeadLink />}
      {hasHeader && HeaderComponent && <HeaderComponent config={layout.header} />}

      {layoutType === "sidebar" ? (
        <div className={`${mainClassName} max-w-layout mx-auto px-4 md:px-content xl:px-8 py-8 w-full`}>
          <div className="flex flex-1 gap-8 min-h-0">
            {layout.sidebar?.position === "left" && (
              <Sidebar config={layout.sidebar} />
            )}
            <main className="flex-1 min-w-0">{children}</main>
            {layout.sidebar?.position !== "left" && (
              <Sidebar config={layout.sidebar} />
            )}
          </div>
        </div>
      ) : (
        <main className={mainClassName}>{children}</main>
      )}

      {layoutType !== "blank" && FooterComponent && <FooterComponent config={layout.footer} />}

      {(features as { qa?: { enabled?: boolean } })?.qa?.enabled && <QAWidget />}
    </div>
  );
}
