import type { EmailConfig } from "./types";
import {
  AdminButton,
  AdminCheckbox,
  AdminField,
  AdminInput,
} from "@/components/admin/ui";

interface SmtpConfigTabProps {
  config: EmailConfig;
  onChange: (config: EmailConfig) => void;
  onSave: () => void;
  onTest: () => void;
  isSaving: boolean;
  isTesting: boolean;
}

function Toggle({
  checked,
  onChange,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="relative inline-flex cursor-pointer items-center">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="peer sr-only"
      />
      <div className="h-6 w-11 rounded-full bg-slate-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-slate-300 after:bg-white after:transition-all peer-checked:bg-blue-600 peer-checked:after:translate-x-full peer-checked:after:border-white peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300/40" />
    </label>
  );
}

export default function SmtpConfigTab({
  config,
  onChange,
  onSave,
  onTest,
  isSaving,
  isTesting,
}: SmtpConfigTabProps) {
  const { smtp, receiver, autoReply } = config;

  const updateSmtp = (field: string, value: string | number | boolean) => {
    onChange({
      ...config,
      smtp: { ...smtp, [field]: value },
    });
  };

  const updateReceiver = (field: string, value: string | boolean) => {
    onChange({
      ...config,
      receiver: { ...receiver, [field]: value },
    });
  };

  const updateAutoReply = (field: string, value: boolean) => {
    onChange({
      ...config,
      autoReply: { ...autoReply, [field]: value },
    });
  };

  return (
    <div className="space-y-8">
      <div>
        <h3 className="mb-4 text-base font-semibold tracking-tight text-slate-900">SMTP 服务器</h3>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <AdminField label="SMTP 主机 *">
            <AdminInput
              type="text"
              value={smtp.host}
              onChange={(e) => updateSmtp("host", e.target.value)}
              className="font-mono"
              placeholder="smtp.example.com"
            />
          </AdminField>
          <AdminField label="端口 *">
            <AdminInput
              type="number"
              value={smtp.port}
              onChange={(e) => updateSmtp("port", parseInt(e.target.value) || 0)}
              className="font-mono"
              placeholder="587"
            />
          </AdminField>
          <AdminField label="用户名">
            <AdminInput
              type="text"
              value={smtp.username}
              onChange={(e) => updateSmtp("username", e.target.value)}
              className="font-mono"
              placeholder="user@example.com"
              autoComplete="off"
            />
          </AdminField>
          <AdminField
            label={
              smtp.password
                ? "密码（已配置）"
                : "密码"
            }
          >
            <AdminInput
              type="password"
              value={smtp.password}
              onChange={(e) => updateSmtp("password", e.target.value)}
              className="font-mono"
              placeholder="输入 SMTP 密码"
              autoComplete="new-password"
            />
          </AdminField>
          <AdminField label="发件人邮箱 *">
            <AdminInput
              type="email"
              value={smtp.from}
              onChange={(e) => updateSmtp("from", e.target.value)}
              className="font-mono"
              placeholder="noreply@example.com"
            />
          </AdminField>
          <AdminField label="发件人名称">
            <AdminInput
              type="text"
              value={smtp.fromName}
              onChange={(e) => updateSmtp("fromName", e.target.value)}
              placeholder="My Site"
            />
          </AdminField>
        </div>

        <div className="mt-4 flex flex-wrap gap-6">
          <AdminCheckbox
            checked={smtp.useTLS}
            onChange={(e) => updateSmtp("useTLS", e.target.checked)}
            label="启用 TLS 加密"
          />
          <AdminCheckbox
            checked={smtp.insecureSkipVerify}
            onChange={(e) => updateSmtp("insecureSkipVerify", e.target.checked)}
            label={
              <>
                跳过证书验证{" "}
                <span className="text-xs text-amber-600">(不推荐)</span>
              </>
            }
          />
        </div>
      </div>

      <div className="border-t border-slate-100 pt-6">
        <div className="mb-4 flex items-center justify-between gap-4">
          <div>
            <h3 className="text-base font-semibold tracking-tight text-slate-900">转发通知</h3>
            <p className="mt-0.5 text-sm text-slate-500">将表单提交内容转发到指定邮箱</p>
          </div>
          <Toggle
            checked={receiver.enabled}
            onChange={(checked) => updateReceiver("enabled", checked)}
          />
        </div>
        {receiver.enabled && (
          <AdminField label="接收邮箱 *" hint="多个邮箱用逗号分隔，表单提交将通知到所有邮箱">
            <AdminInput
              type="text"
              value={receiver.emails}
              onChange={(e) => updateReceiver("emails", e.target.value)}
              className="max-w-md font-mono"
              placeholder="admin@example.com, support@example.com"
            />
          </AdminField>
        )}
      </div>

      <div className="border-t border-slate-100 pt-6">
        <div className="mb-2 flex items-center justify-between gap-4">
          <div>
            <h3 className="text-base font-semibold tracking-tight text-slate-900">自动回复</h3>
            <p className="mt-0.5 text-sm text-slate-500">提交表单后自动向用户发送确认邮件</p>
          </div>
          <Toggle
            checked={autoReply.enabled}
            onChange={(checked) => updateAutoReply("enabled", checked)}
          />
        </div>
      </div>

      <div className="flex items-center justify-between border-t border-slate-100 pt-6">
        <AdminButton
          variant="secondary"
          onClick={onTest}
          disabled={isTesting || !smtp.host || !smtp.from}
        >
          {isTesting ? "发送中…" : "发送测试邮件"}
        </AdminButton>
        <AdminButton onClick={onSave} disabled={isSaving}>
          {isSaving ? "保存中…" : "保存配置"}
        </AdminButton>
      </div>
    </div>
  );
}
