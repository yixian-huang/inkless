import { useState, useEffect } from "react";
import {
  listInstalledThemes,
  activateTheme,
  uninstallTheme,
  installTheme,
  type InstalledThemeDTO,
} from "@/api/installedThemes";
import { themeManager } from "@/plugins/ThemeManager";
import { useBootstrap } from "@/contexts/BootstrapContext";
import {
  AdminButton,
  AdminErrorBanner,
  AdminFilterChip,
  AdminInput,
  AdminLoading,
  AdminSuccessBanner,
  AdminToolbar,
} from "@/components/admin/ui";

type ModalTab = "gallery" | "install";

interface ThemeManagementModalProps {
  onClose: () => void;
}

export default function ThemeManagementModal({ onClose }: ThemeManagementModalProps) {
  const { refetch: refetchBootstrap } = useBootstrap();
  const [activeTab, setActiveTab] = useState<ModalTab>("gallery");

  // --- Gallery state ---
  const [installedThemes, setInstalledThemes] = useState<InstalledThemeDTO[]>([]);
  const [galleryLoading, setGalleryLoading] = useState(true);
  const [galleryMsg, setGalleryMsg] = useState("");

  // --- Install state ---
  const [installUrl, setInstallUrl] = useState("");
  const [installLoading, setInstallLoading] = useState(false);
  const [installPreview, setInstallPreview] = useState<{
    name: string;
    nameZh: string;
    description: string;
    author: string;
    version: string;
    themeId: string;
  } | null>(null);
  const [installMsg, setInstallMsg] = useState("");

  useEffect(() => {
    fetchInstalledThemes();
  }, []);

  async function fetchInstalledThemes() {
    setGalleryLoading(true);
    try {
      const themes = await listInstalledThemes();
      setInstalledThemes(themes);
    } catch {
      setGalleryMsg("加载主题列表失败");
    } finally {
      setGalleryLoading(false);
    }
  }

  async function handleActivate(theme: InstalledThemeDTO) {
    setGalleryMsg("");
    try {
      await activateTheme(theme.id);
      await refetchBootstrap();
      setGalleryMsg(`已激活「${theme.nameZh || theme.name}」，主题页面已同步`);
      fetchInstalledThemes();
    } catch {
      setGalleryMsg("激活失败");
    }
  }

  async function handleUninstall(theme: InstalledThemeDTO) {
    if (!confirm(`确定卸载主题「${theme.nameZh || theme.name}」？`)) return;
    setGalleryMsg("");
    try {
      await uninstallTheme(theme.id);
      setGalleryMsg("主题已卸载");
      fetchInstalledThemes();
    } catch (e: any) {
      setGalleryMsg(e.response?.data?.error || "卸载失败");
    }
  }

  const handleLoadPreview = async () => {
    if (!installUrl.trim()) return;
    setInstallLoading(true);
    setInstallMsg("");
    setInstallPreview(null);
    try {
      const theme = await themeManager.loadExternal(installUrl.trim());
      setInstallPreview({
        themeId: theme.manifest.id,
        name: theme.manifest.name,
        nameZh: theme.manifest.nameZh,
        description: theme.manifest.description,
        author: theme.manifest.author,
        version: theme.manifest.version,
      });
    } catch (e: any) {
      setInstallMsg(e.message || "加载失败");
    } finally {
      setInstallLoading(false);
    }
  };

  const handleInstall = async () => {
    if (!installPreview) return;
    setInstallLoading(true);
    setInstallMsg("");
    try {
      await installTheme({
        themeId: installPreview.themeId,
        name: installPreview.name,
        nameZh: installPreview.nameZh,
        description: installPreview.description,
        author: installPreview.author,
        version: installPreview.version,
        source: "external",
        externalUrl: installUrl.trim(),
      });
      setInstallMsg("主题安装成功");
      setInstallPreview(null);
      setInstallUrl("");
      fetchInstalledThemes();
    } catch (e: any) {
      setInstallMsg(e.response?.data?.error || "安装失败");
    } finally {
      setInstallLoading(false);
    }
  };

  const tabs: { id: ModalTab; label: string }[] = [
    { id: "gallery", label: "已安装主题" },
    { id: "install", label: "安装外部主题" },
  ];

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/45 p-4 backdrop-blur-[2px]"
      onClick={onClose}
    >
      <div
        className="flex max-h-[80vh] w-[90vw] max-w-3xl flex-col overflow-hidden rounded-2xl border border-slate-200/80 bg-white shadow-[0_24px_64px_rgba(15,23,42,0.18)]"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slate-100 px-5 py-4 sm:px-6">
          <h3 className="text-base font-semibold tracking-tight text-slate-900">主题管理</h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg px-2 py-1 text-lg leading-none text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
            aria-label="关闭"
          >
            ×
          </button>
        </div>

        <div className="shrink-0 border-b border-slate-100 px-4 py-2">
          <AdminToolbar className="border-0 bg-transparent p-0 shadow-none">
            {tabs.map((tab) => (
              <AdminFilterChip
                key={tab.id}
                active={activeTab === tab.id}
                onClick={() => setActiveTab(tab.id)}
              >
                {tab.label}
              </AdminFilterChip>
            ))}
          </AdminToolbar>
        </div>

        <div className="flex-1 overflow-y-auto p-5 sm:p-6">
          {activeTab === "gallery" && (
            <div>
              {galleryMsg ? (
                galleryMsg.includes("失败") ? (
                  <AdminErrorBanner message={galleryMsg} />
                ) : (
                  <AdminSuccessBanner message={galleryMsg} />
                )
              ) : null}

              {galleryLoading ? (
                <AdminLoading />
              ) : (
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  {installedThemes.map((theme) => (
                    <div
                      key={theme.id}
                      className={`overflow-hidden rounded-2xl border-2 transition-all ${
                        theme.isActive
                          ? "border-blue-500 ring-2 ring-blue-200"
                          : "border-slate-200 hover:border-slate-300"
                      }`}
                    >
                      <div
                        className="relative h-[50px] w-full"
                        style={{
                          background:
                            theme.preview || "linear-gradient(135deg, #667 0%, #999 100%)",
                        }}
                      >
                        {theme.isActive && (
                          <span className="absolute right-2 top-2 rounded-full bg-blue-500 px-2 py-0.5 text-xs font-medium text-white">
                            当前主题
                          </span>
                        )}
                        {theme.source === "external" && (
                          <span className="absolute left-2 top-2 rounded-full bg-violet-500 px-2 py-0.5 text-xs font-medium text-white">
                            外部
                          </span>
                        )}
                      </div>
                      <div className="p-3">
                        <div className="mb-1 flex items-center justify-between">
                          <h4 className="text-sm font-semibold text-slate-900">
                            {theme.nameZh || theme.name}
                          </h4>
                          <span className="text-xs text-slate-400">v{theme.version}</span>
                        </div>
                        <p className="mb-2 text-xs text-slate-500">{theme.description}</p>
                        <div className="flex items-center justify-between">
                          <span className="text-xs text-slate-400">{theme.author}</span>
                          <div className="flex gap-2">
                            {theme.source === "external" && !theme.isActive && (
                              <AdminButton
                                size="sm"
                                variant="ghost"
                                className="text-red-600 hover:bg-red-50 hover:text-red-800"
                                onClick={() => handleUninstall(theme)}
                              >
                                卸载
                              </AdminButton>
                            )}
                            <AdminButton
                              size="sm"
                              variant={theme.isActive ? "secondary" : "primary"}
                              disabled={theme.isActive}
                              onClick={() => handleActivate(theme)}
                            >
                              {theme.isActive ? "已激活" : "激活"}
                            </AdminButton>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {activeTab === "install" && (
            <div>
              <p className="mb-4 text-sm text-slate-500">
                输入外部主题的 UMD bundle URL，加载预览后安装
              </p>

              {installMsg ? (
                installMsg.includes("成功") ? (
                  <AdminSuccessBanner message={installMsg} />
                ) : (
                  <AdminErrorBanner message={installMsg} />
                )
              ) : null}

              <div className="mb-6 flex flex-col gap-3 sm:flex-row">
                <AdminInput
                  type="url"
                  value={installUrl}
                  onChange={(e) => setInstallUrl(e.target.value)}
                  placeholder="https://example.com/theme-bundle.umd.js"
                  className="flex-1"
                />
                <AdminButton
                  onClick={handleLoadPreview}
                  disabled={installLoading || !installUrl.trim()}
                  className="whitespace-nowrap"
                >
                  {installLoading ? "加载中…" : "加载预览"}
                </AdminButton>
              </div>

              {installPreview && (
                <div className="rounded-2xl border border-slate-200 p-4">
                  <h4 className="mb-2 font-semibold text-slate-900">
                    {installPreview.nameZh || installPreview.name}
                  </h4>
                  <p className="mb-2 text-sm text-slate-500">{installPreview.description}</p>
                  <div className="mb-4 flex flex-wrap items-center gap-4 text-xs text-slate-400">
                    <span>作者: {installPreview.author}</span>
                    <span>版本: {installPreview.version}</span>
                    <span>ID: {installPreview.themeId}</span>
                  </div>
                  <AdminButton onClick={handleInstall} disabled={installLoading}>
                    {installLoading ? "安装中…" : "安装主题"}
                  </AdminButton>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
