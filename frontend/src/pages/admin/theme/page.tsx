import { useState, useEffect, useRef } from "react";
import { getThemeSettings, updateThemeSettings } from "@/api/theme";
import { listInstalledThemes, updateThemeConfig } from "@/api/installedThemes";
import { exportTheme, importTheme } from "@/api/themeExport";
import { defaultTokens, type ThemeTokens } from "@/theme";
import { useThemeManager } from "@/plugins/hooks";
import ThemeManagementModal from "./ThemeManagementModal";
import ThemeSettingsForm from "./ThemeSettingsForm";
import FontPresetSection from "./FontPresetSection";
import {
  AdminButton,
  AdminCard,
  AdminErrorBanner,
  AdminField,
  AdminFilterChip,
  AdminInput,
  AdminLoading,
  AdminPageHeader,
  AdminSuccessBanner,
  AdminToolbar,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useBootstrap } from "@/contexts/BootstrapContext";

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
  useDocumentTitle("主题管理");
  const [activeTab, setActiveTab] = useState<TabId>("customize");
  const [showThemeModal, setShowThemeModal] = useState(false);
  const { activeTheme } = useThemeManager();
  const { refetch: refetchBootstrap } = useBootstrap();

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

  // --- Export/Import ---
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [exportImportMsg, setExportImportMsg] = useState("");

  const handleExport = async () => {
    setExportImportMsg("");
    try {
      const data = await exportTheme("my-theme");
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `theme-${Date.now()}.json`;
      a.click();
      URL.revokeObjectURL(url);
      setExportImportMsg("导出成功");
    } catch {
      setExportImportMsg("导出失败");
    }
  };

  const handleImportFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    setExportImportMsg("");
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      const text = await file.text();
      const pkg = JSON.parse(text);
      await importTheme(pkg);
      setExportImportMsg("导入成功");
    } catch {
      setExportImportMsg("导入失败，请检查文件格式");
    }
    // Reset file input
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

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

  const handleLayoutChange = (key: keyof ThemeTokens["layout"], value: string) => {
    setTokens((prev) => ({ ...prev, layout: { ...prev.layout, [key]: value } }));
  };

  const handleSave = async () => {
    setSaving(true);
    setCustomizeMsg("");
    try {
      const res = await updateThemeSettings(tokens, draftVersion);
      setDraftVersion(res.draftVersion);
      await refetchBootstrap();
      setCustomizeMsg("保存成功，前台已更新");
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
      await refetchBootstrap();
      setSettingsMsg("保存成功，前台已更新");
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
      <AdminPageHeader
        title="主题"
        description="切换主题包、导出导入与样式定制"
        actions={
          <>
            <AdminButton variant="secondary" size="sm" onClick={handleExport}>
              导出主题
            </AdminButton>
            <AdminButton variant="secondary" size="sm" onClick={() => fileInputRef.current?.click()}>
              导入主题
            </AdminButton>
            <input
              ref={fileInputRef}
              type="file"
              accept=".json"
              onChange={handleImportFile}
              className="hidden"
            />
            <AdminButton variant="secondary" size="sm" onClick={() => setShowThemeModal(true)}>
              主题管理
            </AdminButton>
          </>
        }
      />

      {exportImportMsg ? (
        exportImportMsg.includes("成功") ? (
          <AdminSuccessBanner message={exportImportMsg} />
        ) : (
          <AdminErrorBanner message={exportImportMsg} />
        )
      ) : null}

      {manifest && (
        <AdminCard padded={false} className="mb-6 overflow-hidden">
          <div
            className="h-20 w-full"
            style={{
              background: manifest.preview || "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
            }}
          />
          <div className="p-5">
            <div className="mb-2 flex items-center gap-3">
              <h3 className="text-lg font-semibold tracking-tight text-slate-900">
                {manifest.nameZh || manifest.name}
              </h3>
              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-500">
                v{manifest.version}
              </span>
            </div>
            <p className="mb-3 text-sm text-slate-500">
              {manifest.descriptionZh || manifest.description}
            </p>
            <span className="inline-flex items-center text-xs text-slate-400">
              作者: {manifest.author}
            </span>
          </div>
        </AdminCard>
      )}

      <AdminToolbar className="mb-6">
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

      {activeTab === "customize" && (
        <div>
          <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 className="mb-1 text-lg font-semibold tracking-tight text-slate-900">样式定制</h3>
              <p className="text-sm text-slate-500">在主题基础上进一步调整颜色、字体和布局</p>
            </div>
            <div className="flex items-center gap-2">
              <AdminButton variant="secondary" size="sm" onClick={handleReset}>
                重置为默认
              </AdminButton>
              <AdminButton size="sm" onClick={handleSave} disabled={saving}>
                {saving ? "保存中…" : "保存设置"}
              </AdminButton>
            </div>
          </div>

          {customizeMsg ? (
            customizeMsg.includes("成功") ? (
              <AdminSuccessBanner message={customizeMsg} />
            ) : (
              <AdminErrorBanner message={customizeMsg} />
            )
          ) : null}

          {customizeLoading ? (
            <AdminLoading />
          ) : (
            <>
              <AdminCard className="mb-6" title="颜色">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  {colorFields.map(({ key, label }) => (
                    <div key={key} className="flex items-center gap-3">
                      <input
                        type="color"
                        value={tokens.colors[key]}
                        onChange={(e) => handleColorChange(key, e.target.value)}
                        className="h-10 w-10 cursor-pointer rounded-xl border border-slate-200"
                      />
                      <div className="min-w-0 flex-1">
                        <div className="text-sm font-medium text-slate-700">{label}</div>
                        <AdminInput
                          type="text"
                          value={tokens.colors[key]}
                          onChange={(e) => handleColorChange(key, e.target.value)}
                          className="mt-1 font-mono text-xs"
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </AdminCard>

              <FontPresetSection tokens={tokens} onChange={setTokens} />

              <AdminCard className="mb-6" title="布局">
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                  <AdminField label="最大宽度">
                    <AdminInput
                      type="text"
                      value={tokens.layout.maxWidth}
                      onChange={(e) => handleLayoutChange("maxWidth", e.target.value)}
                    />
                  </AdminField>
                  <AdminField label="圆角">
                    <AdminInput
                      type="text"
                      value={tokens.layout.borderRadius}
                      onChange={(e) => handleLayoutChange("borderRadius", e.target.value)}
                    />
                  </AdminField>
                  <AdminField label="内容留白 (Content Padding)">
                    <AdminInput
                      type="text"
                      value={tokens.layout.contentPadding}
                      onChange={(e) => handleLayoutChange("contentPadding", e.target.value)}
                    />
                  </AdminField>
                  <AdminField label="区块间距 (Section Spacing)">
                    <AdminInput
                      type="text"
                      value={tokens.layout.sectionSpacing}
                      onChange={(e) => handleLayoutChange("sectionSpacing", e.target.value)}
                    />
                  </AdminField>
                  <AdminField label="内容间距 (Content Gap)">
                    <AdminInput
                      type="text"
                      value={tokens.layout.contentGap}
                      onChange={(e) => handleLayoutChange("contentGap", e.target.value)}
                    />
                  </AdminField>
                </div>
              </AdminCard>

              <AdminCard title="预览">
                <div className="flex flex-wrap gap-4">
                  <div
                    className="flex h-24 w-24 items-center justify-center rounded-2xl text-xs font-bold text-white"
                    style={{ backgroundColor: tokens.colors.primary }}
                  >
                    Primary
                  </div>
                  <div
                    className="flex h-24 w-24 items-center justify-center rounded-2xl text-xs font-bold text-white"
                    style={{ backgroundColor: tokens.colors.accent }}
                  >
                    Accent
                  </div>
                  <div
                    className="flex h-24 w-24 items-center justify-center rounded-2xl border text-xs font-bold"
                    style={{
                      backgroundColor: tokens.colors.surface,
                      color: tokens.colors.onSurface,
                      borderColor: tokens.colors.border,
                    }}
                  >
                    Surface
                  </div>
                  <div
                    className="flex h-24 w-24 items-center justify-center rounded-2xl text-xs font-bold"
                    style={{
                      backgroundColor: tokens.colors.surfaceAlt,
                      color: tokens.colors.onSurfaceMuted,
                    }}
                  >
                    Surface Alt
                  </div>
                </div>
              </AdminCard>
            </>
          )}
        </div>
      )}

      {activeTab === "presets" && tokenPresets.length > 0 && (
        <div>
          <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 className="mb-1 text-lg font-semibold tracking-tight text-slate-900">预设方案</h3>
              <p className="text-sm text-slate-500">
                选择一个预设快速应用，点击后仍需在「样式定制」中保存
              </p>
            </div>
            <AdminButton size="sm" onClick={handleSave} disabled={saving}>
              {saving ? "保存中…" : "保存设置"}
            </AdminButton>
          </div>

          {customizeMsg ? (
            customizeMsg.includes("成功") ? (
              <AdminSuccessBanner message={customizeMsg} />
            ) : (
              <AdminErrorBanner message={customizeMsg} />
            )
          ) : null}

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {tokenPresets.map((preset) => {
              const active =
                tokens.colors.primary.toLowerCase() === preset.tokens.colors.primary.toLowerCase();
              return (
                <div
                  key={preset.id}
                  className={`overflow-hidden rounded-2xl border-2 transition-all ${
                    active
                      ? "border-blue-500 ring-2 ring-blue-200"
                      : "border-slate-200 hover:border-slate-300"
                  }`}
                >
                  <div className="h-10 w-full" style={{ background: preset.preview }} />
                  <div className="flex items-center justify-between p-3">
                    <span className="text-sm font-medium text-slate-900">
                      {preset.nameZh || preset.name}
                    </span>
                    <AdminButton
                      size="sm"
                      variant={active ? "secondary" : "primary"}
                      disabled={active}
                      onClick={() => setTokens(preset.tokens)}
                    >
                      {active ? "已应用" : "应用"}
                    </AdminButton>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {activeTab === "settings" && activeTheme?.settingSchema && (
        <div>
          <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 className="mb-1 text-lg font-semibold tracking-tight text-slate-900">主题设置</h3>
              <p className="text-sm text-slate-500">配置主题的功能选项</p>
            </div>
            <div className="flex items-center gap-2">
              <AdminButton variant="secondary" size="sm" onClick={handleSettingsReset}>
                重置为默认
              </AdminButton>
              <AdminButton size="sm" onClick={handleSettingsSave} disabled={themeDbId === null}>
                保存设置
              </AdminButton>
            </div>
          </div>

          {settingsMsg ? (
            settingsMsg.includes("成功") ? (
              <AdminSuccessBanner message={settingsMsg} />
            ) : (
              <AdminErrorBanner message={settingsMsg} />
            )
          ) : null}

          {settingsLoading ? (
            <AdminLoading />
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
