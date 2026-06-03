import { useContentMaxWidth } from "@/plugins/hooks";
import type { HeaderChromeProps } from "@/plugins/types";
import { BaseSiteHeader, BrandMark } from "@/theme/layouts/chrome";

/** Minimal header for extension validation — uses shared BaseSiteHeader only. */
export default function MinimalHeader({ config }: HeaderChromeProps) {
  const maxWidth = useContentMaxWidth();

  return (
    <BaseSiteHeader
      config={config}
      variant="blog"
      languagePlacement="inline"
      headerClassName="border-b border-border bg-surface"
      navPaddingClassName="py-4"
      containerClassName="mx-auto px-4 md:px-content w-full"
      containerStyle={{ maxWidth }}
      brand={<BrandMark brandMode="text" />}
    />
  );
}
