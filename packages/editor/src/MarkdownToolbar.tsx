import { useCallback, type ReactNode } from "react";

export interface MarkdownSelectionApi {
  getValue: () => string;
  setValue: (next: string, cursor?: { start: number; end: number }) => void;
  getSelection: () => { start: number; end: number };
  focus: () => void;
  /** Jump to 1-based line and scroll into view (outline nav). */
  gotoLine?: (line: number) => void;
  /** Open CodeMirror search panel when available. */
  openSearch?: () => void;
}

interface MarkdownToolbarProps {
  api: MarkdownSelectionApi | null;
}

type Action =
  | { kind: "wrap"; before: string; after: string; placeholder?: string }
  | { kind: "linePrefix"; prefix: string }
  | { kind: "insert"; text: string; selectOffset?: number; selectLength?: number }
  | { kind: "fence"; lang?: string; placeholder?: string };

const ACTIONS: { title: string; label: ReactNode; action: Action }[] = [
  { title: "一级标题", label: "H1", action: { kind: "linePrefix", prefix: "# " } },
  { title: "二级标题", label: "H2", action: { kind: "linePrefix", prefix: "## " } },
  { title: "三级标题", label: "H3", action: { kind: "linePrefix", prefix: "### " } },
  { title: "粗体", label: <strong>B</strong>, action: { kind: "wrap", before: "**", after: "**", placeholder: "粗体" } },
  { title: "斜体", label: <em>I</em>, action: { kind: "wrap", before: "*", after: "*", placeholder: "斜体" } },
  { title: "删除线", label: <s>S</s>, action: { kind: "wrap", before: "~~", after: "~~", placeholder: "删除线" } },
  { title: "行内代码", label: <span className="font-mono text-xs">&lt;/&gt;</span>, action: { kind: "wrap", before: "`", after: "`", placeholder: "code" } },
  { title: "链接", label: "链接", action: { kind: "wrap", before: "[", after: "](url)", placeholder: "链接文字" } },
  { title: "图片", label: "图片", action: { kind: "insert", text: "![描述](https://)", selectOffset: 2, selectLength: 2 } },
  { title: "无序列表", label: "• 列表", action: { kind: "linePrefix", prefix: "- " } },
  { title: "有序列表", label: "1.", action: { kind: "linePrefix", prefix: "1. " } },
  { title: "引用", label: "引用", action: { kind: "linePrefix", prefix: "> " } },
  { title: "分割线", label: "—", action: { kind: "insert", text: "\n\n---\n\n" } },
  {
    title: "代码块",
    label: "代码块",
    action: { kind: "fence", lang: "", placeholder: "code" },
  },
  {
    title: "表格",
    label: "表格",
    action: {
      kind: "insert",
      text: "\n\n| 列1 | 列2 | 列3 |\n| --- | --- | --- |\n|  |  |  |\n\n",
      selectOffset: 4,
      selectLength: 2,
    },
  },
  {
    title: "Mermaid 图表",
    label: "Mermaid",
    action: {
      kind: "fence",
      lang: "mermaid",
      placeholder: "graph TD\n  A[开始] --> B[结束]",
    },
  },
  {
    title: "任务列表",
    label: "☐",
    action: { kind: "linePrefix", prefix: "- [ ] " },
  },
];

function applyAction(api: MarkdownSelectionApi, action: Action) {
  const value = api.getValue();
  const { start, end } = api.getSelection();
  const selected = value.slice(start, end);

  if (action.kind === "wrap") {
    const inner = selected || action.placeholder || "";
    const next = value.slice(0, start) + action.before + inner + action.after + value.slice(end);
    const selStart = start + action.before.length;
    const selEnd = selStart + inner.length;
    api.setValue(next, { start: selStart, end: selEnd });
    return;
  }

  if (action.kind === "linePrefix") {
    // Apply prefix to each selected line (or current line if empty selection)
    const lineStart = value.lastIndexOf("\n", Math.max(0, start - 1)) + 1;
    const lineEndIdx = value.indexOf("\n", end);
    const lineEnd = lineEndIdx === -1 ? value.length : lineEndIdx;
    const block = value.slice(lineStart, lineEnd);
    const lines = block.split("\n");
    const prefixed = lines
      .map((line) => {
        if (line.startsWith(action.prefix)) return line;
        return action.prefix + line;
      })
      .join("\n");
    const next = value.slice(0, lineStart) + prefixed + value.slice(lineEnd);
    api.setValue(next, {
      start: lineStart,
      end: lineStart + prefixed.length,
    });
    return;
  }

  if (action.kind === "fence") {
    const body = selected || action.placeholder || "";
    const fence = `\`\`\`${action.lang || ""}\n${body}\n\`\`\``;
    const next = value.slice(0, start) + fence + value.slice(end);
    const bodyStart = start + 3 + (action.lang?.length || 0) + 1;
    api.setValue(next, { start: bodyStart, end: bodyStart + body.length });
    return;
  }

  if (action.kind === "insert") {
    const next = value.slice(0, start) + action.text + value.slice(end);
    if (action.selectOffset != null) {
      const s = start + action.selectOffset;
      const e = s + (action.selectLength ?? 0);
      api.setValue(next, { start: s, end: e });
    } else {
      const pos = start + action.text.length;
      api.setValue(next, { start: pos, end: pos });
    }
  }
}

function ToolbarBtn({
  title,
  onClick,
  children,
}: {
  title: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      title={title}
      onMouseDown={(e) => e.preventDefault()}
      onClick={onClick}
      className="px-2 py-1 text-xs text-gray-700 rounded hover:bg-gray-100 border border-transparent hover:border-gray-200 min-w-[1.75rem]"
    >
      {children}
    </button>
  );
}

export default function MarkdownToolbar({ api }: MarkdownToolbarProps) {
  const run = useCallback(
    (action: Action) => {
      if (!api) return;
      applyAction(api, action);
      api.focus();
    },
    [api],
  );

  return (
    <div className="flex flex-wrap items-center gap-0.5 px-2 py-1">
      {ACTIONS.map((item, i) => (
        <ToolbarBtn key={i} title={item.title} onClick={() => run(item.action)}>
          {item.label}
        </ToolbarBtn>
      ))}
      {!api && (
        <span className="text-xs text-gray-400 ml-2">加载编辑器…</span>
      )}
    </div>
  );
}
