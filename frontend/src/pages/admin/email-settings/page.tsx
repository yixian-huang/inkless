import { useState, useEffect } from "react";
import { getEmailSettings, updateEmailSettings, sendTestEmail } from "@/api/emailSettings";
import type { EmailConfig } from "./types";
import { defaultEmailConfig } from "./defaults";
import SmtpConfigTab from "./SmtpConfigTab";
import TemplateEditorTab from "./TemplateEditorTab";
import {
  AdminCard,
  AdminErrorBanner,
  AdminFilterChip,
  AdminLoading,
  AdminPageHeader,
  AdminSuccessBanner,
  AdminToolbar,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

type TabKey = "smtp" | "autoReply" | "forward";

interface Tab {
  key: TabKey;
  label: string;
}

const TABS: Tab[] = [
  { key: "smtp", label: "SMTP 配置" },
  { key: "autoReply", label: "自动回复模板" },
  { key: "forward", label: "转发通知模板" },
];

function deepMerge(defaults: EmailConfig, loaded: Partial<EmailConfig>): EmailConfig {
  return {
    smtp: { ...defaults.smtp, ...loaded.smtp },
    receiver: { ...defaults.receiver, ...loaded.receiver },
    autoReply: { ...defaults.autoReply, ...loaded.autoReply },
    templates: {
      autoReply: {
        ...defaults.templates.autoReply,
        ...loaded.templates?.autoReply,
      },
      forward: {
        ...defaults.templates.forward,
        ...loaded.templates?.forward,
      },
    },
  };
}

export default function AdminEmailSettingsPage() {
  useDocumentTitle("邮箱设置");
  const [config, setConfig] = useState<EmailConfig>(defaultEmailConfig);
  const [activeTab, setActiveTab] = useState<TabKey>("smtp");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [status, setStatus] = useState<{ type: "success" | "error"; message: string } | null>(
    null,
  );

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await getEmailSettings();
        if (!cancelled) {
          setConfig(deepMerge(defaultEmailConfig, data));
        }
      } catch {
        if (!cancelled) {
          setConfig(defaultEmailConfig);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const showStatus = (type: "success" | "error", message: string) => {
    setStatus({ type, message });
    setTimeout(() => setStatus(null), 4000);
  };

  const handleSave = async () => {
    setSaving(true);
    setStatus(null);
    try {
      const updated = await updateEmailSettings(config);
      setConfig(deepMerge(defaultEmailConfig, updated));
      showStatus("success", "配置保存成功");
    } catch (err: any) {
      const msg =
        err?.response?.data?.error?.message ||
        err?.response?.data?.message ||
        "保存失败，请重试";
      showStatus("error", msg);
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    const to = prompt("请输入测试收件人邮箱地址：", config.smtp.from || "");
    if (!to) return;

    setTesting(true);
    setStatus(null);
    try {
      const result = await sendTestEmail(to);
      showStatus(result.success ? "success" : "error", result.message || "测试邮件已发送");
    } catch (err: any) {
      const msg =
        err?.response?.data?.error?.message ||
        err?.response?.data?.message ||
        "发送测试邮件失败";
      showStatus("error", msg);
    } finally {
      setTesting(false);
    }
  };

  return (
    <div className="space-y-6">
      <AdminPageHeader
        title="邮箱设置"
        description="配置 SMTP 服务器、自动回复和转发通知邮件模板"
      />

      {status?.type === "success" ? <AdminSuccessBanner message={status.message} /> : null}
      {status?.type === "error" ? <AdminErrorBanner message={status.message} /> : null}

      {loading ? (
        <AdminLoading />
      ) : (
        <AdminCard padded={false}>
          <AdminToolbar className="rounded-none border-0 border-b border-slate-100 shadow-none">
            {TABS.map((tab) => (
              <AdminFilterChip
                key={tab.key}
                active={activeTab === tab.key}
                onClick={() => setActiveTab(tab.key)}
              >
                {tab.label}
              </AdminFilterChip>
            ))}
          </AdminToolbar>

          <div className="p-5 sm:p-6">
            {activeTab === "smtp" && (
              <SmtpConfigTab
                config={config}
                onChange={setConfig}
                onSave={handleSave}
                onTest={handleTest}
                isSaving={saving}
                isTesting={testing}
              />
            )}
            {activeTab === "autoReply" && (
              <TemplateEditorTab
                templates={config.templates.autoReply}
                onChange={(autoReplyTemplates) =>
                  setConfig((prev) => ({
                    ...prev,
                    templates: { ...prev.templates, autoReply: autoReplyTemplates },
                  }))
                }
                onSave={handleSave}
                isSaving={saving}
              />
            )}
            {activeTab === "forward" && (
              <TemplateEditorTab
                templates={config.templates.forward}
                onChange={(forwardTemplates) =>
                  setConfig((prev) => ({
                    ...prev,
                    templates: { ...prev.templates, forward: forwardTemplates },
                  }))
                }
                onSave={handleSave}
                isSaving={saving}
              />
            )}
          </div>
        </AdminCard>
      )}
    </div>
  );
}
