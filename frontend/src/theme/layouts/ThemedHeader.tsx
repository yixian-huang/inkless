import { useState, useEffect, useRef, useCallback } from "react";
import { Link, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useGlobalConfig } from "@/contexts/GlobalConfigContext";
import { useThemePages } from "@/contexts/ThemePagesContext";
import { resolveLocale } from "@/utils/locale";
import type { HeaderConfig } from "./types";

interface NavItemData {
  label?: string;
  path?: string;
  children?: NavItemData[];
}

interface ThemedHeaderProps {
  config?: HeaderConfig;
}

// ── Desktop: recursive dropdown ──

function DesktopNavItem({ item, scrolled, depth = 0 }: {
  item: NavItemData;
  scrolled: boolean;
  depth?: number;
}) {
  const [open, setOpen] = useState(false);
  const [flipX, setFlipX] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const location = useLocation();

  const enter = useCallback(() => { clearTimeout(timer.current); setOpen(true); }, []);
  const leave = useCallback(() => { timer.current = setTimeout(() => setOpen(false), 120); }, []);

  useEffect(() => () => clearTimeout(timer.current), []);

  // Detect right-edge overflow when dropdown opens
  useEffect(() => {
    if (open && dropdownRef.current) {
      const rect = dropdownRef.current.getBoundingClientRect();
      if (rect.right > window.innerWidth - 8) {
        setFlipX(true);
      } else {
        setFlipX(false);
      }
    }
  }, [open]);

  const hasChildren = item.children && item.children.length > 0;
  const isActive = item.path ? location.pathname === item.path : false;
  const isRoot = depth === 0;

  // Root: adapts to scroll state. Sub-level: light text on translucent dropdown.
  const rootClass = `text-sm font-medium whitespace-nowrap cursor-pointer transition-colors duration-200 ${
    scrolled
      ? `text-gray-700 hover:text-blue-600 ${isActive ? "text-blue-600" : ""}`
      : `text-white/90 hover:text-white ${isActive ? "text-white" : ""}`
  }`;
  const subClass = `flex items-center justify-between gap-4 px-4 py-2.5 text-sm text-white/80 hover:text-white hover:bg-white/10 whitespace-nowrap cursor-pointer transition-colors ${
    isActive ? "text-white bg-white/10" : ""
  }`;

  const linkClass = isRoot ? rootClass : subClass;

  if (!hasChildren) {
    return (
      <Link to={item.path || "/"} className={linkClass}>
        {item.label}
      </Link>
    );
  }

  const chevron = (
    <svg
      className={`w-3 h-3 shrink-0 ${isRoot ? "ml-0.5" : ""} ${isRoot ? (scrolled ? "text-gray-400" : "text-white/60") : "text-white/40"}`}
      fill="none" stroke="currentColor" viewBox="0 0 24 24"
    >
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
        d={isRoot ? "M19 9l-7 7-7-7" : (flipX ? "M15 19l-7-7 7-7" : "M9 5l7 7-7 7")} />
    </svg>
  );

  // Position classes for dropdown
  const positionClass = isRoot
    ? (flipX ? "right-0 top-full pt-2" : "left-0 top-full pt-2")
    : (flipX ? "right-full top-0 pr-1" : "left-full top-0 pl-1");

  return (
    <div
      className="relative"
      onMouseEnter={enter}
      onMouseLeave={leave}
    >
      <Link to={item.path || "/"} className={`${linkClass} inline-flex items-center`}>
        {item.label}
        {chevron}
      </Link>

      {/* Dropdown panel */}
      <div
        ref={dropdownRef}
        className={`absolute z-50 transition-all duration-200 ${positionClass} ${open ? "opacity-100 visible translate-y-0" : "opacity-0 invisible -translate-y-1 pointer-events-none"}`}
      >
        <div className="bg-black/70 backdrop-blur-md rounded-lg shadow-xl ring-1 ring-white/10 py-1.5 min-w-[180px]">
          {item.children!.map((child, ci) => (
            <DesktopNavItem key={child.path || child.label || String(ci)} item={child} scrolled={scrolled} depth={depth + 1} />
          ))}
        </div>
      </div>
    </div>
  );
}

// ── Mobile: collapsible tree ──

