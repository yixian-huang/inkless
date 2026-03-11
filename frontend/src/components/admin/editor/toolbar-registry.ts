import type { Editor } from "@tiptap/react";
import type { ModalControls } from "./types-internal";

export interface ToolbarItemRegistration {
  name: string;
  icon: string;
  title: string;
  isActive: (editor: Editor) => boolean;
  run: (editor: Editor, modals: ModalControls) => void;
  /** "button" | "dropdown" | "color" */
  type?: "button" | "dropdown" | "color";
  /** For dropdown type: returns options */
  getOptions?: (editor: Editor) => { label: string; value: string; active?: boolean }[];
  /** For dropdown type: handle option select */
  onSelect?: (editor: Editor, value: string) => void;
  /** For dropdown: dynamic label */
  getLabel?: (editor: Editor) => string;
  /** For color type */
  getColor?: (editor: Editor) => string | undefined;
  onColorChange?: (editor: Editor, color: string) => void;
  onColorReset?: (editor: Editor) => void;
}

const FONT_SIZES = [
  { label: "默认", value: "" },
  { label: "12px", value: "12px" },
  { label: "14px", value: "14px" },
  { label: "16px", value: "16px" },
  { label: "18px", value: "18px" },
  { label: "20px", value: "20px" },
  { label: "24px", value: "24px" },
  { label: "28px", value: "28px" },
  { label: "32px", value: "32px" },
  { label: "36px", value: "36px" },
];

const LINE_HEIGHTS = [
  { label: "默认", value: "" },
  { label: "1.0", value: "1" },
  { label: "1.25", value: "1.25" },
  { label: "1.5", value: "1.5" },
  { label: "1.75", value: "1.75" },
  { label: "2.0", value: "2" },
  { label: "2.5", value: "2.5" },
  { label: "3.0", value: "3" },
];

const TABLE_OPTIONS = [
  { label: "插入表格 (3x3)", value: "insert" },
  { label: "添加列 (右)", value: "addColumnAfter" },
  { label: "添加列 (左)", value: "addColumnBefore" },
  { label: "删除列", value: "deleteColumn" },
  { label: "添加行 (下)", value: "addRowAfter" },
  { label: "添加行 (上)", value: "addRowBefore" },
  { label: "删除行", value: "deleteRow" },
  { label: "删除表格", value: "deleteTable" },
  { label: "合并单元格", value: "mergeCells" },
  { label: "拆分单元格", value: "splitCell" },
];

function runTableCommand(editor: Editor, value: string) {
  const cmd = editor.chain().focus();
  switch (value) {
    case "insert": cmd.insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run(); break;
    case "addColumnAfter": cmd.addColumnAfter().run(); break;
    case "addColumnBefore": cmd.addColumnBefore().run(); break;
    case "deleteColumn": cmd.deleteColumn().run(); break;
    case "addRowAfter": cmd.addRowAfter().run(); break;
    case "addRowBefore": cmd.addRowBefore().run(); break;
    case "deleteRow": cmd.deleteRow().run(); break;
    case "deleteTable": cmd.deleteTable().run(); break;
    case "mergeCells": cmd.mergeCells().run(); break;
    case "splitCell": cmd.splitCell().run(); break;
  }
}

