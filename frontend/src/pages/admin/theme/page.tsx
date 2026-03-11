import { useState, useEffect } from "react";
import { getThemeSettings, updateThemeSettings } from "@/api/theme";
import { listInstalledThemes, updateThemeConfig } from "@/api/installedThemes";
import { defaultTokens, type ThemeTokens } from "@/theme";
import { useThemeManager } from "@/plugins/hooks";
import ThemeManagementModal from "./ThemeManagementModal";
import ThemeSettingsForm from "./ThemeSettingsForm";

type TabId = "customize" | "presets" | "settings";

interface ColorField {
  key: keyof ThemeTokens["colors"];
  label: string;
}

const colorFields: ColorField[] = [
  { key: "primary", label: "主色 (Primary)" },
  { key: "primaryDark", label: "主色深 (Primary Dark)" },
  { key: "accent", label: "强调色 (Accent)" },
  { key: "accentHover", label: "强调色悬停 (Accent Hover)" },
  { key: "surface", label: "背景色 (Surface)" },
  { key: "surfaceAlt", label: "交替背景 (Surface Alt)" },
  { key: "onPrimary", label: "主色上文字 (On Primary)" },
  { key: "onSurface", label: "背景上文字 (On Surface)" },
  { key: "onSurfaceMuted", label: "次要文字 (On Surface Muted)" },
  { key: "border", label: "边框色 (Border)" },
];

