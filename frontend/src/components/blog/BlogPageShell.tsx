import type { ReactNode } from "react";
import { useContentMaxWidth } from "@/plugins/hooks";

interface ThemeContentShellProps {
  children: ReactNode;
  className?: string;
}

/** Content column width driven by active theme tokens. */
export default function ThemeContentShell({ children, className = "" }: ThemeContentShellProps) {
  const maxWidth = useContentMaxWidth();

  return (
    <div
      className={`mx-auto px-4 md:px-content py-section-sm flex-1 w-full ${className}`.trim()}
      style={{ maxWidth }}
    >
      {children}
    </div>
  );
}
