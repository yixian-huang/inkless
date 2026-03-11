import { Extension } from "@tiptap/core";
import { PluginKey } from "@tiptap/pm/state";
import Suggestion from "@tiptap/suggestion";
import type { Editor, Range } from "@tiptap/core";
import tippy, { type Instance as TippyInstance } from "tippy.js";

export interface SlashCommandItem {
  title: string;
  description: string;
  icon: string; // emoji or short text
  command: (props: { editor: Editor; range: Range }) => void;
  keywords?: string[];
}

const SLASH_ITEMS: SlashCommandItem[] = [
  {
    title: "文本",
    description: "普通文本段落",
    icon: "Aa",
    keywords: ["text", "paragraph", "wenben"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).setParagraph().run();
    },
  },
  {
    title: "标题 1",
    description: "大标题",
    icon: "H1",
    keywords: ["h1", "heading1", "biaoti"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).setHeading({ level: 1 }).run();
    },
  },
  {
    title: "标题 2",
    description: "中标题",
    icon: "H2",
    keywords: ["h2", "heading2"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).setHeading({ level: 2 }).run();
    },
  },
  {
    title: "标题 3",
    description: "小标题",
    icon: "H3",
    keywords: ["h3", "heading3"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).setHeading({ level: 3 }).run();
    },
  },
  {
    title: "无序列表",
    description: "项目符号列表",
    icon: "•",
    keywords: ["ul", "bullet", "list", "liebiao"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).toggleBulletList().run();
    },
  },
  {
    title: "有序列表",
    description: "数字编号列表",
    icon: "1.",
    keywords: ["ol", "ordered", "number"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).toggleOrderedList().run();
    },
  },
  {
    title: "任务列表",
    description: "待办事项",
    icon: "☑",
    keywords: ["todo", "task", "checkbox", "renwu"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).toggleTaskList().run();
    },
  },
  {
    title: "引用",
    description: "引用文本块",
    icon: "❝",
    keywords: ["quote", "blockquote", "yinyong"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).setBlockquote().run();
    },
  },
  {
    title: "代码块",
    description: "代码片段",
    icon: "</>",
    keywords: ["code", "codeblock", "daima"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).setCodeBlock().run();
    },
  },
  {
    title: "分隔线",
    description: "水平分隔线",
    icon: "—",
    keywords: ["hr", "divider", "line", "fengexian"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).setHorizontalRule().run();
    },
  },
  {
    title: "折叠内容",
    description: "可展开/折叠区块",
    icon: "▶",
    keywords: ["details", "collapse", "toggle", "zhedie"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).setDetails().run();
    },
  },
  {
    title: "表格",
    description: "插入 3×3 表格",
    icon: "⊞",
    keywords: ["table", "biaoge"],
    command: ({ editor, range }) => {
      editor
        .chain()
        .focus()
        .deleteRange(range)
        .insertTable({ rows: 3, cols: 3, withHeaderRow: true })
        .run();
    },
  },
  {
    title: "图片",
    description: "从附件库选择图片",
    icon: "🖼",
    keywords: ["image", "picture", "photo", "tupian"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      document.dispatchEvent(
        new CustomEvent("slash-command-media", {
          detail: { type: "image" },
        })
      );
    },
  },
  {
    title: "视频",
    description: "从附件库选择视频",
    icon: "🎬",
    keywords: ["video", "shipin"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      document.dispatchEvent(
        new CustomEvent("slash-command-media", {
          detail: { type: "video" },
        })
      );
    },
  },
  {
    title: "音频",
    description: "从附件库选择音频",
    icon: "🎵",
    keywords: ["audio", "music", "yinpin"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      document.dispatchEvent(
        new CustomEvent("slash-command-media", {
          detail: { type: "audio" },
        })
      );
    },
  },
  {
    title: "嵌入",
    description: "嵌入外部链接 (YouTube/网页)",
    icon: "⧉",
    keywords: ["embed", "iframe", "youtube", "qianru"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      document.dispatchEvent(
        new CustomEvent("slash-command-media", {
          detail: { type: "embed" },
        })
      );
    },
  },
  {
    title: "图片集",
    description: "多图网格展示",
    icon: "⊞",
    keywords: ["gallery", "tuji"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      document.dispatchEvent(
        new CustomEvent("slash-command-media", {
          detail: { type: "gallery" },
        })
      );
    },
  },
  {
    title: "2 栏布局",
    description: "两列并排布局",
    icon: "⫼",
    keywords: ["columns", "2col", "layout", "fenlan"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      (editor.commands as any).setColumns(2);
    },
  },
  {
    title: "3 栏布局",
    description: "三列并排布局",
    icon: "⫼",
    keywords: ["columns", "3col"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      (editor.commands as any).setColumns(3);
    },
  },
];

