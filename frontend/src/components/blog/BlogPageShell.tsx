import type { ReactNode } from "react";
import { useLocation } from "react-router-dom";
import { useContentMaxWidth, useIsReadingLayout } from "@/plugins/hooks";

interface BlogPageShellProps {
  children: ReactNode;
  className?: string;
}

/** Content column width driven by active theme tokens. */
export default function BlogPageShell({ children, className = "" }: BlogPageShellProps) {
  const maxWidth = useContentMaxWidth();
  const isReading = useIsReadingLayout();
  const { pathname } = useLocation();
  const isBlogRoute = /^\/(blog|categories|tags)(\/|$)/.test(pathname);

  return (
    <div
      className={[
        // overflow-visible so floating article TOC can sit in the right gutter
        "mx-auto px-4 md:px-content flex-1 w-full overflow-visible",
        isReading || isBlogRoute ? "py-section font-sans" : "py-section-sm",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
      style={{ maxWidth }}
    >
      {children}
    </div>
  );
}
