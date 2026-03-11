import type { Extension } from "@tiptap/core";

export interface EditorFeatures {
  slashCommands: boolean;
  blockHandles: boolean;
  blockToolbar: boolean;
  imagePaste: boolean;
  dragDrop: boolean;
}

export type ToolbarItemDef = string;
export type ToolbarRow = (ToolbarItemDef | "divider")[];

export interface ToolbarConfig {
  rows: ToolbarRow[];
}

export interface BubbleMenuConfig {
  items: string[];
}

export interface EditorPreset {
  name: string;
  extensions: () => Extension[];
  toolbar: ToolbarConfig | null;
  bubbleMenu: BubbleMenuConfig | null;
  floatingMenu: boolean;
  features: EditorFeatures;
}
