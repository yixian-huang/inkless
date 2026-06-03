import { useBranding } from "@/hooks/useBranding";
import { useContentMaxWidth } from "@/plugins/hooks";
import type { FooterChromeProps } from "@/plugins/types";
import BrandMark from "@/theme/layouts/chrome/BrandMark";
import { useHeaderSettings } from "@/theme/layouts/chrome/useHeaderSettings";

export default function BlogFooter({ config }: FooterChromeProps) {
  const branding = useBranding();
  const { brandMode } = useHeaderSettings();
  const maxWidth = useContentMaxWidth();
  const copyright = config?.copyright ?? branding.footer.copyright;
  const style = config?.style ?? "minimal";

  if (style === "none") {
    return null;
  }

  const showLogo = brandMode === "logo" && Boolean(branding.logo.light?.trim());

  return (
    <footer className="mt-auto border-t border-border bg-surface">
      <div className="mx-auto px-4 md:px-content py-8 w-full" style={{ maxWidth }}>
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
          {showLogo ? (
            <BrandMark
              brandMode="logo"
              hideDefaultLogo
              logoClassName="h-6 w-auto opacity-80"
            />
          ) : (
            <span className="text-sm text-on-surface-muted">{branding.siteName}</span>
          )}
          <p className="text-sm text-on-surface-muted text-center sm:text-right">{copyright}</p>
        </div>
        {branding.footer.icp && (
          <p className="text-xs text-on-surface-muted mt-2 text-center">{branding.footer.icp}</p>
        )}
      </div>
    </footer>
  );
}
