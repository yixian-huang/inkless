import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import Link from "@tiptap/extension-link";
import Underline from "@tiptap/extension-underline";
import Subscript from "@tiptap/extension-subscript";
import Superscript from "@tiptap/extension-superscript";
import TextAlign from "@tiptap/extension-text-align";
import { TextStyle } from "@tiptap/extension-text-style";
import Color from "@tiptap/extension-color";
import Highlight from "@tiptap/extension-highlight";
import { Table } from "@tiptap/extension-table";
import TableRow from "@tiptap/extension-table-row";
import TableHeader from "@tiptap/extension-table-header";
import TableCell from "@tiptap/extension-table-cell";
import TaskList from "@tiptap/extension-task-list";
import TaskItem from "@tiptap/extension-task-item";
import Youtube from "@tiptap/extension-youtube";
import CodeBlockLowlight from "@tiptap/extension-code-block-lowlight";
import Details from "@tiptap/extension-details";
import { DetailsContent, DetailsSummary } from "@tiptap/extension-details";
import Placeholder from "@tiptap/extension-placeholder";
import Typography from "@tiptap/extension-typography";
import CharacterCount from "@tiptap/extension-character-count";
import { common, createLowlight } from "lowlight";
import type { Extension } from "@tiptap/core";
import type { EditorFeatures } from "./types";
import {
  Iframe,
  Video,
  Audio,
  ImageGallery,
  Columns,
  Column,
  FontSize,
  LineHeight,
  ResizableMedia,
  SlashCommands,
  BlockHandle,
  BlockToolbar,
  ImagePaste,
} from "@/components/admin/tiptap-extensions";
import { uploadMedia } from "@/api/media";

const lowlight = createLowlight(common);

/** Core extensions required by all presets */
export function coreExtensions(): Extension[] {
  return [
    StarterKit.configure({
      codeBlock: false,
      dropcursor: { color: "#3b82f6", width: 2 },
    }) as Extension,
    Link.configure({ openOnClick: false, autolink: true }) as Extension,
    Placeholder.configure({
      placeholder: ({ node }) => {
        if (node.type.name === "heading") {
          return `标题 ${node.attrs.level}`;
        }
        return "输入内容，或输入 / 选择内容块...";
      },
    }) as Extension,
  ];
}

/** Text formatting: underline, sub/super, alignment, color, highlight, font size, line height */
export function formattingExtensions(): Extension[] {
  return [
    Underline as Extension,
    Subscript as Extension,
    Superscript as Extension,
    TextAlign.configure({ types: ["heading", "paragraph"] }) as Extension,
    TextStyle as Extension,
    Color as Extension,
    Highlight.configure({ multicolor: true }) as Extension,
    FontSize as Extension,
    LineHeight as Extension,
  ];
}

/** Basic media: image, code block */
export function mediaExtensions(): Extension[] {
  return [
    Image.configure({ inline: false, allowBase64: false }) as Extension,
    CodeBlockLowlight.configure({ lowlight }) as Extension,
  ];
}

/** Advanced media: video, audio, iframe, youtube, image gallery, resizable media */
export function mediaAdvancedExtensions(): Extension[] {
  return [
    Youtube.configure({ width: 640, height: 360 }) as Extension,
    Iframe as Extension,
    Video as Extension,
    Audio as Extension,
    ImageGallery as Extension,
    ResizableMedia as Extension,
  ];
}

/** Table extensions */
export function tableExtensions(): Extension[] {
  return [
    Table.configure({ resizable: true }) as Extension,
    TableRow as Extension,
    TableHeader as Extension,
    TableCell as Extension,
  ];
}

/** Layout: columns, details, task list */
export function layoutExtensions(): Extension[] {
  return [
    TaskList as Extension,
    TaskItem.configure({ nested: true }) as Extension,
    Details as Extension,
    DetailsContent as Extension,
    DetailsSummary as Extension,
    Columns as Extension,
    Column as Extension,
  ];
}

/** Enhancement: typography auto-replace + character count */
export function enhancementExtensions(): Extension[] {
  return [
    Typography as Extension,
    CharacterCount as Extension,
  ];
}

/** Interaction: slash commands, block handles, block toolbar, image paste */
export function interactionExtensions(features: EditorFeatures): Extension[] {
  const exts: Extension[] = [];
  if (features.slashCommands) exts.push(SlashCommands as Extension);
  if (features.blockHandles) exts.push(BlockHandle as Extension);
  if (features.blockToolbar) exts.push(BlockToolbar as Extension);
  if (features.imagePaste) {
    exts.push(
      ImagePaste.configure({
        uploadFn: async (file: File) => {
          const media = await uploadMedia(file);
          return { url: media.url, filename: media.filename };
        },
      }) as Extension
    );
  }
  return exts;
}

/** Build a complete extension array based on feature flags */
export function buildExtensions(
  features: EditorFeatures,
  groups: { formatting?: boolean; media?: boolean; mediaAdvanced?: boolean; table?: boolean; layout?: boolean; enhancements?: boolean } = {}
): Extension[] {
  const {
    formatting = true,
    media = true,
    mediaAdvanced = true,
    table = true,
    layout = true,
    enhancements = true,
  } = groups;

  return [
    ...coreExtensions(),
    ...(formatting ? formattingExtensions() : []),
    ...(media ? mediaExtensions() : []),
    ...(mediaAdvanced ? mediaAdvancedExtensions() : []),
    ...(table ? tableExtensions() : []),
    ...(layout ? layoutExtensions() : []),
    ...(enhancements ? enhancementExtensions() : []),
    ...interactionExtensions(features),
  ];
}
