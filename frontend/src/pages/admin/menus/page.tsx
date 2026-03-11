import { useState, useEffect, useCallback, useRef } from "react";
import {
  listMenuGroups,
  createMenuGroup,
  updateMenuGroup,
  deleteMenuGroup,
  setMenuGroupPrimary,
  getMenuGroup,
  createMenuItem,
  updateMenuItem,
  deleteMenuItem,
  reorderMenuItems,
} from "@/api/menus";
import type { MenuGroup, MenuItem } from "@/api/menus";
import { getCategories, getTags, getAdminArticles } from "@/api/articles";
import type { Category, Tag, Article } from "@/api/articles";
import { listPages } from "@/api/pages";
import type { PageItem } from "@/api/pages";
import MetadataEditor from "@/components/admin/MetadataEditor";

// ── Tree utilities ──

interface TreeNode extends MenuItem {
  treeChildren: TreeNode[];
}

function buildTree(items: MenuItem[]): TreeNode[] {
  const map = new Map<number, TreeNode>();
  const roots: TreeNode[] = [];
  for (const item of items) {
    map.set(item.id, { ...item, treeChildren: [] });
  }
  for (const item of items) {
    const node = map.get(item.id)!;
    if (item.parentId && map.has(item.parentId)) {
      map.get(item.parentId)!.treeChildren.push(node);
    } else {
      roots.push(node);
    }
  }
  const sortLevel = (nodes: TreeNode[]) => {
    nodes.sort((a, b) => a.sortOrder - b.sortOrder);
    nodes.forEach((n) => sortLevel(n.treeChildren));
  };
  sortLevel(roots);
  return roots;
}

/** DFS flatten — returns IDs in tree-display order (used for reorder API) */
function flattenIds(nodes: TreeNode[]): number[] {
  const ids: number[] = [];
  for (const n of nodes) {
    ids.push(n.id);
    ids.push(...flattenIds(n.treeChildren));
  }
  return ids;
}

/** Build indented label list for parent selector dropdown */
function parentOptions(
  nodes: TreeNode[],
  excludeId: number | null,
  depth = 0,
): { id: number; label: string }[] {
  const result: { id: number; label: string }[] = [];
  for (const n of nodes) {
    if (n.id === excludeId) continue;
    const indent = "\u00A0\u00A0".repeat(depth);
    const prefix = depth > 0 ? `${indent}└ ` : "";
    result.push({ id: n.id, label: `${prefix}${n.zhName || n.enName}` });
    result.push(...parentOptions(n.treeChildren, excludeId, depth + 1));
  }
  return result;
}

// ── Type badge ──

const typeLabels: Record<string, string> = {
  custom_link: "链接",
  article: "文章",
  page: "页面",
  category: "分类",
  tag: "标签",
};

const typeColors: Record<string, string> = {
  custom_link: "bg-blue-50 text-blue-600",
  article: "bg-green-50 text-green-600",
  page: "bg-purple-50 text-purple-600",
  category: "bg-yellow-50 text-yellow-700",
  tag: "bg-pink-50 text-pink-600",
};

function TypeBadge({ type }: { type: string }) {
  return (
    <span className={`text-[11px] px-1.5 py-0.5 rounded font-medium ${typeColors[type] || "bg-gray-100 text-gray-600"}`}>
      {typeLabels[type] || type}
    </span>
  );
}

// ── Ref slug picker — searchable dropdown for category/tag/article/page ──

interface RefOption {
  slug: string;
  label: string; // display name (zhName or title)
}

