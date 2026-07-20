import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import {
  BarChart3,
  Eye,
  FileStack,
  FileText,
  Image as ImageIcon,
  Pencil,
  Plus,
  Settings2,
  Upload,
} from "lucide-react";
import { getAnalyticsSummary } from "@/api/analytics";
import { getAdminArticles } from "@/api/articles";
import { listMedia } from "@/api/media";
import { listUnifiedPages } from "@/api/unifiedPages";
import {
  AdminCard,
  AdminPageHeader,
  AdminStatCard,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { ADMIN_PAGES_PATH } from "@/router/adminAccess";

interface QuickAction {
  label: string;
  path: string;
  icon: React.ReactNode;
  color: string;
}

export default function AdminDashboardPage() {
  useDocumentTitle("仪表盘");
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({
    todayVisits: 0,
    pagesCount: 0,
    articlesCount: 0,
    mediaCount: 0,
  });
  const [errors, setErrors] = useState<Record<string, boolean>>({});

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      const results = await Promise.allSettled([
        getAnalyticsSummary(),
        listUnifiedPages(),
        getAdminArticles(1, 1),
        listMedia(1, 1),
      ]);

      const newStats = {
        todayVisits: 0,
        pagesCount: 0,
        articlesCount: 0,
        mediaCount: 0,
      };
      const newErrors: Record<string, boolean> = {};

      if (results[0].status === "fulfilled") {
        newStats.todayVisits = results[0].value.totals.today;
      } else {
        newErrors.todayVisits = true;
      }

      if (results[1].status === "fulfilled") {
        newStats.pagesCount = Array.isArray(results[1].value) ? results[1].value.length : 0;
      } else {
        newErrors.pagesCount = true;
      }

      if (results[2].status === "fulfilled") {
        newStats.articlesCount = results[2].value.total;
      } else {
        newErrors.articlesCount = true;
      }

      if (results[3].status === "fulfilled") {
        newStats.mediaCount = results[3].value.total;
      } else {
        newErrors.mediaCount = true;
      }

      setStats(newStats);
      setErrors(newErrors);
      setLoading(false);
    };

    fetchData();
  }, []);

  const quickActions: QuickAction[] = [
    {
      label: "新建文章",
      path: "/admin/articles/new",
      color: "bg-blue-50 text-blue-700 hover:bg-blue-100 border border-blue-200",
      icon: <Plus className="h-5 w-5" />,
    },
    {
      label: "上传媒体",
      path: "/admin/media",
      color: "bg-purple-50 text-purple-700 hover:bg-purple-100 border border-purple-200",
      icon: <Upload className="h-5 w-5" />,
    },
    {
      label: "编辑页面",
      path: ADMIN_PAGES_PATH,
      color: "bg-emerald-50 text-emerald-700 hover:bg-emerald-100 border border-emerald-200",
      icon: <Pencil className="h-5 w-5" />,
    },
    {
      label: "设置中心",
      path: "/admin/settings",
      color: "bg-amber-50 text-amber-700 hover:bg-amber-100 border border-amber-200",
      icon: <Settings2 className="h-5 w-5" />,
    },
    {
      label: "访问统计",
      path: "/admin/analytics",
      color: "bg-sky-50 text-sky-700 hover:bg-sky-100 border border-sky-200",
      icon: <BarChart3 className="h-5 w-5" />,
    },
    {
      label: "站点配置",
      path: "/admin/site-config",
      color: "bg-slate-50 text-slate-700 hover:bg-slate-100 border border-slate-200",
      icon: <FileStack className="h-5 w-5" />,
    },
  ];

  return (
    <div>
      <AdminPageHeader
        title="仪表盘"
        description="站点运行概览与常用操作"
      />

      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <AdminStatCard
          label="今日访问"
          value={errors.todayVisits ? "—" : stats.todayVisits}
          colorClass="bg-blue-500"
          loading={loading}
          icon={<Eye className="h-6 w-6" />}
        />
        <AdminStatCard
          label="内容页数"
          value={errors.pagesCount ? "—" : stats.pagesCount}
          colorClass="bg-emerald-500"
          loading={loading}
          icon={<FileStack className="h-6 w-6" />}
        />
        <AdminStatCard
          label="文章数"
          value={errors.articlesCount ? "—" : stats.articlesCount}
          colorClass="bg-amber-500"
          loading={loading}
          icon={<FileText className="h-6 w-6" />}
        />
        <AdminStatCard
          label="媒体文件"
          value={errors.mediaCount ? "—" : stats.mediaCount}
          colorClass="bg-purple-500"
          loading={loading}
          icon={<ImageIcon className="h-6 w-6" />}
        />
      </div>

      <AdminCard title="快捷操作" description="从这里进入最常用的管理入口">
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
          {quickActions.map((action) => (
            <Link
              key={action.label}
              to={action.path}
              className={`flex items-center gap-2.5 rounded-xl px-4 py-3 text-sm font-medium transition-colors ${action.color}`}
            >
              {action.icon}
              {action.label}
            </Link>
          ))}
        </div>
      </AdminCard>
    </div>
  );
}
