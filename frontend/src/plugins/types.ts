import type { ComponentType } from "react";
import type { ThemeTokens } from "@/theme/tokens";
import type { SectionProps, SectionMeta } from "@/theme/types";

// --- Plugin base types ---

export type PluginType = "theme" | "widget" | "integration";

export interface PluginManifest {
  id: string;
  name: string;
  nameZh: string;
  description: string;
  descriptionZh: string;
  author: string;
  version: string;
  type: PluginType;
  preview?: string;         // CSS gradient or image URL
  tags?: string[];
}

export interface Plugin {
  manifest: PluginManifest;
  onRegister?(): void;
  onActivate?(): void;
  onDeactivate?(): void;
}

// --- Theme plugin types ---

export type ThemePageRenderMode = "hardcoded" | "dynamic";

export interface ThemePageDefinition {
  slug: string;
  renderMode: ThemePageRenderMode;
  component?: ComponentType;
  lazyComponent?: () => Promise<{ default: ComponentType }>;
  contentKey?: string;
  nav: {
    label: string;
    labelZh: string;
    order: number;
    showInHeader?: boolean;
    showInFooter?: boolean;
  };
}

export interface LayoutConfig {
  type: string;
  header?: { style?: string };
  footer?: { style?: string };
}

export interface TokenPreset {
  id: string;
  name: string;
  nameZh: string;
  preview: string;
  tokens: ThemeTokens;
}

export interface ThemeSettingField {
  name: string;
  type: "text" | "textarea" | "number" | "boolean" | "select" | "color";
  label: string;
  labelZh: string;
  description?: string;
  defaultValue?: any;
  options?: { label: string; value: string }[];
}

export interface ThemeSettingGroup {
  group: string;
  label: string;
  labelZh: string;
  fields: ThemeSettingField[];
}

export interface ThemePlugin extends Plugin {
  manifest: PluginManifest & { type: "theme" };
  defaultTokens: ThemeTokens;
  tokenPresets?: TokenPreset[];
  pages: ThemePageDefinition[];
  sections?: Record<string, ComponentType<SectionProps<any>>>;
  sectionMetas?: SectionMeta[];
  settingSchema?: ThemeSettingGroup[];
  defaultLayout?: LayoutConfig;
}
