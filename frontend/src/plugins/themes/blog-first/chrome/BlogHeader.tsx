import { useContentMaxWidth } from "@/plugins/hooks";
import type { HeaderChromeProps } from "@/plugins/types";
import {
  BaseSiteHeader,
  BrandMark,
  HeaderUtilities,
  useHeaderSettings,
} from "@/theme/layouts/chrome";

export default function BlogHeader({ config }: HeaderChromeProps) {
  const { brandMode } = useHeaderSettings();
  const maxWidth = useContentMaxWidth();

  return (
    <BaseSiteHeader
      config={config}
      variant="blog"
      languagePlacement="inline"
      headerClassName="bg-surface/90 backdrop-blur border-b border-border"
      navPaddingClassName="py-3"
      containerClassName="mx-auto px-4 md:px-content w-full"
      containerStyle={{ maxWidth }}
      brand={<BrandMark brandMode={brandMode} hideDefaultLogo />}
      utilities={<HeaderUtilities />}
    />
  );
}
