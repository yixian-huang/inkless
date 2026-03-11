import { useGlobalConfig } from "@/contexts/GlobalConfigContext";
import { useThemePages } from "@/contexts/ThemePagesContext";
import type { FooterConfig } from "./types";

interface ThemedFooterProps {
  config?: FooterConfig;
}

export default function ThemedFooter({ config }: ThemedFooterProps) {
  const { config: globalConfig } = useGlobalConfig();
  const { footerNavItems } = useThemePages();
  const globalFooter = globalConfig.footer || {};

  // Config prop overrides global config; CMS uses branding.logo.url + footer.links
  const style = config?.style ?? "full";
  const logoSrc = config?.logo ?? globalConfig.branding?.logo?.url ?? "/images/logo.png";
  const logoAlt = globalConfig.branding?.companyName || "Blotting Consultancy";
  const address = config?.address ?? globalFooter.address;
  const phone = config?.phone ?? globalFooter.phone;
  // Use theme page nav items for footer links, fallback to global config
  const links = footerNavItems.length > 0
    ? footerNavItems.map((item) => ({ label: item.label, href: item.path }))
    : (globalFooter.links ?? []);
  const copyright = config?.copyright ?? globalFooter.copyright ?? "\u00A9 2024 Blotting Consultancy";

  if (style === "none") {
    return null;
  }

  if (style === "minimal") {
    return (
      <footer className="bg-primary text-on-primary">
        <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 py-6">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
            <img
              src={logoSrc}
              alt={logoAlt}
              className="h-8 w-auto"
            />
            <p className="text-sm text-gray-300">{copyright}</p>
          </div>
        </div>
      </footer>
    );
  }

  // "full" style (default)
  const sections = config?.sections ?? [];

  return (
    <footer className="bg-primary text-on-primary">
      <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 py-12">
        {sections.length > 0 ? (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-8 xl:gap-10">
            {/* Company Info */}
            <div>
              <img
                src={logoSrc}
                alt={logoAlt}
                className="h-10 w-auto mb-4"
              />
              <div className="space-y-2 text-sm text-on-primary/70">
                {address && <p>{address}</p>}
                {phone && <p>{phone}</p>}
              </div>
            </div>

            {/* Section Columns */}
            {sections.map((section, index) => (
              <div key={index}>
                {section.title && (
                  <h3 className="text-sm font-semibold uppercase tracking-wider mb-4 text-on-primary">
                    {section.title}
                  </h3>
                )}
                {section.links && section.links.length > 0 && (
                  <ul className="space-y-2">
                    {section.links.map((link, linkIndex) => (
                      <li key={linkIndex}>
                        <a
                          href={link.href || "#"}
                          className="text-sm text-on-primary/70 hover:text-accent transition-colors cursor-pointer"
                        >
                          {link.label}
                        </a>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className="flex flex-col md:flex-row md:items-start gap-8 xl:gap-10">
            {/* Company Info */}
            <div>
              <img
                src={logoSrc}
                alt={logoAlt}
                className="h-10 w-auto mb-4"
              />
              <div className="space-y-2 text-sm text-gray-300">
                {address && <p>{address}</p>}
                {phone && <p>{phone}</p>}
              </div>
            </div>

            {/* Links */}
            {links.length > 0 && (
              <div className="md:ml-auto">
                <ul className="flex flex-wrap gap-4 text-sm">
                  {links.map((link, index) => (
                    <li key={index}>
                      <a
                        href={link.href || "#"}
                        className="text-gray-300 hover:text-accent transition-colors cursor-pointer"
                      >
                        {link.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}

        {/* Copyright */}
        <div className="mt-12 pt-8 border-t border-white/20 text-center">
          <p className="text-sm text-gray-300">
            {copyright} |{" "}
            <a
              href="https://readdy.ai/?ref=logo"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-accent transition-colors cursor-pointer"
            >
              Website Builder
            </a>
          </p>
        </div>
      </div>
    </footer>
  );
}
