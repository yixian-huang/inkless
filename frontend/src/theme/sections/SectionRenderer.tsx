import type { ReactNode } from "react";
import type { SectionData, SectionSettings } from "../types";
import { useSectionRegistry } from "@/plugins/hooks";

interface SectionWrapperProps {
  settings?: SectionSettings;
  children: ReactNode;
}

function SectionWrapper({ settings, children }: SectionWrapperProps) {
  if (settings?.hidden) return null;

  const bgClass =
    settings?.background === "primary"
      ? "bg-primary"
      : settings?.background === "surface-alt"
        ? "bg-surface-alt"
        : "bg-surface";

  const padClass =
    settings?.padding === "lg"
      ? "py-section-lg"
      : settings?.padding === "sm"
        ? "py-section-sm"
        : settings?.padding === "none"
          ? ""
          : "py-section";

  return (
    <section className={`${bgClass} ${padClass}`}>
      {children}
    </section>
  );
}

interface SectionRendererProps {
  section: SectionData;
}

export default function SectionRenderer({ section }: SectionRendererProps) {
  const { registry } = useSectionRegistry();
  const Component = registry[section.type];

  if (!Component) {
    if (import.meta.env.DEV) {
      return (
        <div className="p-4 bg-yellow-50 text-yellow-800 border border-yellow-200 rounded mx-4 my-2">
          Unknown section type: <code>{section.type}</code>
        </div>
      );
    }
    return null;
  }

  return (
    <SectionWrapper settings={section.settings}>
      <Component data={section.data} settings={section.settings} />
    </SectionWrapper>
  );
}
