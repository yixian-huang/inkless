import type { LucideIcon } from "lucide-react";
import {
  Activity,
  BarChart3,
  Bot,
  ClipboardList,
  Database,
  FileStack,
  FileText,
  HardDrive,
  Image,
  KeyRound,
  LayoutDashboard,
  Languages,
  Mail,
  Menu,
  MessageSquare,
  Palette,
  RefreshCw,
  ScrollText,
  Settings2,
  Shield,
  SlidersHorizontal,
  Sparkles,
  Timer,
  ToggleLeft,
  Users,
  Wand2,
} from "lucide-react";
import { commentModuleConfig } from "@/modules/comment";
import {
  getAdminRoutePermission,
  hasAdminRoutePermission,
  isAdminRouteVisibleInNavigation,
} from "@/router/adminAccess";

export type AdminNavGroupId =
  | "overview"
  | "content"
  | "appearance"
  | "insights"
  | "access"
  | "settings";

export type AdminNavChild = {
  path: string;
  label: string;
};

export type AdminNavItem = {
  id: string;
  path: string;
  label: string;
  group: AdminNavGroupId;
  icon: LucideIcon;
  exact?: boolean;
  description?: string;
  children?: AdminNavChild[];
  /** Settings hub subsection */
  settingsSection?: "integrations" | "ops" | "tools";
};

export type AdminNavGroup = {
  id: AdminNavGroupId;
  label: string;
  /** Collapsed by default when no item in group is active */
  defaultCollapsed?: boolean;
  items: AdminNavItem[];
};

export const adminNavGroups: AdminNavGroup[] = [
  {
    id: "overview",
    label: "概览",
    items: [
      {
        id: "dashboard",
        path: "/admin",
        label: "仪表盘",
        group: "overview",
        icon: LayoutDashboard,
        exact: true,
        description: "运营概览与快捷入口",
      },
    ],
  },
  {
    id: "content",
    label: "内容",
    items: [
      {
        id: "pages",
        path: "/admin/pages",
        label: "页面管理",
        group: "content",
        icon: FileStack,
        description: "可视化页面与区块",
      },
      {
        id: "articles",
        path: "/admin/articles",
        label: "文章管理",
        group: "content",
        icon: FileText,
        description: "博客文章与发布",
        children: [
          { path: "/admin/articles/categories", label: "分类" },
          { path: "/admin/articles/tags", label: "标签" },
        ],
      },
      {
        id: "media",
        path: "/admin/media",
        label: "媒体管理",
        group: "content",
        icon: Image,
        description: "图片与文件库",
      },
      {
        id: "menus",
        path: "/admin/menus",
        label: "菜单管理",
        group: "content",
        icon: Menu,
        description: "导航菜单结构",
      },
      {
        id: "form-submissions",
        path: "/admin/form-submissions",
        label: "表单提交",
        group: "content",
        icon: ClipboardList,
        description: "联系表单与线索",
      },
      {
        id: "comments",
        path: commentModuleConfig.sidebar.path,
        label: commentModuleConfig.sidebar.label,
        group: "content",
        icon: MessageSquare,
        description: "评论审核与回复",
      },
      {
        id: "scheduled-publications",
        path: "/admin/scheduled-publications",
        label: "定时发布",
        group: "content",
        icon: Timer,
        description: "预约发布队列",
      },
    ],
  },
  {
    id: "appearance",
    label: "外观",
    items: [
      {
        id: "theme",
        path: "/admin/theme",
        label: "主题",
        group: "appearance",
        icon: Palette,
        description: "主题包与视觉样式",
      },
      {
        id: "site-config",
        path: "/admin/site-config",
        label: "站点配置",
        group: "appearance",
        icon: Settings2,
        description: "品牌、SEO 与页眉页脚",
      },
      {
        id: "features",
        path: "/admin/features",
        label: "功能开关",
        group: "appearance",
        icon: ToggleLeft,
        description: "公开页面与模块开关",
      },
    ],
  },
  {
    id: "insights",
    label: "洞察",
    items: [
      {
        id: "analytics",
        path: "/admin/analytics",
        label: "访问统计",
        group: "insights",
        icon: BarChart3,
        description: "流量与访问趋势",
      },
    ],
  },
  {
    id: "access",
    label: "权限",
    items: [
      {
        id: "users",
        path: "/admin/users",
        label: "用户管理",
        group: "access",
        icon: Users,
        description: "后台账号",
      },
      {
        id: "roles",
        path: "/admin/roles",
        label: "角色管理",
        group: "access",
        icon: Shield,
        description: "角色与权限集",
      },
    ],
  },
  {
    id: "settings",
    label: "设置",
    defaultCollapsed: true,
    items: [
      {
        id: "settings-hub",
        path: "/admin/settings",
        label: "设置中心",
        group: "settings",
        icon: SlidersHorizontal,
        exact: true,
        description: "全部设置与运维入口",
      },
      {
        id: "ai-settings",
        path: "/admin/ai-settings",
        label: "AI 配置",
        group: "settings",
        icon: Bot,
        description: "模型与智能能力",
        settingsSection: "integrations",
      },
      {
        id: "email-settings",
        path: "/admin/email-settings",
        label: "邮箱设置",
        group: "settings",
        icon: Mail,
        description: "SMTP 与邮件模板",
        settingsSection: "integrations",
      },
      {
        id: "storage",
        path: "/admin/storage",
        label: "存储配置",
        group: "settings",
        icon: HardDrive,
        description: "本地与对象存储",
        settingsSection: "integrations",
      },
      {
        id: "api-keys",
        path: "/admin/api-keys",
        label: "API Key",
        group: "settings",
        icon: KeyRound,
        description: "PicGo 等客户端上传密钥",
        settingsSection: "integrations",
      },
      {
        id: "translation",
        path: "/admin/translation",
        label: "翻译管理",
        group: "settings",
        icon: Languages,
        description: "双语内容与翻译状态",
        settingsSection: "integrations",
      },
      {
        id: "wizard",
        path: "/admin/wizard",
        label: "建站向导",
        group: "settings",
        icon: Wand2,
        description: "快速初始化站点",
        settingsSection: "tools",
      },
      {
        id: "backups",
        path: "/admin/backups",
        label: "数据备份",
        group: "settings",
        icon: Database,
        description: "备份与恢复",
        settingsSection: "ops",
      },
      {
        id: "audit-logs",
        path: "/admin/audit-logs",
        label: "审计日志",
        group: "settings",
        icon: ScrollText,
        description: "操作审计记录",
        settingsSection: "ops",
      },
      {
        id: "migration",
        path: "/admin/migration",
        label: "数据迁移",
        group: "settings",
        icon: RefreshCw,
        description: "导入导出与迁移",
        settingsSection: "ops",
      },
      {
        id: "system-status",
        path: "/admin/system-status",
        label: "系统状态",
        group: "settings",
        icon: Activity,
        description: "运行健康与诊断",
        settingsSection: "ops",
      },
      {
        id: "qa",
        path: "/admin/qa",
        label: "知识问答",
        group: "settings",
        icon: Sparkles,
        description: "实验性问答模块",
        settingsSection: "tools",
      },
    ],
  },
];

