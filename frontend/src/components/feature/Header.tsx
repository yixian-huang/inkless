import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useGlobalConfig } from '@/contexts/GlobalConfigContext';
import { useThemePages } from '@/contexts/ThemePagesContext';
import { resolveLocale } from '@/utils/locale';
import { useBranding } from '@/hooks/useBranding';
import { useLocaleMode } from '@/hooks/useLocaleMode';
import { isFeatureEnabled, routeFeatureMap } from '@/router/featureMap';
import type { SiteConfigFeatures } from '@/types/siteConfig';

interface NavItem {
  label?: string;
  href?: string;
}

export default function Header() {
  const { i18n } = useTranslation('common');
  const [isScrolled, setIsScrolled] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  const { config: globalConfig, features } = useGlobalConfig();
  const branding = useBranding();
  const { isMono, currentLocale } = useLocaleMode();
  const { headerNavItems, menuNavItems } = useThemePages();
  // Priority: primary menu > theme pages > global config
  const navigation: NavItem[] = menuNavItems.length > 0
    ? menuNavItems.map((item) => ({ label: item.label, href: item.path }))
    : headerNavItems.length > 0
      ? headerNavItems.map((item) => ({ label: item.label, href: item.path }))
      : (globalConfig.nav?.items || []);
  const publishedFeatures = features as unknown as SiteConfigFeatures;
  const visibleNav = navigation.filter((item) => {
    if (!item.href) return true;
    const key = routeFeatureMap[item.href];
    if (!key) return true;
    return isFeatureEnabled(publishedFeatures, key);
  });
  const logoSrc = branding.logo.light || '/images/logo.png';
  const logoAlt = branding.siteName || 'Site';

  useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 50);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  useEffect(() => {
    if (isMono && i18n.language !== currentLocale) {
      i18n.changeLanguage(currentLocale);
    }
  }, [isMono, currentLocale, i18n]);

  const toggleLanguage = () => {
    const newLang = resolveLocale(i18n.language) === 'zh' ? 'en' : 'zh';
    i18n.changeLanguage(newLang);
  };

  return (
    <header
      className={`fixed top-0 left-0 right-0 z-50 transition-all duration-300 ${
        isScrolled ? 'bg-white shadow-md' : 'bg-transparent'
      }`}
    >
      {/* Top Language Bar */}
      {!isMono && (
        <div className="bg-primary text-white py-2">
          <div className="max-w-layout mx-auto px-4 md:px-6 flex justify-end items-center">
            <button
              onClick={toggleLanguage}
              className="text-sm hover:opacity-80 transition-opacity whitespace-nowrap cursor-pointer"
            >
              {resolveLocale(i18n.language) === 'zh' ? 'English' : '中文'}
            </button>
          </div>
        </div>
      )}

      {/* Main Navigation */}
      <nav className="py-8">
        <div className="max-w-layout mx-auto px-4 md:px-6">
          <div className="flex justify-between items-center">
            {/* Logo */}
            <div className="flex items-center">
              <Link to="/">
                <img
                  src={logoSrc}
                  alt={logoAlt}
                  className="h-10 w-auto"
                />
              </Link>
            </div>

            {/* Desktop Navigation */}
            {visibleNav.length > 0 && (
              <div className="hidden lg:flex items-center space-x-8">
                {visibleNav.map((item, index) => (
                  <Link
                    key={item.href || item.label || String(index)}
                    to={item.href || '/'}
                    className={`text-sm font-medium transition-colors whitespace-nowrap cursor-pointer ${
                      isScrolled ? 'text-gray-700 hover:text-accent' : 'text-white hover:text-accent'
                    }`}
                  >
                    {item.label}
                  </Link>
                ))}
              </div>
            )}

            {/* Mobile Menu Button */}
            <button
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              className={`lg:hidden p-2 cursor-pointer ${
                isScrolled ? 'text-gray-700' : 'text-white'
              }`}
            >
              <div className="w-6 h-5 flex flex-col justify-between">
                <span className={`block h-0.5 w-full bg-current transition-all ${isMobileMenuOpen ? 'rotate-45 translate-y-2' : ''}`}></span>
                <span className={`block h-0.5 w-full bg-current transition-all ${isMobileMenuOpen ? 'opacity-0' : ''}`}></span>
                <span className={`block h-0.5 w-full bg-current transition-all ${isMobileMenuOpen ? '-rotate-45 -translate-y-2' : ''}`}></span>
              </div>
            </button>
          </div>

          {/* Mobile Menu */}
          {isMobileMenuOpen && visibleNav.length > 0 && (
            <div className="lg:hidden mt-4 pb-4 space-y-3">
              {visibleNav.map((item, index) => (
                <Link
                  key={item.href || item.label || String(index)}
                  to={item.href || '/'}
                  className="block text-sm font-medium text-gray-700 hover:text-accent transition-colors cursor-pointer"
                  onClick={() => setIsMobileMenuOpen(false)}
                >
                  {item.label}
                </Link>
              ))}
            </div>
          )}
        </div>
      </nav>
    </header>
  );
}
