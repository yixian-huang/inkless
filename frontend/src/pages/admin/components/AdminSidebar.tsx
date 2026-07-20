import { useEffect, useMemo, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { ChevronDown, PanelLeftClose, PanelLeftOpen, Search, X } from "lucide-react";
import { ProductLogo } from "@/components/product/ProductLogo";
import { useAuth } from "@/contexts/AuthContext";
import { BROWSER_STORAGE_KEYS } from "@/lib/browserStorage";
import { prefetchAdminRoute } from "@/pages/admin/adminRoutePrefetch";
import {
  filterNavGroups,
  isGroupActive,
  isNavItemActive,
  type AdminNavGroup,
  type AdminNavGroupId,
  type AdminNavItem,
} from "@/pages/admin/nav/adminNav";

function handlePrefetchPath(path: string) {
  prefetchAdminRoute(path);
}

interface AdminSidebarProps {
  collapsed: boolean;
  onToggle: () => void;
  mobileOpen: boolean;
  onMobileClose: () => void;
}

type CollapsedMap = Partial<Record<AdminNavGroupId, boolean>>;

function readCollapsedMap(): CollapsedMap {
  try {
    const raw = localStorage.getItem(BROWSER_STORAGE_KEYS.adminNavGroupCollapsed);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as CollapsedMap;
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

function writeCollapsedMap(map: CollapsedMap) {
  localStorage.setItem(BROWSER_STORAGE_KEYS.adminNavGroupCollapsed, JSON.stringify(map));
}

function NavLinkItem({
  item,
  collapsed,
  active,
  onNavigate,
}: {
  item: AdminNavItem;
  collapsed: boolean;
  active: boolean;
  onNavigate?: () => void;
}) {
  const Icon = item.icon;
  const location = useLocation();
  const showChildren =
    !collapsed &&
    item.children &&
    item.children.length > 0 &&
    (active || item.children.some((c) => location.pathname.startsWith(c.path)));

  return (
    <div>
      <Link
        to={item.path}
        onClick={onNavigate}
        onMouseEnter={() => handlePrefetchPath(item.path)}
        onFocus={() => handlePrefetchPath(item.path)}
        onTouchStart={() => handlePrefetchPath(item.path)}
        title={collapsed ? item.label : undefined}
        className={`group flex items-center rounded-lg transition-colors duration-150 ${
          collapsed ? "justify-center px-2 py-2.5" : "px-3 py-2"
        } ${
          active
            ? "bg-blue-500/15 text-blue-300 shadow-[inset_3px_0_0_0] shadow-blue-400"
            : "text-slate-300 hover:bg-white/5 hover:text-white"
        }`}
      >
        <Icon className={`h-5 w-5 shrink-0 ${active ? "text-blue-300" : "text-slate-400 group-hover:text-slate-200"}`} />
        {!collapsed && (
          <span className="ml-3 truncate text-sm font-medium">{item.label}</span>
        )}
      </Link>
      {showChildren && (
        <div className="ml-8 mt-0.5 space-y-0.5 border-l border-slate-700 pl-3">
          {item.children!.map((child) => {
            const childActive =
              location.pathname === child.path || location.pathname.startsWith(`${child.path}/`);
            return (
              <Link
                key={child.path}
                to={child.path}
                onClick={onNavigate}
                onMouseEnter={() => handlePrefetchPath(child.path)}
                onFocus={() => handlePrefetchPath(child.path)}
                onTouchStart={() => handlePrefetchPath(child.path)}
                className={`block rounded-md px-2 py-1.5 text-xs transition-colors ${
                  childActive
                    ? "text-blue-300 font-medium"
                    : "text-slate-400 hover:text-slate-200"
                }`}
              >
                {child.label}
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}

function SidebarContent({
  collapsed,
  onToggle,
  onMobileClose,
}: {
  collapsed: boolean;
  onToggle: () => void;
  onMobileClose?: () => void;
}) {
  const location = useLocation();
  const { hasPermission } = useAuth();
  const [query, setQuery] = useState("");
  const [collapsedMap, setCollapsedMap] = useState<CollapsedMap>(() => readCollapsedMap());

  const filteredGroups = useMemo(
    () => filterNavGroups(hasPermission, { query }),
    [hasPermission, query],
  );

  // Auto-expand group containing the active route
  useEffect(() => {
    setCollapsedMap((prev) => {
      let changed = false;
      const next = { ...prev };
      for (const group of filteredGroups) {
        if (isGroupActive(location.pathname, group) && next[group.id]) {
          next[group.id] = false;
          changed = true;
        }
      }
      if (changed) writeCollapsedMap(next);
      return changed ? next : prev;
    });
  }, [location.pathname, filteredGroups]);

  const isGroupCollapsed = (group: AdminNavGroup): boolean => {
    if (query) return false;
    if (collapsedMap[group.id] !== undefined) return Boolean(collapsedMap[group.id]);
    if (isGroupActive(location.pathname, group)) return false;
    return Boolean(group.defaultCollapsed);
  };

  const toggleGroup = (groupId: AdminNavGroupId) => {
    setCollapsedMap((prev) => {
      const currently = prev[groupId];
      const group = filteredGroups.find((g) => g.id === groupId);
      const defaultCollapsed = Boolean(group?.defaultCollapsed);
      const active = group ? isGroupActive(location.pathname, group) : false;
      const effective = currently !== undefined ? currently : active ? false : defaultCollapsed;
      const next = { ...prev, [groupId]: !effective };
      writeCollapsedMap(next);
      return next;
    });
  };

  const handleNavClick = () => {
    onMobileClose?.();
  };

  return (
    <div className="flex h-full flex-col bg-slate-950 text-white">
      <div className="flex h-14 shrink-0 items-center border-b border-white/10 px-4">
        <ProductLogo collapsed={collapsed} className={collapsed ? "mx-auto" : ""} />
      </div>

      {!collapsed && (
        <div className="shrink-0 px-3 pt-3">
          <label className="relative block">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="过滤菜单…"
              className="w-full rounded-lg border border-white/10 bg-white/5 py-2 pl-8 pr-8 text-xs text-slate-200 placeholder:text-slate-500 outline-none focus:border-blue-500/50 focus:ring-1 focus:ring-blue-500/30"
            />
            {query ? (
              <button
                type="button"
                onClick={() => setQuery("")}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300"
                aria-label="清除过滤"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            ) : null}
          </label>
        </div>
      )}

      <nav className="flex-1 overflow-y-auto px-2 py-3 scrollbar-thin">
        {filteredGroups.length === 0 ? (
          <p className="px-3 py-6 text-center text-xs text-slate-500">无匹配菜单</p>
        ) : (
          filteredGroups.map((group, groupIndex) => {
            const groupCollapsed = isGroupCollapsed(group);
            const showHeader = !collapsed && Boolean(group.label);

            return (
              <div key={group.id} className={groupIndex > 0 ? "mt-3" : ""}>
                {showHeader ? (
                  <button
                    type="button"
                    onClick={() => toggleGroup(group.id)}
                    className="mb-1 flex w-full items-center justify-between rounded-md px-3 py-1.5 text-left text-[11px] font-semibold uppercase tracking-wider text-slate-500 hover:text-slate-300"
                  >
                    <span>{group.label}</span>
                    <ChevronDown
                      className={`h-3.5 w-3.5 transition-transform ${groupCollapsed ? "-rotate-90" : ""}`}
                    />
                  </button>
                ) : null}

                {collapsed && groupIndex > 0 ? (
                  <div className="mx-2 mb-2 border-t border-white/10" />
                ) : null}

                {!groupCollapsed || collapsed ? (
                  <div className="space-y-0.5">
                    {group.items.map((item) => (
                      <NavLinkItem
                        key={item.path}
                        item={item}
                        collapsed={collapsed}
                        active={isNavItemActive(location.pathname, item)}
                        onNavigate={handleNavClick}
                      />
                    ))}
                  </div>
                ) : null}
              </div>
            );
          })
        )}
      </nav>

      <div className="shrink-0 border-t border-white/10 p-2">
        <button
          type="button"
          onClick={onToggle}
          className="flex w-full items-center justify-center gap-2 rounded-lg py-2.5 text-slate-400 transition-colors hover:bg-white/5 hover:text-white"
          title={collapsed ? "展开侧边栏" : "收起侧边栏"}
        >
          {collapsed ? (
            <PanelLeftOpen className="h-5 w-5" />
          ) : (
            <>
              <PanelLeftClose className="h-5 w-5" />
              <span className="text-xs font-medium">收起</span>
            </>
          )}
        </button>
      </div>
    </div>
  );
}

export default function AdminSidebar({
  collapsed,
  onToggle,
  mobileOpen,
  onMobileClose,
}: AdminSidebarProps) {
  return (
    <>
      <aside
        className={`fixed top-0 left-0 z-20 hidden h-screen transition-all duration-200 md:block ${
          collapsed ? "w-16" : "w-64"
        }`}
      >
        <SidebarContent collapsed={collapsed} onToggle={onToggle} />
      </aside>

      {mobileOpen && (
        <div className="fixed inset-0 z-40 md:hidden">
          <div className="absolute inset-0 bg-black/50 backdrop-blur-[1px]" onClick={onMobileClose} />
          <aside className="absolute top-0 left-0 h-full w-64 shadow-2xl">
            <SidebarContent collapsed={false} onToggle={onToggle} onMobileClose={onMobileClose} />
          </aside>
        </div>
      )}
    </>
  );
}
