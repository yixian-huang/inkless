import { useContext, useMemo } from "react";
import { ThemeManagerContext, type ThemeManagerContextValue } from "./ThemeManagerContextDef";
import { sectionRegistry, sectionMetas as baseSectionMetas } from "@/theme/sections";
import { useBootstrap } from "@/contexts/BootstrapContext";
import type { ComponentType } from "react";
import type { SectionProps, SectionMeta } from "@/theme/types";

/** Access the ThemeManager context */
export function useThemeManager(): ThemeManagerContextValue {
  return useContext(ThemeManagerContext);
}

/** Merge base section registry with active theme's section overrides */
export function useSectionRegistry(): {
  registry: Record<string, ComponentType<SectionProps<any>>>;
  metas: SectionMeta[];
} {
  const { activeTheme } = useThemeManager();

  return useMemo(() => {
    const merged = { ...sectionRegistry };
    if (activeTheme?.sections) {
      Object.assign(merged, activeTheme.sections);
    }

    const mergedMetas = [...baseSectionMetas];
    if (activeTheme?.sectionMetas) {
      // Add theme-specific metas that aren't already in base
      const existingTypes = new Set(mergedMetas.map((m) => m.type));
      for (const meta of activeTheme.sectionMetas) {
        if (!existingTypes.has(meta.type)) {
          mergedMetas.push(meta);
        }
      }
    }

    return { registry: merged, metas: mergedMetas };
  }, [activeTheme]);
}

export function useThemeSettings(): Record<string, any> {
  const { data: bootstrapData } = useBootstrap();
  const { activeTheme } = useThemeManager();

  return useMemo(() => {
    const config = (bootstrapData?.activeTheme as any)?.config || {};

    const defaults: Record<string, any> = {};
    if (activeTheme?.settingSchema) {
      for (const group of activeTheme.settingSchema) {
        for (const field of group.fields) {
          const key = `${group.group}.${field.name}`;
          if (field.defaultValue !== undefined) {
            defaults[key] = field.defaultValue;
          }
        }
      }
    }

    return { ...defaults, ...config };
  }, [bootstrapData, activeTheme]);
}
