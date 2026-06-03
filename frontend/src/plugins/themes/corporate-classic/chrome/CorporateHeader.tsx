import { Link } from "react-router-dom";
import { useBranding } from "@/hooks/useBranding";
import type { HeaderChromeProps } from "@/plugins/types";
import { BaseSiteHeader, useHeaderScroll } from "@/theme/layouts/chrome";

export default function CorporateHeader({ config }: HeaderChromeProps) {
  const branding = useBranding();
  const style = config?.style ?? "sticky";
  const showScrollEffect = style !== "static";
  const scrolled = useHeaderScroll(showScrollEffect);
  const logoSrc = (config?.logo ?? branding.logo.light?.trim()) || "/images/logo.png";
  const logoAlt = branding.siteName || "Site";
  const isSticky = style === "sticky" || style === "transparent";

  return (
    <BaseSiteHeader
      config={config}
      variant="corporate"
      scrolled={scrolled}
      sticky={isSticky}
      languagePlacement="top-bar"
      showMobileLanguagePanel
      headerClassName={`transition-all duration-300 ${
        showScrollEffect && scrolled ? "bg-white/95 backdrop-blur-sm shadow-md" : "bg-transparent"
      }`}
      brand={(
        <Link to="/">
          <img src={logoSrc} alt={logoAlt} className="h-10 w-auto" />
        </Link>
      )}
    />
  );
}