export default function AdminThemePage() {
  const [activeTab, setActiveTab] = useState<TabId>("customize");
  const [showThemeModal, setShowThemeModal] = useState(false);
  const { activeTheme } = useThemeManager();

  // --- Customize tab state ---
  const [tokens, setTokens] = useState<ThemeTokens>(defaultTokens);
  const [draftVersion, setDraftVersion] = useState(0);
  const [customizeLoading, setCustomizeLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [customizeMsg, setCustomizeMsg] = useState("");

  // --- Settings tab state ---
  const [settingsValues, setSettingsValues] = useState<Record<string, any>>({});
  const [settingsLoading, setSettingsLoading] = useState(true);
  const [settingsMsg, setSettingsMsg] = useState("");
  const [themeDbId, setThemeDbId] = useState<number | null>(null);

  // Fetch settings from installed themes
  useEffect(() => {
    if (!activeTheme?.settingSchema?.length) {
      setSettingsLoading(false);
      return;
    }
    listInstalledThemes()
      .then((themes) => {
        const active = themes.find((t) => t.isActive);
        if (active) {
          setThemeDbId(active.id);
          setSettingsValues(active.config as Record<string, any> ?? {});
        }
      })
      .catch(() => {})
      .finally(() => setSettingsLoading(false));
  }, [activeTheme]);

  // Fetch current tokens
  useEffect(() => {
    getThemeSettings()
      .then((data) => {
        if (data.draftConfig) setTokens(data.draftConfig);
        setDraftVersion(data.draftVersion ?? 0);
      })
      .catch(() => {})
      .finally(() => setCustomizeLoading(false));
  }, []);

  // --- Customize handlers ---
  const tokenPresets = activeTheme?.tokenPresets ?? [];

  const handleColorChange = (key: keyof ThemeTokens["colors"], value: string) => {
    setTokens((prev) => ({ ...prev, colors: { ...prev.colors, [key]: value } }));
  };
  const handleFontChange = (key: keyof ThemeTokens["fonts"], value: string) => {
    setTokens((prev) => ({ ...prev, fonts: { ...prev.fonts, [key]: value } }));
  };
  const handleLayoutChange = (key: keyof ThemeTokens["layout"], value: string) => {
    setTokens((prev) => ({ ...prev, layout: { ...prev.layout, [key]: value } }));
  };

  const handleSave = async () => {
    setSaving(true);
    setCustomizeMsg("");
    try {
      const res = await updateThemeSettings(tokens, draftVersion);
      setDraftVersion(res.draftVersion);
      setCustomizeMsg("保存成功，刷新页面后生效");
    } catch {
      setCustomizeMsg("保存失败");
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setTokens(activeTheme?.defaultTokens ?? defaultTokens);
  };

  const manifest = activeTheme?.manifest;

  const handleSettingsSave = async () => {
    if (themeDbId === null) return;
    setSettingsMsg("");
    try {
      await updateThemeConfig(themeDbId, settingsValues);
      setSettingsMsg("保存成功，刷新页面后生效");
    } catch {
      setSettingsMsg("保存失败");
    }
  };

  const handleSettingsReset = () => {
    setSettingsValues({});
  };

  const tabs: { id: TabId; label: string }[] = [
    { id: "customize", label: "样式定制" },
    ...(tokenPresets.length > 0 ? [{ id: "presets" as TabId, label: "预设方案" }] : []),
    ...(activeTheme?.settingSchema?.length ? [{ id: "settings" as TabId, label: "主题设置" }] : []),
  ];

  return (
    <div>
      {/* Header row */}
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">主题</h2>
        <button
          onClick={() => setShowThemeModal(true)}
          className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm font-medium text-gray-700"
        >
          主题管理
        </button>
      </div>

      {/* Active theme card */}
      {manifest && (
        <div className="bg-white rounded-lg shadow overflow-hidden mb-6">
          <div
            className="h-20 w-full"
            style={{ background: manifest.preview || "linear-gradient(135deg, #667eea 0%, #764ba2 100%)" }}
          />
          <div className="p-5">
            <div className="flex items-center gap-3 mb-2">
              <h3 className="text-lg font-bold text-gray-900">
                {manifest.nameZh || manifest.name}
              </h3>
              <span className="text-xs text-gray-400 bg-gray-100 px-2 py-0.5 rounded">
                v{manifest.version}
              </span>
            </div>
            <p className="text-sm text-gray-500 mb-3">
              {manifest.descriptionZh || manifest.description}
            </p>
            <span className="inline-flex items-center text-xs text-gray-400">
              作者: {manifest.author}
            </span>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-200 mb-6">
        <nav className="flex -mb-px space-x-6">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`pb-3 px-1 text-sm font-medium border-b-2 transition-colors ${
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

      {/* Customize tab */}
      {activeTab === "customize" && (
        <div>
          <div className="flex items-center justify-between mb-6">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-1">样式定制</h3>
              <p className="text-sm text-gray-500">在主题基础上进一步调整颜色、字体和布局</p>
            </div>
            <div className="flex items-center gap-3">
              <button
                onClick={handleReset}
                className="px-4 py-2 border border-gray-300 rounded-md text-sm text-gray-700 hover:bg-gray-50"
              >
                重置为默认
              </button>
              <button
                onClick={handleSave}
                disabled={saving}
                className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700 disabled:opacity-50"
              >
                {saving ? "保存中..." : "保存设置"}
              </button>
            </div>
          </div>

          {customizeMsg && (
            <div className={`mb-4 p-3 rounded-md text-sm ${customizeMsg.includes("成功") ? "bg-green-50 text-green-700" : "bg-red-50 text-red-700"}`}>
              {customizeMsg}
            </div>
          )}

          {customizeLoading ? (
            <div className="py-12 text-center text-gray-500">加载中...</div>
          ) : (
            <>
              {/* Colors */}
              <div className="bg-white rounded-lg shadow p-6 mb-6">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">颜色</h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                  {colorFields.map(({ key, label }) => (
                    <div key={key} className="flex items-center gap-3">
                      <input
                        type="color"
                        value={tokens.colors[key]}
                        onChange={(e) => handleColorChange(key, e.target.value)}
                        className="w-10 h-10 rounded border border-gray-200 cursor-pointer"
                      />
                      <div className="flex-1">
                        <div className="text-sm font-medium text-gray-700">{label}</div>
                        <input
                          type="text"
                          value={tokens.colors[key]}
                          onChange={(e) => handleColorChange(key, e.target.value)}
                          className="w-full text-xs font-mono text-gray-500 border border-gray-200 rounded px-2 py-1 mt-1"
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Fonts */}
              <div className="bg-white rounded-lg shadow p-6 mb-6">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">字体</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">正文字体 (Sans)</label>
                    <input
                      type="text"
                      value={tokens.fonts.sans}
                      onChange={(e) => handleFontChange("sans", e.target.value)}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">标题字体 (Heading)</label>
                    <input
                      type="text"
                      value={tokens.fonts.heading}
                      onChange={(e) => handleFontChange("heading", e.target.value)}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono"
                    />
                  </div>
                </div>
              </div>

              {/* Layout */}
              <div className="bg-white rounded-lg shadow p-6 mb-6">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">布局</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">最大宽度</label>
                    <input
                      type="text"
                      value={tokens.layout.maxWidth}
                      onChange={(e) => handleLayoutChange("maxWidth", e.target.value)}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">圆角</label>
                    <input
                      type="text"
                      value={tokens.layout.borderRadius}
                      onChange={(e) => handleLayoutChange("borderRadius", e.target.value)}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">内容留白 (Content Padding)</label>
                    <input
                      type="text"
                      value={tokens.layout.contentPadding}
                      onChange={(e) => handleLayoutChange("contentPadding", e.target.value)}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">区块间距 (Section Spacing)</label>
                    <input
                      type="text"
                      value={tokens.layout.sectionSpacing}
                      onChange={(e) => handleLayoutChange("sectionSpacing", e.target.value)}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">内容间距 (Content Gap)</label>
                    <input
                      type="text"
                      value={tokens.layout.contentGap}
                      onChange={(e) => handleLayoutChange("contentGap", e.target.value)}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                    />
                  </div>
                </div>
              </div>

              {/* Preview */}
              <div className="bg-white rounded-lg shadow p-6">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">预览</h3>
                <div className="flex gap-4 flex-wrap">
                  <div
                    className="w-24 h-24 rounded-lg flex items-center justify-center text-white text-xs font-bold"
                    style={{ backgroundColor: tokens.colors.primary }}
                  >
                    Primary
                  </div>
                  <div
                    className="w-24 h-24 rounded-lg flex items-center justify-center text-white text-xs font-bold"
                    style={{ backgroundColor: tokens.colors.accent }}
                  >
                    Accent
                  </div>
                  <div
                    className="w-24 h-24 rounded-lg border flex items-center justify-center text-xs font-bold"
                    style={{
                      backgroundColor: tokens.colors.surface,
                      color: tokens.colors.onSurface,
                      borderColor: tokens.colors.border,
                    }}
                  >
                    Surface
                  </div>
                  <div
                    className="w-24 h-24 rounded-lg flex items-center justify-center text-xs font-bold"
                    style={{
                      backgroundColor: tokens.colors.surfaceAlt,
                      color: tokens.colors.onSurfaceMuted,
                    }}
                  >
                    Surface Alt
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      )}

      {/* Presets tab */}
      {activeTab === "presets" && tokenPresets.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-6">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-1">预设方案</h3>
              <p className="text-sm text-gray-500">选择一个预设快速应用，点击后仍需在「样式定制」中保存</p>
            </div>
            <div className="flex items-center gap-3">
              <button
                onClick={handleSave}
                disabled={saving}
                className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700 disabled:opacity-50"
              >
                {saving ? "保存中..." : "保存设置"}
              </button>
            </div>
          </div>

          {customizeMsg && (
            <div className={`mb-4 p-3 rounded-md text-sm ${customizeMsg.includes("成功") ? "bg-green-50 text-green-700" : "bg-red-50 text-red-700"}`}>
              {customizeMsg}
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {tokenPresets.map((preset) => {
              const active = tokens.colors.primary.toLowerCase() === preset.tokens.colors.primary.toLowerCase();
              return (
                <div
                  key={preset.id}
                  className={`rounded-lg border-2 overflow-hidden transition-all ${
                    active ? "border-blue-500 ring-2 ring-blue-200" : "border-gray-200 hover:border-gray-300"
                  }`}
                >
                  <div className="h-[40px] w-full" style={{ background: preset.preview }} />
                  <div className="p-3 flex items-center justify-between">
                    <span className="font-medium text-gray-900 text-sm">{preset.nameZh || preset.name}</span>
                    <button
                      onClick={() => setTokens(preset.tokens)}
                      disabled={active}
                      className={`px-3 py-1 rounded text-xs font-medium transition-colors ${
                        active ? "bg-gray-100 text-gray-400 cursor-not-allowed" : "bg-blue-600 text-white hover:bg-blue-700"
                      }`}
                    >
                      {active ? "已应用" : "应用"}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Settings tab */}
      {activeTab === "settings" && activeTheme?.settingSchema && (
        <div>
          <div className="flex items-center justify-between mb-6">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-1">主题设置</h3>
              <p className="text-sm text-gray-500">配置主题的功能选项</p>
            </div>
            <div className="flex items-center gap-3">
              <button
                onClick={handleSettingsReset}
                className="px-4 py-2 border border-gray-300 rounded-md text-sm text-gray-700 hover:bg-gray-50"
              >
                重置为默认
              </button>
              <button
                onClick={handleSettingsSave}
                disabled={themeDbId === null}
                className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700 disabled:opacity-50"
              >
                保存设置
              </button>
            </div>
          </div>

          {settingsMsg && (
            <div className={`mb-4 p-3 rounded-md text-sm ${settingsMsg.includes("成功") ? "bg-green-50 text-green-700" : "bg-red-50 text-red-700"}`}>
              {settingsMsg}
            </div>
          )}

          {settingsLoading ? (
            <div className="py-12 text-center text-gray-500">加载中...</div>
          ) : (
            <ThemeSettingsForm
              schema={activeTheme.settingSchema}
              values={settingsValues}
              onChange={setSettingsValues}
            />
          )}
        </div>
      )}

      {/* Theme management modal */}
      {showThemeModal && (
        <ThemeManagementModal onClose={() => setShowThemeModal(false)} />
      )}
    </div>
  );
}
