import { Extension } from "@tiptap/core";
import { PluginKey } from "@tiptap/pm/state";
import Suggestion from "@tiptap/suggestion";
import type { Editor, Range } from "@tiptap/core";
import tippy, { type Instance as TippyInstance } from "tippy.js";
import type { MediaPickerPort } from "@/components/admin/editor/ports/types";

export interface SlashCommandItem {
  title: string;
  description: string;
  icon: string; // emoji or short text
  command: (props: { editor: Editor; range: Range }) => void;
  keywords?: string[];
}

function buildSlashItems(picker?: MediaPickerPort): SlashCommandItem[] {
  return [
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
    title: "提示框",
    description: "插入高亮提示段落",
    icon: "💡",
    keywords: ["tip", "callout", "note", "tishi", "tixing"],
    command: ({ editor, range }) => {
      editor
        .chain()
        .focus()
        .deleteRange(range)
        .insertContent(
          `<blockquote><p><strong>提示：</strong>在这里写需要读者注意的内容。</p></blockquote><p></p>`,
        )
        .run();
    },
  },
  {
    title: "FAQ",
    description: "问答条目",
    icon: "❓",
    keywords: ["faq", "qa", "wenda", "question"],
    command: ({ editor, range }) => {
      editor
        .chain()
        .focus()
        .deleteRange(range)
        .insertContent(
          `<h3>常见问题</h3><p><strong>Q：问题是什么？</strong><br>A：在这里写回答。</p><p></p>`,
        )
        .run();
    },
  },
  {
    title: "步骤列表",
    description: "1-2-3 操作步骤",
    icon: "①",
    keywords: ["steps", "how-to", "buzhou", "tutorial"],
    command: ({ editor, range }) => {
      editor
        .chain()
        .focus()
        .deleteRange(range)
        .insertContent(
          `<h3>步骤</h3><ol><li><p><strong>第一步</strong> — 描述操作</p></li><li><p><strong>第二步</strong> — 描述操作</p></li><li><p><strong>第三步</strong> — 描述操作</p></li></ol><p></p>`,
        )
        .run();
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
    title: "Mermaid 图表",
    description: "插入流程图 / 时序图",
    icon: "◆",
    keywords: ["mermaid", "diagram", "flowchart", "tubiao"],
    command: ({ editor, range }) => {
      editor
        .chain()
        .focus()
        .deleteRange(range)
        .setMermaid({ source: "graph TD\n  A[开始] --> B[结束]" })
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
      picker?.openImage((ref) => {
        if (editor.isDestroyed || !ref.url) return;
        editor.chain().focus().setImage({ src: ref.url, alt: ref.filename }).run();
      });
    },
  },
  {
    title: "视频",
    description: "从附件库选择视频",
    icon: "🎬",
    keywords: ["video", "shipin"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      picker?.openVideo((ref) => {
        if (editor.isDestroyed || !ref.url) return;
        (editor.commands as any).setVideo({ src: ref.url });
      });
    },
  },
  {
    title: "音频",
    description: "从附件库选择音频",
    icon: "🎵",
    keywords: ["audio", "music", "yinpin"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      picker?.openAudio((ref) => {
        if (editor.isDestroyed || !ref.url) return;
        (editor.commands as any).setAudio({ src: ref.url });
      });
    },
  },
  {
    title: "嵌入",
    description: "嵌入外部链接 (YouTube/网页)",
    icon: "⧉",
    keywords: ["embed", "iframe", "youtube", "qianru"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      picker?.openEmbed((pick) => {
        if (editor.isDestroyed) return;
        if (pick.type === "youtube") {
          editor.commands.setYoutubeVideo({ src: pick.url });
        } else {
          (editor.commands as any).setIframe({ src: pick.url });
        }
      });
    },
  },
  {
    title: "图片集",
    description: "多图网格展示",
    icon: "⊞",
    keywords: ["gallery", "tuji"],
    command: ({ editor, range }) => {
      editor.chain().focus().deleteRange(range).run();
      picker?.openGallery((pick) => {
        if (editor.isDestroyed) return;
        (editor.commands as any).setImageGallery({
          images: pick.images.map((i) => ({ src: i.url, alt: i.filename })),
          columns: pick.columns ?? Math.min(pick.images.length, 3),
        });
      });
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
}

function filterItems(items: SlashCommandItem[], query: string): SlashCommandItem[] {
  if (!query) return items;
  const lower = query.toLowerCase();
  return items.filter((item) => {
    return (
      item.title.toLowerCase().includes(lower) ||
      item.description.toLowerCase().includes(lower) ||
      item.keywords?.some((k) => k.includes(lower))
    );
  });
}

const slashPluginKey = new PluginKey("slashCommands");

export interface SlashCommandsOptions {
  picker?: MediaPickerPort;
}

export const SlashCommands = Extension.create<SlashCommandsOptions>({
  name: "slashCommands",

  addOptions() {
    return {
      picker: undefined,
    };
  },

  addProseMirrorPlugins() {
    const picker = this.options.picker;
    const slashItems = buildSlashItems(picker);

    return [
      Suggestion({
        editor: this.editor,
        char: "/",
        pluginKey: slashPluginKey,
        allowSpaces: false,
        startOfLine: false,
        items: ({ query }) => filterItems(slashItems, query),
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
