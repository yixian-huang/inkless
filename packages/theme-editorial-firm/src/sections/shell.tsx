import type { ReactNode } from "react";

/**
 * Shared content shell for editorial-firm sections.
 * SectionRenderer already applies background/padding from settings —
 * this only constrains horizontal measure.
 */
export function EfShell({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={`max-w-layout mx-auto px-4 md:px-content ${className}`}>
      {children}
    </div>
  );
}
