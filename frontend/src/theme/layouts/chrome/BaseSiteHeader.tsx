import { useEffect, useState, type CSSProperties, type ReactNode } from "react";
import { useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useLocaleMode } from "@/hooks/useLocaleMode";
import type { HeaderConfig } from "@/theme/layouts/types";
import DesktopNavLinks from "./DesktopNavLinks";
import MobileNavPanel from "./MobileNavPanel";
import MobileMenuButton from "./MobileMenuButton";
import HeaderLanguageToggle from "./HeaderLanguageToggle";
import { useSiteNavigation } from "./useSiteNavigation";

export interface BaseSiteHeaderProps {
  config?: HeaderConfig;
  variant: "corporate" | "blog";
  brand: ReactNode;
  utilities?: ReactNode;
  containerClassName?: string;
  containerStyle?: CSSProperties;
  headerClassName?: string;
  navPaddingClassName?: string;
  languagePlacement?: "top-bar" | "inline" | "none";
  showMobileLanguagePanel?: boolean;
  scrolled?: boolean;
  sticky?: boolean;
}

export default function BaseSiteHeader({
  config,
  variant,
  brand,
  utilities,
  containerClassName = "max-w-layout mx-auto px-4 md:px-content xl:px-8 w-full",
  containerStyle,
  headerClassName = "",
  navPaddingClassName = "py-6",
  languagePlacement = "none",
  showMobileLanguagePanel = false,
  scrolled = true,
  sticky = true,
}: BaseSiteHeaderProps) {
  const { t, i18n } = useTranslation("common");
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const location = useLocation();
  const { isMono, currentLocale } = useLocaleMode();
  const navigation = useSiteNavigation(config?.navigation);
  const showLanguageToggle = config?.showLanguageToggle ?? true;

  useEffect(() => setIsMobileMenuOpen(false), [location.pathname]);

  useEffect(() => {
    if (isMono && i18n.language !== currentLocale) {
      i18n.changeLanguage(currentLocale);
    }
  }, [isMono, currentLocale, i18n]);

  const showLang = showLanguageToggle && !isMono;
  const positionClass = sticky ? "sticky top-0 left-0 right-0 z-50" : "relative z-50";

  return (
    <header className={`${positionClass} ${headerClassName}`.trim()}>
      {showLang && languagePlacement === "top-bar" && (
        <div className={`py-1.5 transition-colors duration-300 ${
          scrolled ? "bg-primary text-white" : "bg-transparent text-white/80"
        }`}>
          <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 flex justify-end">
            <HeaderLanguageToggle variant="corporate-bar" scrolled={scrolled} />
          </div>
        </div>
      )}

      <nav className={navPaddingClassName} aria-label={t("nav.main", { defaultValue: "Main" })}>
        <div className={containerClassName} style={containerStyle}>
          <div className="flex justify-between items-center gap-4">
            <div className="shrink-0">{brand}</div>

            <div className="flex items-center flex-1 justify-end gap-2">
              <DesktopNavLinks items={navigation} variant={variant} scrolled={scrolled} />
              {utilities}
              {showLang && languagePlacement === "inline" && (
                <HeaderLanguageToggle variant="inline" scrolled={scrolled} />
              )}
              <MobileMenuButton
                open={isMobileMenuOpen}
                onToggle={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
                ariaLabel={t("nav.menu", { defaultValue: "Menu" })}
                className={variant === "corporate"
                  ? (scrolled ? "text-gray-700" : "text-white")
                  : "text-on-surface"}
              />
            </div>
          </div>

          <MobileNavPanel
            items={navigation}
            open={isMobileMenuOpen}
            onNavigate={() => setIsMobileMenuOpen(false)}
            variant={variant}
          />

          {showLang && showMobileLanguagePanel && isMobileMenuOpen && (
            <div className="lg:hidden pt-3">
              <HeaderLanguageToggle
                variant="corporate-mobile"
                onAfterToggle={() => setIsMobileMenuOpen(false)}
              />
            </div>
          )}
        </div>
      </nav>
    </header>
  );
}
