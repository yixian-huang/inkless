import type { EditorPreset } from "./types";
import { buildExtensions } from "./extension-groups";

/** Full preset — all extensions, two-row toolbar, bubble menu, floating menu, all interactions */
export const fullPreset: EditorPreset = {
  name: "full",
  extensions: () =>
    buildExtensions(
      { slashCommands: true, blockHandles: true, blockToolbar: true, imagePaste: true, dragDrop: true },
      { formatting: true, media: true, mediaAdvanced: true, table: true, layout: true }
    ),
  toolbar: {
    rows: [
      [
        "bold", "italic", "underline", "strike", "divider",
        "superscript", "subscript", "divider",
        "fontSize", "lineHeight", "divider",
        "textColor", "highlight", "divider",
        "h1", "h2", "h3", "divider",
        "alignLeft", "alignCenter", "alignRight", "alignJustify",
      ],
      [
        "bulletList", "orderedList", "taskList", "divider",
        "blockquote", "codeBlock", "horizontalRule", "details", "divider",
        "table", "divider",
        "columns2", "columns3", "divider",
        "image", "gallery", "video", "audio", "embed", "divider",
        "link",
      ],
    ],
  },
  bubbleMenu: {
    items: ["bold", "italic", "underline", "strike", "code", "link", "highlight", "textColor"],
  },
  floatingMenu: true,
  features: {
    slashCommands: true,
    blockHandles: true,
    blockToolbar: true,
    imagePaste: true,
    dragDrop: true,
  },
};

/** Standard preset — basic formatting + images/links/code, single-row toolbar, bubble menu */
export const standardPreset: EditorPreset = {
  name: "standard",
  extensions: () =>
    buildExtensions(
      { slashCommands: true, blockHandles: false, blockToolbar: false, imagePaste: true, dragDrop: false },
      { formatting: true, media: true, mediaAdvanced: false, table: true, layout: false }
    ),
  toolbar: {
    rows: [
      [
        "bold", "italic", "underline", "strike", "divider",
        "h1", "h2", "h3", "divider",
        "bulletList", "orderedList", "divider",
        "blockquote", "codeBlock", "divider",
        "image", "link",
      ],
    ],
  },
  bubbleMenu: {
    items: ["bold", "italic", "underline", "strike", "code", "link", "highlight"],
  },
  floatingMenu: true,
  features: {
    slashCommands: true,
    blockHandles: false,
    blockToolbar: false,
    imagePaste: true,
    dragDrop: false,
  },
};

/** Minimal preset — basic text formatting only, no fixed toolbar (bubble menu only) */
export const minimalPreset: EditorPreset = {
  name: "minimal",
  extensions: () =>
    buildExtensions(
      { slashCommands: false, blockHandles: false, blockToolbar: false, imagePaste: false, dragDrop: false },
      { formatting: true, media: false, mediaAdvanced: false, table: false, layout: false, enhancements: false }
    ),
  toolbar: null,
  bubbleMenu: {
    items: ["bold", "italic", "underline", "strike", "link"],
  },
  floatingMenu: false,
  features: {
    slashCommands: false,
    blockHandles: false,
    blockToolbar: false,
    imagePaste: false,
    dragDrop: false,
  },
};

const PRESETS: Record<string, EditorPreset> = {
  full: fullPreset,
  standard: standardPreset,
  minimal: minimalPreset,
};

export function getPreset(name: string): EditorPreset {
  return PRESETS[name] || fullPreset;
}