function MobileNavItem({ item, depth = 0, onNavigate }: {
  item: NavItemData;
  depth?: number;
  onNavigate: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const location = useLocation();
  const hasChildren = item.children && item.children.length > 0;
  const isActive = item.path ? location.pathname === item.path : false;

  return (
    <div>
      <div
        className="flex items-center"
        style={{ paddingLeft: depth * 20 }}
      >
        <Link
          to={item.path || "/"}
          className={`flex-1 py-2.5 text-sm font-medium transition-colors cursor-pointer ${
            isActive
              ? "text-blue-600"
              : depth === 0
                ? "text-gray-800 hover:text-blue-600"
                : "text-gray-500 hover:text-blue-600"
          }`}
          onClick={onNavigate}
        >
          {item.label}
        </Link>
        {hasChildren && (
          <button
            onClick={() => setExpanded(!expanded)}
            className="p-2 text-gray-400 hover:text-gray-600 cursor-pointer"
          >
            <svg className={`w-4 h-4 transition-transform duration-200 ${expanded ? "rotate-90" : ""}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </button>
        )}
      </div>
      {hasChildren && expanded && (
        <div>
          {item.children!.map((child, ci) => (
            <MobileNavItem key={child.path || child.label || String(ci)} item={child} depth={depth + 1} onNavigate={onNavigate} />
          ))}
        </div>
      )}
    </div>
  );
}

// ── Header ──

export default function ThemedHeader({ config }: ThemedHeaderProps) {
  const { i18n } = useTranslation("common");
  const [isScrolled, setIsScrolled] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const location = useLocation();

  const { config: globalConfig } = useGlobalConfig();
  const { headerNavItems, menuNavItems } = useThemePages();

  // Close mobile menu on route change
  useEffect(() => setIsMobileMenuOpen(false), [location.pathname]);

  const globalNavItems = globalConfig.nav?.items || [];
  const navigation: NavItemData[] = config?.navigation
    ?? (menuNavItems.length > 0
      ? menuNavItems.map((item) => ({ label: item.label, path: item.path, children: item.children }))
      : headerNavItems.length > 0
        ? headerNavItems.map((item) => ({ label: item.label, path: item.path }))
        : globalNavItems.map((item) => ({ label: item.label, path: item.href })));

  const logoSrc = config?.logo ?? globalConfig.branding?.logo?.url ?? "/images/logo.png";
  const logoAlt = globalConfig.branding?.companyName || "Site Logo";
  const showLanguageToggle = config?.showLanguageToggle ?? true;
  const style = config?.style ?? "sticky";

  useEffect(() => {
    if (style === "static") return;
    const handleScroll = () => {
      const heroEl = document.querySelector("[data-page-hero]");
      if (heroEl) {
        const rect = heroEl.getBoundingClientRect();
        setIsScrolled(rect.bottom <= 80);
      } else {
        setIsScrolled(window.scrollY > 50);
      }
    };
    window.addEventListener("scroll", handleScroll, { passive: true });
    handleScroll();
    return () => window.removeEventListener("scroll", handleScroll);
  }, [style]);

  const toggleLanguage = () => {
    const newLang = resolveLocale(i18n.language) === "zh" ? "en" : "zh";
    i18n.changeLanguage(newLang);
  };

  const isFixed = style === "sticky" || style === "transparent";
  const showScrollEffect = style !== "static";
  const scrolled = showScrollEffect && isScrolled;

  return (
    <header
      className={`${isFixed ? "fixed top-0 left-0 right-0 z-50" : "relative z-50"} transition-all duration-300 ${
        scrolled ? "bg-white/95 backdrop-blur-sm shadow-md" : "bg-transparent"
      }`}
    >
      {/* Language bar */}
      {showLanguageToggle && (
        <div className={`py-1.5 transition-colors duration-300 ${
          scrolled ? "bg-primary text-white" : "bg-transparent text-white/80"
        }`}>
          <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 flex justify-end">
            <button
              onClick={toggleLanguage}
              className="text-xs hover:opacity-80 transition-opacity cursor-pointer"
            >
              {resolveLocale(i18n.language) === "zh" ? "English" : "\u4E2D\u6587"}
            </button>
          </div>
        </div>
      )}

      {/* Navigation */}
      <nav className="py-6">
        <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
          <div className="flex justify-between items-center">
            {/* Logo */}
            <Link to="/" className="shrink-0">
              <img src={logoSrc} alt={logoAlt} className="h-10 w-auto" />
            </Link>

            {/* Desktop nav */}
            {navigation.length > 0 && (
              <div className="hidden lg:flex items-center gap-7">
                {navigation.map((item, index) => (
                  <DesktopNavItem key={item.path || item.label || String(index)} item={item} scrolled={scrolled} />
                ))}
              </div>
            )}

            {/* Mobile hamburger */}
            <button
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              className={`lg:hidden p-2 cursor-pointer transition-colors ${
                scrolled ? "text-gray-700" : "text-white"
              }`}
              aria-label="Toggle menu"
            >
              <div className="w-5 h-4 flex flex-col justify-between">
                <span className={`block h-0.5 w-full bg-current transition-all duration-200 ${isMobileMenuOpen ? "rotate-45 translate-y-[7px]" : ""}`} />
                <span className={`block h-0.5 w-full bg-current transition-all duration-200 ${isMobileMenuOpen ? "opacity-0" : ""}`} />
                <span className={`block h-0.5 w-full bg-current transition-all duration-200 ${isMobileMenuOpen ? "-rotate-45 -translate-y-[7px]" : ""}`} />
              </div>
            </button>
          </div>

          {/* Mobile menu */}
          <div className={`lg:hidden overflow-hidden transition-all duration-300 ${
            isMobileMenuOpen ? "max-h-[80vh] opacity-100 mt-4" : "max-h-0 opacity-0"
          }`}>
            <div className="bg-white rounded-xl shadow-xl ring-1 ring-black/5 p-4 divide-y divide-gray-100">
              <div className="pb-2 space-y-0.5">
                {navigation.map((item, index) => (
                  <MobileNavItem
                    key={item.path || item.label || String(index)}
                    item={item}
                    onNavigate={() => setIsMobileMenuOpen(false)}
                  />
                ))}
              </div>
              {showLanguageToggle && (
                <div className="pt-3">
                  <button
                    onClick={() => { toggleLanguage(); setIsMobileMenuOpen(false); }}
                    className="text-sm text-gray-500 hover:text-blue-600 transition-colors cursor-pointer"
                  >
                    {resolveLocale(i18n.language) === "zh" ? "Switch to English" : "切换到中文"}
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </nav>
    </header>
  );
}
