import { useBranding } from "@/hooks/useBranding";
import { useContentMaxWidth, useIsReadingLayout, useIsThemeHomePath } from "@/plugins/hooks";
import type { HeaderChromeProps } from "@/plugins/types";
import {
  BaseSiteHeader,
  BrandMark,
  HeaderUtilities,
  useHeaderSettings,
} from "@/theme/layouts/chrome";
import { resolveBlogHomeBrandMode } from "./resolveHomeBrand";

export default function BlogHeader({ config }: HeaderChromeProps) {
  const { brandMode } = useHeaderSettings();
  const branding = useBranding();
  const maxWidth = useContentMaxWidth();
  const isReading = useIsReadingLayout();
  const isThemeHome = useIsThemeHomePath();
  const compactHome = isReading && isThemeHome;
  const resolvedBrandMode = resolveBlogHomeBrandMode(brandMode, branding, compactHome);

  return (
    <BaseSiteHeader
      config={config}
      variant="blog"
      languagePlacement="inline"
      headerClassName={
        compactHome
          ? "bg-surface/90 backdrop-blur-sm border-b border-border/70 font-sans"
          : "bg-surface border-b border-border font-sans"
      }
      navPaddingClassName={compactHome ? "py-3.5" : "py-4"}
      containerClassName="mx-auto px-4 md:px-content w-full"
      containerStyle={{ maxWidth }}
      brand={
        <BrandMark
          brandMode={resolvedBrandMode}
          hideDefaultLogo
          showLabel={!compactHome}
          textClassName="text-base font-sans font-medium tracking-tight text-on-surface"
          avatarClassName="h-9 w-9 rounded-full object-contain bg-[#141310] ring-1 ring-border"
          logoClassName="h-7 w-auto"
        />
      }
      utilities={<HeaderUtilities />}
    />
  );
}