function filterItems(query: string): SlashCommandItem[] {
  if (!query) return SLASH_ITEMS;
  const lower = query.toLowerCase();
  return SLASH_ITEMS.filter((item) => {
    return (
      item.title.toLowerCase().includes(lower) ||
      item.description.toLowerCase().includes(lower) ||
      item.keywords?.some((k) => k.includes(lower))
    );
  });
}

const slashPluginKey = new PluginKey("slashCommands");

export const SlashCommands = Extension.create({
  name: "slashCommands",

  addProseMirrorPlugins() {
    return [
      Suggestion({
        editor: this.editor,
        char: "/",
        pluginKey: slashPluginKey,
        allowSpaces: false,
        startOfLine: false,
        items: ({ query }) => filterItems(query),
        command: ({ editor, range, props }: any) => {
          props.command({ editor, range });
        },
        render: () => {
          let popup: TippyInstance | null = null;
          let menuEl: HTMLDivElement | null = null;
          let selectedIndex = 0;
          let currentItems: SlashCommandItem[] = [];

          const updateMenu = () => {
            if (!menuEl) return;
            menuEl.innerHTML = "";

            if (currentItems.length === 0) {
              const empty = document.createElement("div");
              empty.className = "slash-menu-empty";
              empty.textContent = "无匹配结果";
              menuEl.appendChild(empty);
              return;
            }

            currentItems.forEach((item, i) => {
              const row = document.createElement("button");
              row.className = `slash-menu-item${i === selectedIndex ? " slash-menu-item-active" : ""}`;
              row.type = "button";

              const icon = document.createElement("span");
              icon.className = "slash-menu-icon";
              icon.textContent = item.icon;

              const text = document.createElement("div");
              text.className = "slash-menu-text";

              const title = document.createElement("div");
              title.className = "slash-menu-title";
              title.textContent = item.title;

              const desc = document.createElement("div");
              desc.className = "slash-menu-desc";
              desc.textContent = item.description;

              text.appendChild(title);
              text.appendChild(desc);
              row.appendChild(icon);
              row.appendChild(text);

              row.addEventListener("mouseenter", () => {
                selectedIndex = i;
                updateMenu();
              });

              row.addEventListener("mousedown", (e) => {
                e.preventDefault();
              });

              row.addEventListener("click", (e) => {
                e.preventDefault();
                selectItem(i);
              });

              menuEl.appendChild(row);
            });

            // Scroll active item into view
            const activeEl = menuEl.querySelector(".slash-menu-item-active");
            if (activeEl) activeEl.scrollIntoView({ block: "nearest" });
          };

          let onSelectItem: ((index: number) => void) | null = null;
          const selectItem = (index: number) => {
            if (onSelectItem) onSelectItem(index);
          };

          return {
            onStart: (props: any) => {
              menuEl = document.createElement("div");
              menuEl.className = "slash-menu";

              currentItems = props.items;
              selectedIndex = 0;

              onSelectItem = (index: number) => {
                const item = currentItems[index];
                if (item) {
                  props.command(item);
                }
              };

              updateMenu();

              popup = tippy(document.body, {
                getReferenceClientRect: () =>
                  props.clientRect?.() || new DOMRect(),
                appendTo: () => document.body,
                content: menuEl,
                showOnCreate: true,
                interactive: true,
                trigger: "manual",
                placement: "bottom-start",
                maxWidth: 320,
                offset: [0, 8],
                popperOptions: {
                  modifiers: [
                    {
                      name: "flip",
                      options: { fallbackPlacements: ["top-start"] },
                    },
                  ],
                },
              });
            },

            onUpdate: (props: any) => {
              currentItems = props.items;
              selectedIndex = 0;

              onSelectItem = (index: number) => {
                const item = currentItems[index];
                if (item) props.command(item);
              };

              updateMenu();

              if (popup) {
                popup.setProps({
                  getReferenceClientRect: () =>
                    props.clientRect?.() || new DOMRect(),
                });
              }
            },

            onKeyDown: ({ event }: { event: KeyboardEvent }) => {
              if (currentItems.length === 0) return false;
              if (event.key === "ArrowUp") {
                selectedIndex =
                  (selectedIndex - 1 + currentItems.length) %
                  currentItems.length;
                updateMenu();
                return true;
              }
              if (event.key === "ArrowDown") {
                selectedIndex = (selectedIndex + 1) % currentItems.length;
                updateMenu();
                return true;
              }
              if (event.key === "Enter") {
                selectItem(selectedIndex);
                return true;
              }
              // Don't handle Escape — let @tiptap/suggestion handle it via onExit
              return false;
            },

            onExit: () => {
              popup?.destroy();
              popup = null;
              menuEl = null;
              onSelectItem = null;
            },
          };
        },
      }),
    ];
  },
});
