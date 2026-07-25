import type { Extension } from "@tiptap/core";
import type { EditorPorts } from "./ports/types";

export type { EditorPorts, MediaUploadPort, MediaRef } from "./ports/types";

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
  /** Build TipTap extensions; pass host ports so image paste etc. can talk to the app. */
  extensions: (ports?: EditorPorts) => Extension[];
  toolbar: ToolbarConfig | null;
  bubbleMenu: BubbleMenuConfig | null;
  floatingMenu: boolean;
  features: EditorFeatures;
}