export function isNavItemActive(pathname: string, item: Pick<AdminNavItem, "path" | "exact">): boolean {
  if (item.exact) {
    return pathname === item.path;
  }
  return pathname === item.path || pathname.startsWith(`${item.path}/`);
}

export function isGroupActive(pathname: string, group: AdminNavGroup): boolean {
  return group.items.some((item) => isNavItemActive(pathname, item));
}

export function findNavItemByPath(pathname: string): AdminNavItem | null {
  let best: AdminNavItem | null = null;
  for (const group of adminNavGroups) {
    for (const item of group.items) {
      if (!isNavItemActive(pathname, item)) continue;
      if (!best || item.path.length > best.path.length) {
        best = item;
      }
    }
    for (const item of group.items) {
      for (const child of item.children ?? []) {
        if (pathname === child.path || pathname.startsWith(`${child.path}/`)) {
          if (!best || child.path.length > best.path.length) {
            // Return parent item for breadcrumb context; label comes from child when needed
            best = item;
          }
        }
      }
    }
  }
  return best;
}

export function getNavTitle(pathname: string): string {
  if (pathname.startsWith("/admin/pages/edit") || pathname === "/admin/pages/new") {
    return "页面编辑";
  }
  if (pathname.startsWith("/admin/articles/edit") || pathname === "/admin/articles/new") {
    return "文章编辑";
  }
  for (const group of adminNavGroups) {
    for (const item of group.items) {
      for (const child of item.children ?? []) {
        if (pathname === child.path || pathname.startsWith(`${child.path}/`)) {
          return child.label;
        }
      }
      if (isNavItemActive(pathname, item)) {
        return item.label;
      }
    }
  }
  return "管理后台";
}

export function filterNavGroups(
  hasPermission: (permission: string) => boolean,
  options?: { query?: string },
): AdminNavGroup[] {
  const query = options?.query?.trim().toLowerCase() ?? "";

  return adminNavGroups
    .map((group) => {
      const items = group.items.filter((item) => {
        if (!isAdminRouteVisibleInNavigation(item.path)) return false;
        const permission = getAdminRoutePermission(item.path);
        if (!hasAdminRoutePermission(permission, hasPermission)) return false;
        if (!query) return true;
        const haystack = [item.label, item.description ?? "", ...(item.children?.map((c) => c.label) ?? [])]
          .join(" ")
          .toLowerCase();
        return haystack.includes(query);
      });
      return { ...group, items };
    })
    .filter((group) => group.items.length > 0);
}

export function getSettingsHubItems(
  hasPermission: (permission: string) => boolean,
): AdminNavItem[] {
  const settingsGroup = adminNavGroups.find((g) => g.id === "settings");
  if (!settingsGroup) return [];

  return settingsGroup.items.filter((item) => {
    if (item.id === "settings-hub") return false;
    if (!isAdminRouteVisibleInNavigation(item.path)) return false;
    const permission = getAdminRoutePermission(item.path);
    return hasAdminRoutePermission(permission, hasPermission);
  });
}

export const SETTINGS_SECTION_LABELS: Record<
  NonNullable<AdminNavItem["settingsSection"]>,
  string
> = {
  integrations: "集成与能力",
  ops: "运维与安全",
  tools: "工具",
};

export function isAdminEditorPath(pathname: string): boolean {
  return (
    pathname === "/admin/pages/new" ||
    pathname.startsWith("/admin/pages/edit/") ||
    pathname === "/admin/articles/new" ||
    pathname.startsWith("/admin/articles/edit/")
  );
}
