import { useBranding } from "@/hooks/useBranding";
import { useContentMaxWidth } from "@/plugins/hooks";
import type { FooterChromeProps } from "@/plugins/types";

export default function MinimalFooter({ config }: FooterChromeProps) {
  const branding = useBranding();
  const maxWidth = useContentMaxWidth();
  const copyright = config?.copyright ?? branding.footer.copyright;
  const style = config?.style ?? "minimal";

  if (style === "none") return null;

  return (
    <footer className="mt-auto border-t border-border">
      <div className="mx-auto px-4 md:px-content py-6 w-full" style={{ maxWidth }}>
        <p className="text-sm text-on-surface-muted text-center">{copyright}</p>
      </div>
    </footer>
  );
}
