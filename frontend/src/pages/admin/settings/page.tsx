import { Link } from "react-router-dom";
import { ChevronRight } from "lucide-react";
import { AdminCard, AdminEmptyState, AdminPageHeader } from "@/components/admin/ui";
import { useAuth } from "@/contexts/AuthContext";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import {
  getSettingsHubItems,
  SETTINGS_SECTION_LABELS,
  type AdminNavItem,
} from "@/pages/admin/nav/adminNav";

const SECTION_ORDER: NonNullable<AdminNavItem["settingsSection"]>[] = [
  "integrations",
  "ops",
  "tools",
];

export default function AdminSettingsPage() {
  useDocumentTitle("设置中心");
  const { hasPermission } = useAuth();
  const items = getSettingsHubItems(hasPermission);

  const bySection = SECTION_ORDER.map((section) => ({
    section,
    label: SETTINGS_SECTION_LABELS[section],
    items: items.filter((item) => item.settingsSection === section),
  })).filter((group) => group.items.length > 0);

  const unsectioned = items.filter((item) => !item.settingsSection);

  return (
    <div>
      <AdminPageHeader
        title="设置中心"
        description="集中管理集成、运维与站点工具。也可从左侧「设置」分组直接进入。"
      />

      {items.length === 0 ? (
        <AdminEmptyState
          title="暂无可用设置"
          description="当前账号没有可访问的设置项，请联系管理员开通权限。"
        />
      ) : (
        <div className="space-y-8">
          {bySection.map((group) => (
            <section key={group.section}>
              <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">
                {group.label}
              </h2>
              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                {group.items.map((item) => (
                  <SettingsCard key={item.id} item={item} />
                ))}
              </div>
            </section>
          ))}

          {unsectioned.length > 0 && (
            <section>
              <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">
                其他
              </h2>
              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                {unsectioned.map((item) => (
                  <SettingsCard key={item.id} item={item} />
                ))}
              </div>
            </section>
          )}
        </div>
      )}
    </div>
  );
}

function SettingsCard({ item }: { item: AdminNavItem }) {
  const Icon = item.icon;
  return (
    <Link to={item.path} className="group block focus:outline-none">
      <AdminCard
        padded
        className="h-full transition-all duration-150 group-hover:border-blue-200 group-hover:shadow-md group-focus-visible:ring-2 group-focus-visible:ring-blue-500/30"
      >
        <div className="flex items-start gap-3">
          <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-100 text-slate-600 group-hover:bg-blue-50 group-hover:text-blue-600 transition-colors">
            <Icon className="h-5 w-5" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex items-center justify-between gap-2">
              <h3 className="truncate text-sm font-semibold text-slate-900">{item.label}</h3>
              <ChevronRight className="h-4 w-4 shrink-0 text-slate-300 transition-transform group-hover:translate-x-0.5 group-hover:text-blue-500" />
            </div>
            {item.description ? (
              <p className="mt-1 text-xs leading-relaxed text-slate-500">{item.description}</p>
            ) : null}
          </div>
        </div>
      </AdminCard>
    </Link>
  );
}
