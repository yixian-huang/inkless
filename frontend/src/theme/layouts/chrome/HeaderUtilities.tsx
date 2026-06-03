import { useGlobalConfig } from "@/contexts/GlobalConfigContext";
import { useBranding } from "@/hooks/useBranding";
import { useHeaderSettings } from "./useHeaderSettings";

/** RSS + social links for blog-style headers (theme settingSchema + Site Config Header). */
export default function HeaderUtilities() {
  const { showRssLink, showSocials } = useHeaderSettings();
  const { features } = useGlobalConfig();
  const branding = useBranding();
  const rssEnabled = features?.blog?.rss === true;

  if (!showRssLink && !showSocials) return null;

  return (
    <div className="hidden lg:flex items-center gap-3 ml-4">
      {showRssLink && rssEnabled && (
        <a
          href="/feed.xml"
          className="text-xs text-on-surface-muted hover:text-primary transition-colors"
          aria-label="RSS feed"
        >
          RSS
        </a>
      )}
      {showSocials && branding.author.socials.map((s) => (
        <a
          key={`${s.kind}-${s.url}`}
          href={s.url}
          className="text-xs text-on-surface-muted hover:text-primary transition-colors"
          target="_blank"
          rel="noopener noreferrer"
        >
          {s.label || s.kind}
        </a>
      ))}
    </div>
  );
}
