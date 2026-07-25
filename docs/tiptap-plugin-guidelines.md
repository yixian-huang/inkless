# TipTap / ProseMirror 自定义插件开发规范

本文档记录在开发 TipTap 自定义扩展时必须遵守的规则，源于一次生产环境编辑器卡死事故的复盘。

## 背景

ProseMirror 的视图更新流程（`prosemirror-view` v1.41.6）：

```
updateStateInner()
  → domObserver.stop()
  → 更新 DOM（docView.update）
  → selectionToDOM()
  → domObserver.start()          ← DOM 观察者重新激活
  → updatePluginViews(prev)      ← 此时才调用插件的 view.update()
```

`view.update()` 在 DOMObserver **已激活** 的状态下运行。插件对 `view.dom` 内部的任何 DOM 操作都会被 MutationObserver 捕获，可能触发 `readDOMChange` → 新 transaction → 再次调用 `view.update()` → 无限循环 → 页面卡死。

---

## 规则 1：不要在 view.update() 中直接修改编辑器 DOM

### 错误示范

```ts
// ❌ 直接在 view.update() 中操作 DOM
view() {
  return {
    update(view) {
      const dom = view.nodeDOM(pos) as HTMLElement;
      dom.classList.add("selected");           // 触发 MutationObserver
      dom.style.outline = "2px solid blue";    // 触发 MutationObserver
      dom.parentElement.appendChild(handle);   // 触发 MutationObserver
    },
  };
}
```

### 正确做法（按优先级）

**方案 A：使用 ProseMirror Decoration（推荐）**

Widget Decoration 是添加 UI 覆盖层的标准方式，PM 自己管理其生命周期，不会触发 DOMObserver 循环。

```ts
addProseMirrorPlugins() {
  return [
    new Plugin({
      props: {
        decorations(state) {
          const widgets = [];
          // 在目标位置添加 widget
          widgets.push(Decoration.widget(pos, () => {
            const handle = document.createElement("div");
            handle.className = "resize-handle";
            handle.contentEditable = "false";
            return handle;
          }));
          return DecorationSet.create(state.doc, widgets);
        },
      },
    }),
  ];
}
```

**方案 B：放在编辑器 DOM 外部**

将 overlay 容器放在 `view.dom` 的兄弟节点或更外层，用 `getBoundingClientRect()` 绝对定位。MutationObserver 只监听 `view.dom` 内部。

```ts
// ✅ 添加到 view.dom 的父级（不在观察范围内）
const editorParent = view.dom.parentElement;
editorParent.appendChild(handleContainer);
```

**方案 C：用 domObserver.stop()/start() 包裹**

如果必须修改编辑器内 DOM，临时关闭 DOMObserver，与 ProseMirror 内部使用的模式一致。

```ts
// ✅ 抑制 DOM 观察
function withoutDomObserver(view: EditorView, fn: () => void) {
  const obs = (view as any).domObserver;
  if (obs) {
    obs.stop();
    try { fn(); } finally { obs.start(); }
  } else {
    fn();
  }
}

// 使用
view() {
  return {
    update(view) {
      withoutDomObserver(view, () => {
        dom.classList.add("selected");
        container.appendChild(handle);
      });
    },
  };
}
```

### handleDOMEvents 同理

`mouseover`、`mouseleave` 等 DOM 事件处理器中修改编辑器 DOM 也需要保护：

```ts
props: {
  handleDOMEvents: {
    mouseover(view, event) {
      withoutDomObserver(view, () => {
        container.appendChild(hoverButton);
      });
      return false;
    },
  },
},
```

---

## 规则 2：不要在多个编辑器间共享扩展实例

TipTap 扩展是有状态的——初始化时绑定 `this.editor`、`this.storage`。共享实例会导致 `this.editor` 指向最后初始化的编辑器，插件闭包中捕获的 `editor` 引用错误。

```ts
// ❌ 两个编辑器共享同一组扩展
const extensions = useMemo(() => createEditorKit("full", ports).extensions, [ports]);
const editorA = useEditor({ extensions });
const editorB = useEditor({ extensions });

// ✅ 每个编辑器独立实例（各自 createEditorKit / 各自 ports）
const extA = useMemo(() => createEditorKit("full", portsA).extensions, [portsA]);
const extB = useMemo(() => createEditorKit("full", portsB).extensions, [portsB]);
const editorA = useEditor({ extensions: extA });
const editorB = useEditor({ extensions: extB });
```

---

## 规则 3：插件开发检查清单

编写或审查 TipTap 自定义扩展时，逐项检查：

- [ ] `addProseMirrorPlugins()` 中 `view.update()` 是否修改了 `view.dom` 内的 DOM？
- [ ] `handleDOMEvents` 处理器是否修改了编辑器内 DOM？
- [ ] 事件监听器回调中是否修改了编辑器 DOM？
- [ ] 所有编辑器内 DOM 操作是否用 `withoutDomObserver` 包裹？
- [ ] 添加到编辑器内的元素是否设置了 `contentEditable = "false"`？
- [ ] 多编辑器场景是否使用了独立的扩展实例？

---

## 涉及文件

本项目中应用了上述规则的文件：

- `packages/editor/src/extensions/resizable-media.ts` — 图片/视频缩放 handle + 替换按钮
- `packages/editor/src/extensions/block-toolbar.ts` — 块级节点选中高亮 + 工具栏
- `packages/editor/src/extensions/block-handle.ts` — 块级节点拖拽 handle（已在 `view.dom` 外部，无需 stop/start）
- `frontend/src/pages/admin/articles/editor/page.tsx` — 文章编辑器，双编辑器独立扩展实例（kit 来自 `@inkless/editor`）

---

## 事故回顾

**现象**：文章编辑器中插入图片或删除图片时页面完全卡死（无限循环）。

**根因**：`ResizableMedia` 和 `BlockToolbar` 在 `view.update()` 中直接操作编辑器 DOM（appendChild、classList、style），触发 MutationObserver → `readDOMChange` → 新 transaction → 再次 `view.update()` → 无限循环。

**修复**：用 `withoutDomObserver()` 包裹所有编辑器内 DOM 操作；双编辑器使用独立扩展实例。

**验证**：Playwright 自动化测试在生产环境确认图片插入（25ms）和删除（~200ms）均无卡顿。