/** All registered toolbar items */
export const TOOLBAR_ITEMS: ToolbarItemRegistration[] = [
  // ── Text formatting ──
  { name: "bold", icon: "B", title: "粗体", isActive: (e) => e.isActive("bold"), run: (e) => e.chain().focus().toggleBold().run(), type: "button" },
  { name: "italic", icon: "I", title: "斜体", isActive: (e) => e.isActive("italic"), run: (e) => e.chain().focus().toggleItalic().run(), type: "button" },
  { name: "underline", icon: "U", title: "下划线", isActive: (e) => e.isActive("underline"), run: (e) => e.chain().focus().toggleUnderline().run(), type: "button" },
  { name: "strike", icon: "S", title: "删除线", isActive: (e) => e.isActive("strike"), run: (e) => e.chain().focus().toggleStrike().run(), type: "button" },
  { name: "superscript", icon: "X²", title: "上标", isActive: (e) => e.isActive("superscript"), run: (e) => e.chain().focus().toggleSuperscript().run(), type: "button" },
  { name: "subscript", icon: "X₂", title: "下标", isActive: (e) => e.isActive("subscript"), run: (e) => e.chain().focus().toggleSubscript().run(), type: "button" },
  {
    name: "fontSize", icon: "", title: "字号", type: "dropdown",
    isActive: () => false, run: () => {},
    getLabel: (e) => e.getAttributes("textStyle").fontSize || "字号",
    getOptions: (e) => {
      const current = e.getAttributes("textStyle").fontSize || "";
      return FONT_SIZES.map((s) => ({ label: s.label, value: s.value, active: current === s.value }));
    },
    onSelect: (e, v) => { if (v) (e.commands as any).setFontSize(v); else (e.commands as any).unsetFontSize(); },
  },
  {
    name: "lineHeight", icon: "", title: "行高", type: "dropdown",
    isActive: () => false, run: () => {},
    getLabel: (e) => e.getAttributes("paragraph").lineHeight || e.getAttributes("heading").lineHeight || "行高",
    getOptions: (e) => {
      const current = e.getAttributes("paragraph").lineHeight || e.getAttributes("heading").lineHeight || "";
      return LINE_HEIGHTS.map((h) => ({ label: h.label, value: h.value, active: current === h.value }));
    },
    onSelect: (e, v) => { if (v) (e.commands as any).setLineHeight(v); else (e.commands as any).unsetLineHeight(); },
  },
  {
    name: "textColor", icon: "A", title: "文字颜色", type: "color",
    isActive: () => false, run: () => {},
    getColor: (e) => e.getAttributes("textStyle").color,
    onColorChange: (e, c) => e.chain().focus().setColor(c).run(),
    onColorReset: (e) => e.chain().focus().unsetColor().run(),
  },
  {
    name: "highlight", icon: "H", title: "高亮颜色", type: "color",
    isActive: () => false, run: () => {},
    getColor: (e) => e.getAttributes("highlight").color,
    onColorChange: (e, c) => e.chain().focus().toggleHighlight({ color: c }).run(),
    onColorReset: (e) => e.chain().focus().unsetHighlight().run(),
  },
  // ── Headings ──
  { name: "h1", icon: "H1", title: "标题 1", isActive: (e) => e.isActive("heading", { level: 1 }), run: (e) => e.chain().focus().toggleHeading({ level: 1 }).run(), type: "button" },
  { name: "h2", icon: "H2", title: "标题 2", isActive: (e) => e.isActive("heading", { level: 2 }), run: (e) => e.chain().focus().toggleHeading({ level: 2 }).run(), type: "button" },
  { name: "h3", icon: "H3", title: "标题 3", isActive: (e) => e.isActive("heading", { level: 3 }), run: (e) => e.chain().focus().toggleHeading({ level: 3 }).run(), type: "button" },
  // ── Alignment ──
  { name: "alignLeft", icon: "☰L", title: "左对齐", isActive: (e) => e.isActive({ textAlign: "left" }), run: (e) => e.chain().focus().setTextAlign("left").run(), type: "button" },
  { name: "alignCenter", icon: "☰C", title: "居中", isActive: (e) => e.isActive({ textAlign: "center" }), run: (e) => e.chain().focus().setTextAlign("center").run(), type: "button" },
  { name: "alignRight", icon: "☰R", title: "右对齐", isActive: (e) => e.isActive({ textAlign: "right" }), run: (e) => e.chain().focus().setTextAlign("right").run(), type: "button" },
  { name: "alignJustify", icon: "☰J", title: "两端对齐", isActive: (e) => e.isActive({ textAlign: "justify" }), run: (e) => e.chain().focus().setTextAlign("justify").run(), type: "button" },
  // ── Lists ──
  { name: "bulletList", icon: "• 列表", title: "无序列表", isActive: (e) => e.isActive("bulletList"), run: (e) => e.chain().focus().toggleBulletList().run(), type: "button" },
  { name: "orderedList", icon: "1. 列表", title: "有序列表", isActive: (e) => e.isActive("orderedList"), run: (e) => e.chain().focus().toggleOrderedList().run(), type: "button" },
  { name: "taskList", icon: "☑ 任务", title: "任务列表", isActive: (e) => e.isActive("taskList"), run: (e) => e.chain().focus().toggleTaskList().run(), type: "button" },
  // ── Blocks ──
  { name: "blockquote", icon: "❝ 引用", title: "引用", isActive: (e) => e.isActive("blockquote"), run: (e) => e.chain().focus().toggleBlockquote().run(), type: "button" },
  { name: "codeBlock", icon: "</>", title: "代码块", isActive: (e) => e.isActive("codeBlock"), run: (e) => e.chain().focus().toggleCodeBlock().run(), type: "button" },
  { name: "horizontalRule", icon: "—", title: "分隔线", isActive: () => false, run: (e) => e.chain().focus().setHorizontalRule().run(), type: "button" },
  { name: "details", icon: "▶ 折叠", title: "折叠内容", isActive: (e) => e.isActive("details"), run: (e) => e.chain().focus().setDetails().run(), type: "button" },
  // ── Table ──
  {
    name: "table", icon: "", title: "表格操作", type: "dropdown",
    isActive: () => false, run: () => {},
    getLabel: () => "表格",
    getOptions: () => TABLE_OPTIONS,
    onSelect: runTableCommand,
  },
  // ── Layout ──
  { name: "columns2", icon: "2栏", title: "2 栏布局", isActive: () => false, run: (e) => (e.commands as any).setColumns(2), type: "button" },
  { name: "columns3", icon: "3栏", title: "3 栏布局", isActive: () => false, run: (e) => (e.commands as any).setColumns(3), type: "button" },
  // ── Media (trigger modals) ──
  { name: "image", icon: "图片", title: "插入图片", isActive: () => false, run: (_e, m) => m.openImagePicker(), type: "button" },
  { name: "gallery", icon: "图片集", title: "图片集", isActive: () => false, run: (_e, m) => m.openGalleryPicker(), type: "button" },
  { name: "video", icon: "视频", title: "插入视频", isActive: () => false, run: (_e, m) => m.openVideoPicker(), type: "button" },
  { name: "audio", icon: "音频", title: "插入音频", isActive: () => false, run: (_e, m) => m.openAudioPicker(), type: "button" },
  { name: "embed", icon: "嵌入", title: "嵌入外部内容", isActive: () => false, run: (_e, m) => m.openEmbedUrl(), type: "button" },
  // ── Link ──
  {
    name: "link", icon: "链接", title: "链接",
    isActive: (e) => e.isActive("link"),
    run: (e) => {
      const previousUrl = e.getAttributes("link").href;
      const url = window.prompt("URL", previousUrl);
      if (url === null) return;
      if (url === "") { e.chain().focus().extendMarkRange("link").unsetLink().run(); return; }
      e.chain().focus().extendMarkRange("link").setLink({ href: url }).run();
    },
    type: "button",
  },
];

const itemMap = new Map(TOOLBAR_ITEMS.map((item) => [item.name, item]));

export function getToolbarItem(name: string): ToolbarItemRegistration | undefined {
  return itemMap.get(name);
}
