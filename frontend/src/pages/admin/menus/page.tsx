import { useState, useEffect, useCallback } from "react";
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
import {
  AdminErrorBanner,
  AdminLoading,
  AdminPageHeader,
  useAdminConfirm,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import TreeItemRow from "./MenuTree";
import { buildTree, flattenIds } from "./menuTreeUtils";
import type { TreeNode } from "./menuTreeUtils";
import MenuItemForm from "./MenuItemForm";

// ── Main component ──

export default function MenusPage() {
  useDocumentTitle("菜单管理");
  const { confirm, confirmDialog } = useAdminConfirm();
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
    const ok = await confirm({
      title: "删除菜单组",
      message: `删除菜单组「${group.name}」？此操作不可恢复。`,
      confirmLabel: "删除",
      danger: true,
    });
    if (!ok) return;
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
    if (!selectedGroupId) return;
    const ok = await confirm({
      title: "删除菜单项",
      message: `删除菜单项「${item.zhName}」？`,
      confirmLabel: "删除",
      danger: true,
    });
    if (!ok) return;
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

  // -- Item form shared props --
  const itemFormProps = {
    tree,
    editingItemId,
    itemZhName, setItemZhName,
    itemEnName, setItemEnName,
    itemType, setItemType,
    itemUrl, setItemUrl,
    itemRefSlug, setItemRefSlug,
    itemTarget, setItemTarget,
    itemParentId, setItemParentId,
    itemSortOrder, setItemSortOrder,
    itemMetadata, setItemMetadata,
    savingItem,
  };

  const renderItemForm = (mode: "new" | "edit") => (
    <MenuItemForm
      {...itemFormProps}
      mode={mode}
      onSave={mode === "new" ? handleCreateItem : handleUpdateItem}
      onCancel={() => {
        if (mode === "new") setShowNewItem(false);
        else setEditingItemId(null);
        resetItemForm();
      }}
    />
  );

  return (
    <div>
      {confirmDialog}
      <AdminPageHeader
        title="菜单管理"
        description="管理导航菜单组和菜单项，支持多级嵌套"
      />

      {error && <AdminErrorBanner message={error} onDismiss={() => setError(null)} />}

      {loading ? (
        <AdminLoading />
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
