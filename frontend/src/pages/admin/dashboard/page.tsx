import { useState, useEffect, ReactNode } from "react";
import { Link } from "react-router-dom";
import { getAnalyticsSummary } from "@/api/analytics";
import { getAdminArticles } from "@/api/articles";
import { listMedia } from "@/api/media";
import { listUnifiedPages } from "@/api/unifiedPages";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { ADMIN_PAGES_PATH } from "@/router/adminAccess";

interface StatCard {
  label: string;
  value: string | number;
  icon: ReactNode;
  color: string;
}

interface QuickAction {
  label: string;
  path: string;
  icon: ReactNode;
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

      const newStats = { ...stats };
      const newErrors: Record<string, boolean> = {};

      // Analytics
      if (results[0].status === "fulfilled") {
        newStats.todayVisits = results[0].value.totals.today;
      } else {
        newErrors.todayVisits = true;
      }

      // Pages
      if (results[1].status === "fulfilled") {
        newStats.pagesCount = Array.isArray(results[1].value) ? results[1].value.length : 0;
      } else {
        newErrors.pagesCount = true;
      }

      // Articles
      if (results[2].status === "fulfilled") {
        newStats.articlesCount = results[2].value.total;
      } else {
        newErrors.articlesCount = true;
      }

      // Media
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const statCards: StatCard[] = [
    {
      label: "今日访问",
      value: errors.todayVisits ? "\u2014" : stats.todayVisits,
      color: "bg-blue-500",
      icon: (
        <svg className="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          <path strokeLinecap="round" strokeLinejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
        </svg>
      ),
    },
    {
      label: "内容页数",
      value: errors.pagesCount ? "\u2014" : stats.pagesCount,
      color: "bg-emerald-500",
      icon: (
        <svg className="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
        </svg>
      ),
    },
    {
      label: "文章数",
      value: errors.articlesCount ? "\u2014" : stats.articlesCount,
      color: "bg-amber-500",
      icon: (
        <svg className="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v12a2 2 0 01-2 2zM16 2v4M8 2v4M3 10h18M8 14h.01M12 14h.01M16 14h.01M8 18h.01M12 18h.01M16 18h.01" />
        </svg>
      ),
    },
    {
      label: "媒体文件",
      value: errors.mediaCount ? "\u2014" : stats.mediaCount,
      color: "bg-purple-500",
      icon: (
        <svg className="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
      ),
    },
  ];

  const quickActions: QuickAction[] = [
    {
      label: "新建文章",
      path: "/admin/articles/new",
      color: "bg-blue-50 text-blue-700 hover:bg-blue-100 border border-blue-200",
      icon: (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
        </svg>
      ),
    },
    {
      label: "上传媒体",
      path: "/admin/media",
      color: "bg-purple-50 text-purple-700 hover:bg-purple-100 border border-purple-200",
      icon: (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
        </svg>
      ),
    },
    {
      label: "编辑首页",
      path: ADMIN_PAGES_PATH,
      color: "bg-emerald-50 text-emerald-700 hover:bg-emerald-100 border border-emerald-200",
      icon: (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
        </svg>
      ),
    },
    {
      label: "管理页面",
      path: "/admin/pages",
      color: "bg-amber-50 text-amber-700 hover:bg-amber-100 border border-amber-200",
      icon: (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
        </svg>
      ),
    },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">仪表盘</h1>

      {/* Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        {statCards.map((card) => (
          <div
            key={card.label}
            className="bg-white rounded-lg shadow-sm overflow-hidden"
          >
            {loading ? (
              <div className="p-5">
                <div className="animate-pulse flex items-center gap-4">
                  <div className="w-14 h-14 rounded-lg bg-gray-200" />
                  <div className="flex-1">
                    <div className="h-3 bg-gray-200 rounded w-16 mb-2" />
                    <div className="h-6 bg-gray-200 rounded w-12" />
                  </div>
                </div>
              </div>
            ) : (
              <div className="p-5 flex items-center gap-4">
                <div className={`${card.color} rounded-lg p-3 shrink-0`}>
                  {card.icon}
                </div>
                <div>
                  <p className="text-sm text-gray-500">{card.label}</p>
                  <p className="text-2xl font-bold text-gray-900">{card.value}</p>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Quick Actions */}
      <div className="bg-white rounded-lg shadow-sm p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">快捷操作</h2>
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
          {quickActions.map((action) => (
            <Link
              key={action.label}
              to={action.path}
              className={`flex items-center gap-2.5 px-4 py-3 rounded-lg text-sm font-medium transition-colors ${action.color}`}
            >
              {action.icon}
              {action.label}
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
