import { useEffect, useMemo, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { ChevronDown, PanelLeftClose, PanelLeftOpen, Search, X } from "lucide-react";
import { ProductLogo } from "@/components/product/ProductLogo";
import { useAuth } from "@/contexts/AuthContext";
import { BROWSER_STORAGE_KEYS } from "@/lib/browserStorage";
import { prefetchAdminRouteWithEditors } from "@/pages/admin/adminRoutePrefetch";
import {
  filterNavGroups,
  isGroupActive,
  isNavItemActive,
  type AdminNavGroup,
  type AdminNavGroupId,
  type AdminNavItem,
} from "@/pages/admin/nav/adminNav";
import { adminTheme } from "@/components/admin/ui/adminTheme";

function handlePrefetchPath(path: string) {
  prefetchAdminRouteWithEditors(path);
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
        className={`group flex items-center rounded-lg transition-all duration-150 ${
          collapsed ? "justify-center px-2 py-2.5" : "px-3 py-2"
        } ${
          active
            ? adminTheme.sidebarActive
            : `text-[#d6d0c6] hover:bg-[#f4efe6]/[0.07] hover:text-[#f7f3ec]`
        }`}
      >
        <Icon
          className={`h-[1.125rem] w-[1.125rem] shrink-0 ${
            active ? "text-[#171512]" : "text-[#9a9286] group-hover:text-[#e8e2d8]"
          }`}
          strokeWidth={active ? 2.25 : 1.75}
        />
        {!collapsed && (
          <span
            className={`ml-3 truncate text-[13px] ${
              active ? "font-semibold tracking-[-0.01em]" : "font-medium"
            }`}
          >
            {item.label}
          </span>
        )}
      </Link>
      {showChildren && (
        <div className="ml-8 mt-0.5 space-y-0.5 border-l border-[#3a342e] pl-3">
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
                    ? "font-semibold text-[#f4efe6]"
                    : "text-[#8a8378] hover:text-[#e8e2d8]"
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
    <div className="relative flex h-full flex-col overflow-hidden bg-[#171512] text-[#f7f3ec]">
      {/* Subtle paper grain / ink wash — no blue glow */}
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.35]"
        style={{
          backgroundImage:
            "radial-gradient(ellipse 80% 50% at 20% -10%, rgba(244,239,230,0.08), transparent 55%), radial-gradient(ellipse 60% 40% at 100% 100%, rgba(155,59,46,0.06), transparent 50%)",
        }}
        aria-hidden
      />
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.04]"
        style={{
          backgroundImage:
            "url(\"data:image/svg+xml,%3Csvg viewBox='0 0 200 200' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.85' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E\")",
        }}
        aria-hidden
      />

      <div className="relative flex h-14 shrink-0 items-center border-b border-[#2e2924] px-4">
        <ProductLogo
          collapsed={collapsed}
          className={collapsed ? "mx-auto" : ""}
          variant="ink"
        />
      </div>

      {!collapsed && (
        <div className="relative shrink-0 px-3 pt-3">
          <label className="relative block">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[#7a7368]" />
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="过滤菜单…"
              className={adminTheme.sidebarSearch}
            />
            {query ? (
              <button
                type="button"
                onClick={() => setQuery("")}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-[#7a7368] hover:text-[#d6d0c6]"
                aria-label="清除过滤"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            ) : null}
          </label>
        </div>
      )}

      <nav className="relative flex-1 overflow-y-auto px-2 py-3 scrollbar-thin">
        {filteredGroups.length === 0 ? (
          <p className="px-3 py-6 text-center text-xs text-[#7a7368]">无匹配菜单</p>
        ) : (
          filteredGroups.map((group, groupIndex) => {
            const groupCollapsed = isGroupCollapsed(group);
            const showHeader = !collapsed && Boolean(group.label);

            return (
              <div key={group.id} className={groupIndex > 0 ? "mt-4" : ""}>
                {showHeader ? (
                  <button
                    type="button"
                    onClick={() => toggleGroup(group.id)}
                    className="mb-1.5 flex w-full items-center justify-between rounded-md px-3 py-1 text-left text-[10px] font-semibold uppercase tracking-[0.14em] text-[#7a7368] transition hover:text-[#c4b8a4]"
                  >
                    <span>{group.label}</span>
                    <ChevronDown
                      className={`h-3.5 w-3.5 transition-transform ${groupCollapsed ? "-rotate-90" : ""}`}
                    />
                  </button>
                ) : null}

                {collapsed && groupIndex > 0 ? (
                  <div className="mx-2.5 mb-2 border-t border-[#2e2924]" />
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

      <div className="relative shrink-0 border-t border-[#2e2924] p-2">
        <button
          type="button"
          onClick={onToggle}
          className="flex w-full items-center justify-center gap-2 rounded-lg py-2.5 text-[#8a8378] transition-colors hover:bg-[#f4efe6]/[0.07] hover:text-[#f7f3ec]"
          title={collapsed ? "展开侧边栏" : "收起侧边栏"}
        >
          {collapsed ? (
            <PanelLeftOpen className="h-5 w-5" />
          ) : (
            <>
              <PanelLeftClose className="h-5 w-5" />
              <span className="text-xs font-medium tracking-wide">收起</span>
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
          <div
            className="absolute inset-0 bg-[#171512]/60 backdrop-blur-[2px]"
            onClick={onMobileClose}
          />
          <aside className="absolute top-0 left-0 h-full w-64 overflow-hidden shadow-2xl shadow-[#171512]/50">
            <SidebarContent collapsed={false} onToggle={onToggle} onMobileClose={onMobileClose} />
          </aside>
        </div>
      )}
    </>
  );
}
