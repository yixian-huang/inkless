import AuthorSocialLinks from "@/components/blog/AuthorSocialLinks";
import { useBranding } from "@/hooks/useBranding";
import { useContentMaxWidth, useIsReadingLayout, useIsThemeHomePath } from "@/plugins/hooks";
import type { FooterChromeProps } from "@/plugins/types";
import { useHeaderSettings } from "@/theme/layouts/chrome/useHeaderSettings";

export default function BlogFooter({ config }: FooterChromeProps) {
  const branding = useBranding();
  const maxWidth = useContentMaxWidth();
  const isReading = useIsReadingLayout();
  const isThemeHome = useIsThemeHomePath();
  const { showSocials } = useHeaderSettings();
  const copyright = config?.copyright ?? branding.footer.copyright;
  const style = config?.style ?? "minimal";
  // Socials live in the home hero; keep footer quiet on theme home.
  const showHomeSocials =
    !isThemeHome &&
    showSocials &&
    branding.author.socials.some((s) => s.url?.trim());

  if (style === "none") {
    return null;
  }

  return (
    <footer className="mt-auto border-t border-border bg-surface font-sans">
      <div className="mx-auto px-4 md:px-content py-10 w-full" style={{ maxWidth }}>
        <div className={isReading ? "text-center space-y-4" : "flex flex-col items-center gap-4 text-center"}>
          {showHomeSocials && <AuthorSocialLinks />}
          <p className="text-sm text-on-surface-muted">{copyright}</p>
        </div>
        {branding.footer.icp && (
          <p className="text-xs text-on-surface-muted mt-3 text-center">{branding.footer.icp}</p>
        )}
      </div>
    </footer>
  );
}