function RefSlugPicker({
  type,
  value,
  onChange,
}: {
  type: MenuItem["type"];
  value: string;
  onChange: (slug: string) => void;
}) {
  const [options, setOptions] = useState<RefOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);

  // Fetch options when type changes
  useEffect(() => {
    if (type === "custom_link") return;
    let cancelled = false;
    setLoading(true);
    const fetch = async () => {
      try {
        let opts: RefOption[] = [];
        if (type === "category") {
          const cats = await getCategories();
          opts = cats.map((c: Category) => ({ slug: c.slug, label: c.zhName || c.enName || c.slug }));
        } else if (type === "tag") {
          const tags = await getTags();
          opts = tags.map((t: Tag) => ({ slug: t.slug, label: t.zhName || t.enName || t.slug }));
        } else if (type === "article") {
          const res = await getAdminArticles(1, 200);
          opts = (res.items || []).map((a: Article) => ({ slug: a.slug, label: a.zhTitle || a.enTitle || a.slug }));
        } else if (type === "page") {
          const pages = await listPages();
          opts = pages.map((p: PageItem) => ({ slug: p.slug, label: p.title?.zh || p.title?.en || p.slug }));
        }
        if (!cancelled) setOptions(opts);
      } catch {
        if (!cancelled) setOptions([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    fetch();
    return () => { cancelled = true; };
  }, [type]);

  // Close dropdown on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  // Filter options by query
  const q = query.toLowerCase();
  const filtered = q
    ? options.filter((o) => o.slug.toLowerCase().includes(q) || o.label.toLowerCase().includes(q))
    : options;

  // Find display label for current value
  const selectedLabel = options.find((o) => o.slug === value)?.label;

  return (
    <div ref={wrapperRef} className="relative">
      <label className="block text-xs font-medium text-gray-500 mb-1">
        关联{typeLabels[type] || "内容"}
      </label>
      <div
        className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm focus-within:ring-1 focus-within:ring-blue-400 focus-within:border-blue-400 flex items-center gap-1 cursor-pointer bg-white"
        onClick={() => setOpen(!open)}
      >
        {value ? (
          <span className="flex-1 truncate">
            <span className="text-gray-900">{selectedLabel || value}</span>
            <span className="text-gray-400 ml-1 text-xs">({value})</span>
          </span>
        ) : (
          <span className="flex-1 text-gray-400">
            {loading ? "加载中..." : "请选择..."}
          </span>
        )}
        <svg className="w-3.5 h-3.5 text-gray-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </div>

      {open && (
        <div className="absolute z-50 mt-1 w-full bg-white border border-gray-200 rounded-lg shadow-lg max-h-60 overflow-hidden">
          {/* Search input */}
          <div className="p-1.5 border-b border-gray-100">
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="w-full px-2 py-1.5 text-sm border border-gray-200 rounded focus:ring-1 focus:ring-blue-400 focus:border-blue-400 outline-none"
              placeholder="搜索名称或 slug..."
              autoFocus
              onClick={(e) => e.stopPropagation()}
            />
          </div>
          {/* Options list */}
          <div className="overflow-y-auto max-h-48">
            {loading ? (
              <div className="px-3 py-4 text-center text-xs text-gray-400">加载中...</div>
            ) : filtered.length === 0 ? (
              <div className="px-3 py-4 text-center text-xs text-gray-400">
                {q ? "无匹配结果" : "暂无数据"}
              </div>
            ) : (
              filtered.map((opt) => (
                <button
                  key={opt.slug}
                  type="button"
                  className={`w-full text-left px-3 py-2 text-sm hover:bg-blue-50 transition-colors flex items-center justify-between gap-2 cursor-pointer ${
                    opt.slug === value ? "bg-blue-50 text-blue-700" : "text-gray-700"
                  }`}
                  onClick={(e) => {
                    e.stopPropagation();
                    onChange(opt.slug);
                    setQuery("");
                    setOpen(false);
                  }}
                >
                  <span className="truncate">{opt.label}</span>
                  <span className="text-xs text-gray-400 shrink-0">{opt.slug}</span>
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ── Tree item row ──

function TreeItemRow({
  node,
  depth,
  siblings,
  siblingIndex,
  onEdit,
  onDelete,
  onAddChild,
  onSwap,
  onToggleVisible,
  editingItemId,
  renderForm,
}: {
  node: TreeNode;
  depth: number;
  siblings: TreeNode[];
  siblingIndex: number;
  onEdit: (item: MenuItem) => void;
  onDelete: (item: MenuItem) => void;
  onAddChild: (parentId: number) => void;
  onSwap: (siblings: TreeNode[], a: number, b: number) => void;
  onToggleVisible: (item: MenuItem) => void;
  editingItemId: number | null;
  renderForm: () => React.ReactNode;
}) {
  const [collapsed, setCollapsed] = useState(false);
  const hasChildren = node.treeChildren.length > 0;
  const isFirst = siblingIndex === 0;
  const isLast = siblingIndex === siblings.length - 1;

  if (editingItemId === node.id) {
    return (
      <div style={{ paddingLeft: depth * 28 }}>
        {renderForm()}
      </div>
    );
  }

  return (
    <>
      <div
        className="group flex items-center gap-2 py-1.5 rounded-md hover:bg-gray-50 transition-colors"
        style={{ paddingLeft: depth * 28 + 4 }}
      >
        {/* Collapse toggle / tree indicator */}
        <div className="w-5 h-5 flex items-center justify-center shrink-0">
          {hasChildren ? (
            <button
              onClick={() => setCollapsed(!collapsed)}
              className="text-gray-400 hover:text-gray-600 cursor-pointer"
            >
              <svg
                className={`w-3.5 h-3.5 transition-transform ${collapsed ? "" : "rotate-90"}`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          ) : (
            <span className="w-1.5 h-1.5 rounded-full bg-gray-300" />
          )}
        </div>

        {/* Reorder arrows */}
        <div className="flex flex-col shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={() => !isFirst && onSwap(siblings, siblingIndex, siblingIndex - 1)}
            disabled={isFirst}
            className="text-gray-400 hover:text-gray-600 disabled:opacity-20 leading-none cursor-pointer"
          >
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
            </svg>
          </button>
          <button
            onClick={() => !isLast && onSwap(siblings, siblingIndex, siblingIndex + 1)}
            disabled={isLast}
            className="text-gray-400 hover:text-gray-600 disabled:opacity-20 leading-none cursor-pointer"
          >
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          </button>
        </div>

        {/* Item info */}
        <div className={`flex-1 min-w-0 flex items-center gap-2 ${node.visible === false ? "opacity-50" : ""}`}>
          <span className="text-sm text-gray-900 font-medium truncate">{node.zhName}</span>
          {node.enName && (
            <span className="text-xs text-gray-400 truncate">{node.enName}</span>
          )}
          <TypeBadge type={node.type} />
          {node.visible === false && (
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-gray-100 text-gray-400 font-medium">隐藏</span>
          )}
          <span className="text-xs text-gray-300 truncate hidden sm:inline">
            {node.type === "custom_link" ? node.url : node.refSlug || ""}
          </span>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-0.5 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={() => onToggleVisible(node)}
            className={`px-1.5 py-0.5 text-[11px] rounded cursor-pointer ${
              node.visible === false
                ? "text-gray-400 hover:text-green-600 hover:bg-green-50"
                : "text-green-600 hover:text-gray-400 hover:bg-gray-50"
            }`}
            title={node.visible === false ? "显示" : "隐藏"}
          >
            {node.visible === false ? (
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.878 9.878L3 3m6.878 6.878L21 21" /></svg>
            ) : (
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" /></svg>
            )}
          </button>
          <button
            onClick={() => onAddChild(node.id)}
            className="px-1.5 py-0.5 text-[11px] text-gray-500 hover:text-blue-600 hover:bg-blue-50 rounded cursor-pointer"
            title="添加子菜单"
          >
            + 子级
          </button>
          <button
            onClick={() => onEdit(node)}
            className="px-1.5 py-0.5 text-[11px] text-blue-600 hover:bg-blue-50 rounded cursor-pointer"
          >
            编辑
          </button>
          <button
            onClick={() => onDelete(node)}
            className="px-1.5 py-0.5 text-[11px] text-red-500 hover:bg-red-50 rounded cursor-pointer"
          >
            删除
          </button>
        </div>
      </div>

      {/* Children */}
      {hasChildren && !collapsed && (
        <div className="relative">
          {/* Vertical connector line */}
          <div
            className="absolute top-0 bottom-2 border-l border-gray-200"
            style={{ left: depth * 28 + 14 }}
          />
          {node.treeChildren.map((child, ci) => (
            <TreeItemRow
              key={child.id}
              node={child}
              depth={depth + 1}
              siblings={node.treeChildren}
              siblingIndex={ci}
              onEdit={onEdit}
              onDelete={onDelete}
              onAddChild={onAddChild}
              onSwap={onSwap}
              onToggleVisible={onToggleVisible}
              editingItemId={editingItemId}
              renderForm={renderForm}
            />
          ))}
        </div>
      )}
    </>
  );
}

// ── Main component ──

export default function MenusPage() {
  // -- Group state --
  const [groups, setGroups] = useState<MenuGroup[]>([]);
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // -- Group form --
  const [showNewGroup, setShowNewGroup] = useState(false);
  const [newGroupName, setNewGroupName] = useState("");
  const [newGroupSlug, setNewGroupSlug] = useState("");
  const [editingGroupId, setEditingGroupId] = useState<number | null>(null);
  const [editGroupName, setEditGroupName] = useState("");
  const [editGroupSlug, setEditGroupSlug] = useState("");
  const [savingGroup, setSavingGroup] = useState(false);

  // -- Items state --
  const [items, setItems] = useState<MenuItem[]>([]);
  const [loadingItems, setLoadingItems] = useState(false);

  // Derived tree
  const tree = buildTree(items);

  // -- Item form --
  const [showNewItem, setShowNewItem] = useState(false);
  const [editingItemId, setEditingItemId] = useState<number | null>(null);
  const [itemZhName, setItemZhName] = useState("");
  const [itemEnName, setItemEnName] = useState("");
  const [itemType, setItemType] = useState<MenuItem["type"]>("custom_link");
  const [itemUrl, setItemUrl] = useState("");
  const [itemRefSlug, setItemRefSlug] = useState("");
  const [itemTarget, setItemTarget] = useState<MenuItem["target"]>("_self");
  const [itemParentId, setItemParentId] = useState<number | null>(null);
  const [itemSortOrder, setItemSortOrder] = useState(0);
  const [itemMetadata, setItemMetadata] = useState<Record<string, unknown>>({});
  const [savingItem, setSavingItem] = useState(false);

  const loadGroups = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listMenuGroups();
      setGroups(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load menu groups");
    } finally {
      setLoading(false);
    }
  }, []);

  const loadItems = useCallback(async (groupId: number) => {
    setLoadingItems(true);
    try {
      const group = await getMenuGroup(groupId);
      setItems(group.items || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load menu items");
    } finally {
      setLoadingItems(false);
    }
  }, []);

  useEffect(() => {
    loadGroups();
  }, [loadGroups]);

  useEffect(() => {
    if (selectedGroupId) {
      loadItems(selectedGroupId);
    } else {
      setItems([]);
    }
  }, [selectedGroupId, loadItems]);

  // -- Group handlers --
  const handleCreateGroup = async () => {
    if (!newGroupName.trim() || !newGroupSlug.trim()) {
      setError("请填写菜单组名称和 slug");
      return;
    }
    setSavingGroup(true);
    setError(null);
    try {
      await createMenuGroup({ name: newGroupName, slug: newGroupSlug });
      setShowNewGroup(false);
      setNewGroupName("");
      setNewGroupSlug("");
      await loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Create group failed");
    } finally {
      setSavingGroup(false);
    }
  };

  const handleUpdateGroup = async () => {
    if (!editingGroupId || !editGroupName.trim()) {
      setError("请填写菜单组名称");
      return;
    }
    setSavingGroup(true);
    setError(null);
    try {
      await updateMenuGroup(editingGroupId, { name: editGroupName, slug: editGroupSlug });
      setEditingGroupId(null);
      await loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Update group failed");
    } finally {
      setSavingGroup(false);
    }
  };

  const handleDeleteGroup = async (group: MenuGroup) => {
    if (!confirm(`删除菜单组「${group.name}」？此操作不可恢复。`)) return;
    setError(null);
    try {
      await deleteMenuGroup(group.id);
      if (selectedGroupId === group.id) {
        setSelectedGroupId(null);
        setItems([]);
      }
      await loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Delete group failed");
    }
  };

  const handleSetPrimary = async (group: MenuGroup) => {
    setError(null);
    try {
      await setMenuGroupPrimary(group.id);
      await loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Set primary failed");
    }
  };

  // -- Item handlers --
  const resetItemForm = () => {
    setItemZhName("");
    setItemEnName("");
    setItemType("custom_link");
    setItemUrl("");
    setItemRefSlug("");
    setItemTarget("_self");
    setItemParentId(null);
    setItemSortOrder(0);
    setItemMetadata({});
  };

  const startEditItem = (item: MenuItem) => {
    setEditingItemId(item.id);
    setShowNewItem(false);
    setItemZhName(item.zhName);
    setItemEnName(item.enName);
    setItemType(item.type);
    setItemUrl(item.url || "");
    setItemRefSlug(item.refSlug || "");
    setItemTarget(item.target);
    setItemParentId(item.parentId ?? null);
    setItemSortOrder(item.sortOrder);
    setItemMetadata(item.metadata || {});
  };

  const startAddChild = (parentId: number) => {
    setEditingItemId(null);
    setShowNewItem(true);
    resetItemForm();
    setItemParentId(parentId);
    // Auto-set sort order to end of siblings
    const siblings = items.filter((i) => i.parentId === parentId);
    setItemSortOrder(siblings.length > 0 ? Math.max(...siblings.map((s) => s.sortOrder)) + 1 : 0);
  };

  const handleCreateItem = async () => {
    if (!selectedGroupId || !itemZhName.trim()) {
      setError("请填写中文名称");
      return;
    }
    setSavingItem(true);
    setError(null);
    try {
      await createMenuItem(selectedGroupId, {
        zhName: itemZhName,
        enName: itemEnName,
        type: itemType,
        url: itemType === "custom_link" ? itemUrl : "",
        refSlug: itemType !== "custom_link" ? itemRefSlug : "",
        target: itemTarget,
        parentId: itemParentId,
        sortOrder: itemSortOrder,
        metadata: itemMetadata,
      });
      setShowNewItem(false);
      resetItemForm();
      await loadItems(selectedGroupId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Create item failed");
    } finally {
      setSavingItem(false);
    }
  };

  const handleUpdateItem = async () => {
    if (!selectedGroupId || !editingItemId) return;
    setSavingItem(true);
    setError(null);
    try {
      await updateMenuItem(selectedGroupId, editingItemId, {
        zhName: itemZhName,
        enName: itemEnName,
        type: itemType,
        url: itemType === "custom_link" ? itemUrl : "",
        refSlug: itemType !== "custom_link" ? itemRefSlug : "",
        target: itemTarget,
        parentId: itemParentId,
        sortOrder: itemSortOrder,
        metadata: itemMetadata,
      });
      setEditingItemId(null);
      resetItemForm();
      await loadItems(selectedGroupId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Update item failed");
    } finally {
      setSavingItem(false);
    }
  };

  const handleDeleteItem = async (item: MenuItem) => {
    if (!selectedGroupId || !confirm(`删除菜单项「${item.zhName}」？`)) return;
    setError(null);
    try {
      await deleteMenuItem(selectedGroupId, item.id);
      if (editingItemId === item.id) {
        setEditingItemId(null);
        resetItemForm();
      }
      await loadItems(selectedGroupId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Delete item failed");
    }
  };

  /** Toggle visibility of a menu item */
  const handleToggleVisible = async (item: MenuItem) => {
    if (!selectedGroupId) return;
    const newVisible = item.visible === false ? true : false;
    // Optimistic update
    setItems((prev) => prev.map((i) => i.id === item.id ? { ...i, visible: newVisible } : i));
    try {
      await updateMenuItem(selectedGroupId, item.id, { ...item, visible: newVisible });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Toggle visible failed");
      await loadItems(selectedGroupId);
    }
  };

  /** Swap two siblings in the tree, then persist via reorder API */
  const handleSwapSiblings = async (siblings: TreeNode[], idxA: number, idxB: number) => {
    if (!selectedGroupId) return;
    // Swap in the sibling array
    [siblings[idxA], siblings[idxB]] = [siblings[idxB], siblings[idxA]];
    // Flatten entire tree to get new global order
    const newOrder = flattenIds(tree);
    // Optimistic update
    const reordered = newOrder.map((id, i) => {
      const orig = items.find((it) => it.id === id)!;
      return { ...orig, sortOrder: i };
    });
    setItems(reordered);
    try {
      await reorderMenuItems(selectedGroupId, newOrder);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Reorder failed");
      await loadItems(selectedGroupId);
    }
  };

  // Parent options for dropdown
  const parentOpts = parentOptions(tree, editingItemId);

  // -- Item form --
  const renderItemForm = (mode: "new" | "edit") => (
    <div className="bg-white border border-blue-200 rounded-lg p-4 space-y-3 shadow-sm">
      <h4 className="text-sm font-semibold text-gray-800">
        {mode === "new" ? "新建菜单项" : "编辑菜单项"}
      </h4>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">中文名称 *</label>
          <input
            type="text"
            value={itemZhName}
            onChange={(e) => setItemZhName(e.target.value)}
            className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm focus:ring-1 focus:ring-blue-400 focus:border-blue-400 outline-none"
            placeholder="首页"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">English Name</label>
          <input
            type="text"
            value={itemEnName}
            onChange={(e) => setItemEnName(e.target.value)}
            className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm focus:ring-1 focus:ring-blue-400 focus:border-blue-400 outline-none"
            placeholder="Home"
          />
        </div>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">类型</label>
          <select
            value={itemType}
            onChange={(e) => setItemType(e.target.value as MenuItem["type"])}
            className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm focus:ring-1 focus:ring-blue-400 focus:border-blue-400 outline-none"
          >
            <option value="custom_link">自定义链接</option>
            <option value="article">文章</option>
            <option value="page">页面</option>
            <option value="category">分类</option>
            <option value="tag">标签</option>
          </select>
        </div>
        <div>
          {itemType === "custom_link" ? (
            <>
              <label className="block text-xs font-medium text-gray-500 mb-1">URL</label>
              <input
                type="text"
                value={itemUrl}
                onChange={(e) => setItemUrl(e.target.value)}
                className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm focus:ring-1 focus:ring-blue-400 focus:border-blue-400 outline-none"
                placeholder="/about"
              />
            </>
          ) : (
            <RefSlugPicker
              type={itemType}
              value={itemRefSlug}
              onChange={setItemRefSlug}
            />
          )}
        </div>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">打开方式</label>
          <select
            value={itemTarget}
            onChange={(e) => setItemTarget(e.target.value as MenuItem["target"])}
            className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm focus:ring-1 focus:ring-blue-400 focus:border-blue-400 outline-none"
          >
            <option value="_self">当前页 (_self)</option>
            <option value="_blank">新窗口 (_blank)</option>
          </select>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">父级菜单</label>
          <select
            value={itemParentId ?? ""}
            onChange={(e) => setItemParentId(e.target.value ? Number(e.target.value) : null)}
            className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm focus:ring-1 focus:ring-blue-400 focus:border-blue-400 outline-none"
          >
            <option value="">无（顶级菜单）</option>
            {parentOpts.map((o) => (
              <option key={o.id} value={o.id}>{o.label}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">排序值</label>
          <input
            type="number"
            value={itemSortOrder}
            onChange={(e) => setItemSortOrder(Number(e.target.value))}
            className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm focus:ring-1 focus:ring-blue-400 focus:border-blue-400 outline-none"
          />
        </div>
      </div>
      <details className="text-xs">
        <summary className="text-gray-400 cursor-pointer hover:text-gray-600">元数据（高级）</summary>
        <div className="mt-2">
          <MetadataEditor value={itemMetadata} onChange={setItemMetadata} />
        </div>
      </details>
      <div className="flex items-center gap-2 pt-1">
        <button
          onClick={mode === "new" ? handleCreateItem : handleUpdateItem}
          disabled={savingItem}
          className="px-4 py-1.5 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50 cursor-pointer"
        >
          {savingItem ? "保存中..." : "保存"}
        </button>
        <button
          onClick={() => {
            if (mode === "new") setShowNewItem(false);
            else setEditingItemId(null);
            resetItemForm();
          }}
          className="px-4 py-1.5 border border-gray-300 rounded-md text-sm hover:bg-gray-50 cursor-pointer"
        >
          取消
        </button>
      </div>
    </div>
  );

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">菜单管理</h1>
        <p className="text-sm text-gray-500 mt-1">管理导航菜单组和菜单项，支持多级嵌套</p>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm flex items-center justify-between">
          <span>{error}</span>
          <button onClick={() => setError(null)} className="text-red-400 hover:text-red-600 text-lg leading-none cursor-pointer">&times;</button>
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-400">加载中...</div>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* ── Left: Menu Groups ── */}
          <div className="lg:col-span-1">
            <div className="bg-white shadow-sm rounded-lg border border-gray-200">
              <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
                <h2 className="text-sm font-semibold text-gray-800">菜单组</h2>
                <button
                  onClick={() => { setShowNewGroup(!showNewGroup); setEditingGroupId(null); }}
                  className="text-xs text-blue-600 hover:text-blue-800 font-medium cursor-pointer"
                >
                  {showNewGroup ? "取消" : "+ 新建"}
                </button>
              </div>

              {showNewGroup && (
                <div className="px-4 py-3 border-b border-gray-100 space-y-2 bg-gray-50">
                  <input
                    type="text"
                    value={newGroupName}
                    onChange={(e) => setNewGroupName(e.target.value)}
                    className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm"
                    placeholder="菜单组名称"
                  />
                  <input
                    type="text"
                    value={newGroupSlug}
                    onChange={(e) => setNewGroupSlug(e.target.value)}
                    className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm"
                    placeholder="slug (如 main-nav)"
                  />
                  <button
                    onClick={handleCreateGroup}
                    disabled={savingGroup}
                    className="px-3 py-1.5 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700 disabled:opacity-50 cursor-pointer"
                  >
                    {savingGroup ? "创建中..." : "创建"}
                  </button>
                </div>
              )}

              <div className="divide-y divide-gray-50">
                {groups.length === 0 ? (
                  <div className="px-4 py-10 text-center text-sm text-gray-400">
                    暂无菜单组
                  </div>
                ) : (
                  groups.map((group) => (
                    <div key={group.id}>
                      {editingGroupId === group.id ? (
                        <div className="px-4 py-3 space-y-2 bg-gray-50">
                          <input
                            type="text"
                            value={editGroupName}
                            onChange={(e) => setEditGroupName(e.target.value)}
                            className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm"
                          />
                          <input
                            type="text"
                            value={editGroupSlug}
                            onChange={(e) => setEditGroupSlug(e.target.value)}
                            className="w-full px-2.5 py-1.5 border border-gray-300 rounded-md text-sm"
                          />
                          <div className="flex items-center gap-2">
                            <button
                              onClick={handleUpdateGroup}
                              disabled={savingGroup}
                              className="px-2.5 py-1 bg-blue-600 text-white rounded-md text-xs disabled:opacity-50 cursor-pointer"
                            >
                              保存
                            </button>
                            <button
                              onClick={() => setEditingGroupId(null)}
                              className="px-2.5 py-1 border border-gray-300 rounded-md text-xs cursor-pointer"
                            >
                              取消
                            </button>
                          </div>
                        </div>
                      ) : (
                        <div
                          className={`px-4 py-3 cursor-pointer hover:bg-gray-50 flex items-center justify-between transition-colors ${
                            selectedGroupId === group.id ? "bg-blue-50 border-l-[3px] border-l-blue-500" : ""
                          }`}
                          onClick={() => setSelectedGroupId(group.id)}
                        >
                          <div className="min-w-0">
                            <div className="text-sm font-medium text-gray-900 truncate flex items-center gap-2">
                              {group.name}
                              {group.isPrimary && (
                                <span className="text-[10px] bg-green-100 text-green-700 px-1.5 py-0.5 rounded-full font-medium">主菜单</span>
                              )}
                            </div>
                            <div className="text-xs text-gray-400 mt-0.5">{group.slug}</div>
                          </div>
                          <div className="flex items-center gap-0.5 shrink-0" onClick={(e) => e.stopPropagation()}>
                            {!group.isPrimary && (
                              <button
                                onClick={() => handleSetPrimary(group)}
                                className="text-[11px] text-green-600 hover:bg-green-50 px-1.5 py-0.5 rounded cursor-pointer"
                                title="设为主菜单"
                              >
                                设主
                              </button>
                            )}
                            <button
                              onClick={() => {
                                setEditingGroupId(group.id);
                                setEditGroupName(group.name);
                                setEditGroupSlug(group.slug);
                                setShowNewGroup(false);
                              }}
                              className="text-[11px] text-blue-600 hover:bg-blue-50 px-1.5 py-0.5 rounded cursor-pointer"
                            >
                              编辑
                            </button>
                            <button
                              onClick={() => handleDeleteGroup(group)}
                              className="text-[11px] text-red-500 hover:bg-red-50 px-1.5 py-0.5 rounded cursor-pointer"
                            >
                              删除
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>

          {/* ── Right: Menu Items (Tree) ── */}
          <div className="lg:col-span-3">
            {!selectedGroupId ? (
              <div className="bg-white shadow-sm rounded-lg border border-gray-200 flex items-center justify-center h-64">
                <p className="text-sm text-gray-400">请先在左侧选择一个菜单组</p>
              </div>
            ) : (
              <div className="bg-white shadow-sm rounded-lg border border-gray-200">
                <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
                  <h2 className="text-sm font-semibold text-gray-800">
                    菜单项 — {groups.find((g) => g.id === selectedGroupId)?.name}
                    <span className="ml-2 text-xs text-gray-400 font-normal">
                      {items.length} 项
                    </span>
                  </h2>
                  <button
                    onClick={() => {
                      setShowNewItem(!showNewItem);
                      setEditingItemId(null);
                      resetItemForm();
                    }}
                    className="text-xs text-blue-600 hover:text-blue-800 font-medium cursor-pointer"
                  >
                    {showNewItem ? "取消" : "+ 新建菜单项"}
                  </button>
                </div>

                <div className="p-4 space-y-1">
                  {showNewItem && (
                    <div className="mb-3">
                      {renderItemForm("new")}
                    </div>
                  )}

                  {loadingItems ? (
                    <div className="py-10 text-center text-sm text-gray-400">加载中...</div>
                  ) : items.length === 0 && !showNewItem ? (
                    <div className="py-10 text-center text-sm text-gray-400">
                      暂无菜单项，点击右上角「+ 新建菜单项」开始添加
                    </div>
                  ) : (
                    tree.map((node, idx) => (
                      <TreeItemRow
                        key={node.id}
                        node={node}
                        depth={0}
                        siblings={tree}
                        siblingIndex={idx}
                        onEdit={startEditItem}
                        onDelete={handleDeleteItem}
                        onAddChild={startAddChild}
                        onSwap={handleSwapSiblings}
                        onToggleVisible={handleToggleVisible}
                        editingItemId={editingItemId}
                        renderForm={() => renderItemForm("edit")}
                      />
                    ))
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
