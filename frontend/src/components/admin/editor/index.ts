// Types (light)
export type {
  EditorPreset,
  EditorFeatures,
  ToolbarConfig,
  BubbleMenuConfig,
  ToolbarRow,
  ToolbarItemDef,
  EditorPorts,
  MediaUploadPort,
  MediaPickerPort,
  MediaRef,
  GalleryPick,
  EmbedPick,
} from "./types";
export type { ModalControls, ModalState } from "./types-internal";
export { useModalState } from "./useModalState";

// Host ports (types only — inkless adapters live in editor-host/)
export { DEFAULT_EDITOR_IMAGE_MAX_BYTES } from "./ports/types";

// Extension building — heavy; prefer dynamic import of RichTextEditor / LangEditorMountInner
export { buildExtensions } from "./extension-groups";

// Presets
export { getPreset, fullPreset, standardPreset, minimalPreset } from "./presets";

// Toolbar
export { default as EditorToolbar, ToolbarButton, ToolbarDivider } from "./EditorToolbar";
export { getToolbarItem, TOOLBAR_ITEMS } from "./toolbar-registry";

// Menus
export { default as EditorBubbleMenu } from "./EditorBubbleMenu";
export { default as TableBubbleMenu } from "./TableBubbleMenu";
export { default as EditorFloatingMenu } from "./EditorFloatingMenu";
export { default as LinkEditPopover } from "./LinkEditPopover";
