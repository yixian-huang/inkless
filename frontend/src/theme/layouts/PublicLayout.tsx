import type { ReactNode } from "react";
import ThemedHeader from "./ThemedHeader";
import ThemedFooter from "./ThemedFooter";
import Sidebar from "./Sidebar";
import type { LayoutConfig } from "./types";
import { QAWidget } from "@/modules/qa";
import { useGlobalConfig } from "@/contexts/GlobalConfigContext";

interface PublicLayoutProps {
  layout?: LayoutConfig;
  children: ReactNode;
}

export default function PublicLayout({ layout, children }: PublicLayoutProps) {
  const layoutType = layout?.type ?? "default";
  const { features } = useGlobalConfig();

  return (
    <div className="min-h-screen bg-surface">
      {/* Header -- unless layout is "blank" */}
      {layoutType !== "blank" && <ThemedHeader config={layout?.header} />}

      {/* Main content area -- varies by layout type */}
      {layoutType === "sidebar" ? (
        <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 py-8 flex gap-8">
          {layout?.sidebar?.position === "left" && (
            <Sidebar config={layout.sidebar} />
          )}
          <main className="flex-1 min-w-0">{children}</main>
          {layout?.sidebar?.position !== "left" && (
            <Sidebar config={layout?.sidebar} />
          )}
        </div>
      ) : (
        <main>{children}</main>
      )}

      {/* Footer -- unless layout is "blank" */}
      {layoutType !== "blank" && <ThemedFooter config={layout?.footer} />}

      {/* Floating Q&A Widget -- only when QA feature is enabled */}
      {features?.qa?.enabled && <QAWidget />}
    </div>
  );
}
