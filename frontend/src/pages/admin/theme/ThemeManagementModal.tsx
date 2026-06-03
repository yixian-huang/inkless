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
      setGalleryMsg(`已激活主题「${theme.nameZh || theme.name}」`);
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="bg-white rounded-xl shadow-xl w-[90vw] max-w-3xl max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between shrink-0">
          <h3 className="text-lg font-semibold text-gray-900">主题管理</h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 text-xl"
          >
            &times;
          </button>
        </div>

        {/* Tabs */}
        <div className="px-6 border-b border-gray-200 shrink-0">
          <nav className="flex -mb-px space-x-6">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`pb-3 pt-3 px-1 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === tab.id
                    ? "border-blue-500 text-blue-600"
                    : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
                }`}
              >
                {tab.label}
              </button>
            ))}
          </nav>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Gallery tab */}
          {activeTab === "gallery" && (
            <div>
              {galleryMsg && (
                <div className={`mb-4 p-3 rounded-md text-sm ${galleryMsg.includes("失败") ? "bg-red-50 text-red-700" : "bg-green-50 text-green-700"}`}>
                  {galleryMsg}
                </div>
              )}

              {galleryLoading ? (
                <div className="py-12 text-center text-gray-500">加载中...</div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {installedThemes.map((theme) => (
                    <div
                      key={theme.id}
                      className={`rounded-lg border-2 overflow-hidden transition-all ${
                        theme.isActive
                          ? "border-blue-500 ring-2 ring-blue-200"
                          : "border-gray-200 hover:border-gray-300"
                      }`}
                    >
                      <div
                        className="h-[50px] w-full relative"
                        style={{ background: theme.preview || "linear-gradient(135deg, #667 0%, #999 100%)" }}
                      >
                        {theme.isActive && (
                          <span className="absolute top-2 right-2 bg-blue-500 text-white text-xs font-medium px-2 py-0.5 rounded">
                            当前主题
                          </span>
                        )}
                        {theme.source === "external" && (
                          <span className="absolute top-2 left-2 bg-purple-500 text-white text-xs font-medium px-2 py-0.5 rounded">
                            外部
                          </span>
                        )}
                      </div>
                      <div className="p-3">
                        <div className="flex items-center justify-between mb-1">
                          <h4 className="font-bold text-gray-900 text-sm">{theme.nameZh || theme.name}</h4>
                          <span className="text-xs text-gray-400">v{theme.version}</span>
                        </div>
                        <p className="text-xs text-gray-500 mb-2">{theme.description}</p>
                        <div className="flex items-center justify-between">
                          <span className="text-xs text-gray-400">{theme.author}</span>
                          <div className="flex gap-2">
                            {theme.source === "external" && !theme.isActive && (
                              <button
                                onClick={() => handleUninstall(theme)}
                                className="px-2.5 py-1 rounded text-xs font-medium text-red-600 border border-red-200 hover:bg-red-50 transition-colors"
                              >
                                卸载
                              </button>
                            )}
                            <button
                              onClick={() => handleActivate(theme)}
                              disabled={theme.isActive}
                              className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                                theme.isActive
                                  ? "bg-gray-100 text-gray-400 cursor-not-allowed"
                                  : "bg-blue-600 text-white hover:bg-blue-700"
                              }`}
                            >
                              {theme.isActive ? "已激活" : "激活"}
                            </button>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Install tab */}
          {activeTab === "install" && (
            <div>
              <p className="text-sm text-gray-500 mb-4">
                输入外部主题的 UMD bundle URL，加载预览后安装
              </p>

              {installMsg && (
                <div className={`mb-4 p-3 rounded-md text-sm ${installMsg.includes("成功") ? "bg-green-50 text-green-700" : "bg-red-50 text-red-700"}`}>
                  {installMsg}
                </div>
              )}

              <div className="flex gap-3 mb-6">
                <input
                  type="url"
                  value={installUrl}
                  onChange={(e) => setInstallUrl(e.target.value)}
                  placeholder="https://example.com/theme-bundle.umd.js"
                  className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
                <button
                  onClick={handleLoadPreview}
                  disabled={installLoading || !installUrl.trim()}
                  className="px-4 py-2 bg-gray-800 text-white rounded-md text-sm hover:bg-gray-900 disabled:opacity-50 whitespace-nowrap"
                >
                  {installLoading ? "加载中..." : "加载预览"}
                </button>
              </div>

              {installPreview && (
                <div className="border border-gray-200 rounded-lg p-4">
                  <h4 className="font-bold text-gray-900 mb-2">{installPreview.nameZh || installPreview.name}</h4>
                  <p className="text-sm text-gray-500 mb-2">{installPreview.description}</p>
                  <div className="flex items-center gap-4 text-xs text-gray-400 mb-4">
                    <span>作者: {installPreview.author}</span>
                    <span>版本: {installPreview.version}</span>
                    <span>ID: {installPreview.themeId}</span>
                  </div>
                  <button
                    onClick={handleInstall}
                    disabled={installLoading}
                    className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700 disabled:opacity-50"
                  >
                    {installLoading ? "安装中..." : "安装主题"}
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
