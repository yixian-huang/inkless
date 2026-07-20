import { useState, useRef, useEffect } from "react";
import { ExternalLink, LogOut, Menu, UserRound } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { getNavTitle } from "@/pages/admin/nav/adminNav";

interface AdminTopbarProps {
  pathname: string;
  siteName: string;
  onOpenMobileMenu: () => void;
  onLogout: () => void;
}

function roleBadgeLabel(user: { isSuperAdmin?: boolean; role?: string } | null | undefined): string {
  if (user?.isSuperAdmin) return "超级管理员";
  if (user?.role === "admin") return "管理员";
  return "编辑";
}

export default function AdminTopbar({
  pathname,
  siteName,
  onOpenMobileMenu,
  onLogout,
}: AdminTopbarProps) {
  const { user } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const pageTitle = getNavTitle(pathname);

  useEffect(() => {
    if (!menuOpen) return;
    const onPointerDown = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", onPointerDown);
    return () => document.removeEventListener("mousedown", onPointerDown);
  }, [menuOpen]);

  const isSuper = Boolean(user?.isSuperAdmin);

  return (
    <header className="sticky top-0 z-10 h-14 border-b border-[#e4ddd2]/90 bg-[#fbfaf7]/85 backdrop-blur-xl supports-[backdrop-filter]:bg-[#fbfaf7]/72">
      <div className="flex h-full items-center justify-between gap-3 px-4 sm:px-6">
        <div className="flex min-w-0 items-center gap-3">
          <button
            type="button"
            onClick={onOpenMobileMenu}
            className="rounded-lg p-1.5 text-[#6b6560] transition hover:bg-[#f0ebe3] hover:text-[#1a1814] md:hidden"
            aria-label="打开菜单"
          >
            <Menu className="h-5 w-5" />
          </button>
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold tracking-[-0.015em] text-[#1a1814]">
              {pageTitle}
            </p>
            <p className="hidden truncate text-xs tracking-wide text-[#8a8378] sm:block">
              {siteName}
              <span className="mx-1.5 text-[#d4cbbf]">·</span>
              管理后台
            </p>
          </div>
        </div>

        <div className="relative flex items-center gap-2" ref={menuRef}>
          <a
            href="/"
            target="_blank"
            rel="noreferrer"
            className="hidden items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-medium tracking-wide text-[#5c564f] transition-colors hover:bg-[#f0ebe3] hover:text-[#1a1814] sm:inline-flex"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            前台
          </a>

          <button
            type="button"
            onClick={() => setMenuOpen((open) => !open)}
            className="inline-flex items-center gap-2 rounded-lg border border-[#e4ddd2] bg-[#fbfaf7] px-2 py-1.5 text-sm shadow-[0_1px_0_rgba(26,24,20,0.04)] transition hover:border-[#d4cbbf] hover:bg-[#f5f1ea]"
            aria-expanded={menuOpen}
            aria-haspopup="menu"
          >
            <span className="flex h-7 w-7 items-center justify-center rounded-full bg-[#1a1814] text-[#f4efe6]">
              <UserRound className="h-3.5 w-3.5" />
            </span>
            <span className="hidden max-w-[8rem] truncate text-[#3d3832] sm:inline">
              {user?.username || "管理员"}
            </span>
            <span
              className={`hidden rounded-full px-1.5 py-0.5 text-[10px] font-medium tracking-wide md:inline ${
                isSuper
                  ? "bg-[#faf0ee] text-[#9b3b2e] ring-1 ring-[#ebd4cf]"
                  : "bg-[#f0ebe3] text-[#5c564f] ring-1 ring-[#e4ddd2]"
              }`}
            >
              {roleBadgeLabel(user)}
            </span>
          </button>

          {menuOpen && (
            <div
              role="menu"
              className="absolute right-0 top-full z-30 mt-1.5 w-56 overflow-hidden rounded-xl border border-[#e4ddd2] bg-[#fbfaf7] py-1 shadow-[0_16px_40px_rgba(26,24,20,0.12)]"
            >
              <div className="border-b border-[#ece6dc] px-3.5 py-3">
                <p className="truncate text-sm font-semibold tracking-tight text-[#1a1814]">
                  {user?.username || "管理员"}
                </p>
                <p className="mt-0.5 text-xs text-[#8a8378]">{roleBadgeLabel(user)}</p>
              </div>
              <a
                href="/"
                target="_blank"
                rel="noreferrer"
                role="menuitem"
                className="flex items-center gap-2 px-3.5 py-2.5 text-sm text-[#3d3832] transition hover:bg-[#f5f1ea]"
                onClick={() => setMenuOpen(false)}
              >
                <ExternalLink className="h-4 w-4 text-[#8a8378]" />
                打开前台
              </a>
              <button
                type="button"
                role="menuitem"
                onClick={() => {
                  setMenuOpen(false);
                  onLogout();
                }}
                className="flex w-full items-center gap-2 px-3.5 py-2.5 text-sm text-[#9b3b2e] transition hover:bg-[#faf0ee]"
              >
                <LogOut className="h-4 w-4" />
                退出登录
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
