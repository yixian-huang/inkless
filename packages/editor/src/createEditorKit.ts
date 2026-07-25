import type { Extension } from "@tiptap/core";
import type {
  BubbleMenuConfig,
  EditorFeatures,
  EditorPreset,
  ToolbarConfig,
} from "./types";
import type { EditorPorts } from "./ports/types";
import { getPreset } from "./presets";

export type EditorPresetName = "full" | "standard" | "minimal";

/** Full kit: extensions + chrome config for one preset + host ports. */
export type EditorKit = {
  name: EditorPresetName;
  preset: EditorPreset;
  extensions: Extension[];
  toolbar: ToolbarConfig | null;
  bubbleMenu: BubbleMenuConfig | null;
  floatingMenu: boolean;
  features: EditorFeatures;
};

/** Toolbar / bubble / floating only — no TipTap extension cost. */
export type EditorSurface = {
  name: EditorPresetName;
  toolbar: ToolbarConfig | null;
  bubbleMenu: BubbleMenuConfig | null;
  floatingMenu: boolean;
  features: EditorFeatures;
};

/**
 * Single entry for building a preset-bound editor kit.
 * Article shell and standalone RichTextEditor should both use this.
 */
export function createEditorKit(
  presetName: EditorPresetName = "full",
  ports: EditorPorts = {},
): EditorKit {
  const preset = getPreset(presetName);
  return {
    name: (preset.name as EditorPresetName) || presetName,
    preset,
    extensions: preset.extensions(ports),
    toolbar: preset.toolbar,
    bubbleMenu: preset.bubbleMenu,
    floatingMenu: preset.floatingMenu,
    features: preset.features,
  };
}

/** Chrome config for a preset without building TipTap extensions. */
export function getEditorSurface(presetName: EditorPresetName = "full"): EditorSurface {
  const preset = getPreset(presetName);
  return {
    name: (preset.name as EditorPresetName) || presetName,
    toolbar: preset.toolbar,
    bubbleMenu: preset.bubbleMenu,
    floatingMenu: preset.floatingMenu,
    features: preset.features,
  };
}
