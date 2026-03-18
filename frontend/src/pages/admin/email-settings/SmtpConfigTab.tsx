import type { EmailConfig } from "./types";

interface SmtpConfigTabProps {
  config: EmailConfig;
  onChange: (config: EmailConfig) => void;
  onSave: () => void;
  onTest: () => void;
  isSaving: boolean;
  isTesting: boolean;
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
      {/* SMTP Server Settings */}
      <div>
        <h3 className="text-base font-semibold text-gray-900 mb-4">SMTP 服务器</h3>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              SMTP 主机 <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={smtp.host}
              onChange={(e) => updateSmtp("host", e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
              placeholder="smtp.example.com"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              端口 <span className="text-red-500">*</span>
            </label>
            <input
              type="number"
              value={smtp.port}
              onChange={(e) => updateSmtp("port", parseInt(e.target.value) || 0)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
              placeholder="587"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">用户名</label>
            <input
              type="text"
              value={smtp.username}
              onChange={(e) => updateSmtp("username", e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
              placeholder="user@example.com"
              autoComplete="off"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              密码
              {smtp.password && (
                <span className="text-xs text-green-600 font-normal ml-2">(已配置)</span>
              )}
            </label>
            <input
              type="password"
              value={smtp.password}
              onChange={(e) => updateSmtp("password", e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
              placeholder="输入 SMTP 密码"
              autoComplete="new-password"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              发件人邮箱 <span className="text-red-500">*</span>
            </label>
            <input
              type="email"
              value={smtp.from}
              onChange={(e) => updateSmtp("from", e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
              placeholder="noreply@example.com"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">发件人名称</label>
            <input
              type="text"
              value={smtp.fromName}
              onChange={(e) => updateSmtp("fromName", e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"
              placeholder="印迹咨询"
            />
          </div>
        </div>

        {/* TLS Options */}
        <div className="mt-4 flex flex-wrap gap-6">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={smtp.useTLS}
              onChange={(e) => updateSmtp("useTLS", e.target.checked)}
              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <span className="text-sm text-gray-700">启用 TLS 加密</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={smtp.insecureSkipVerify}
              onChange={(e) => updateSmtp("insecureSkipVerify", e.target.checked)}
              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <span className="text-sm text-gray-700">跳过证书验证</span>
            <span className="text-xs text-amber-600">(不推荐)</span>
          </label>
        </div>
      </div>

      {/* Receiver Settings */}
      <div className="pt-6 border-t border-gray-200">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-base font-semibold text-gray-900">转发通知</h3>
            <p className="text-sm text-gray-500 mt-0.5">将表单提交内容转发到指定邮箱</p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={receiver.enabled}
              onChange={(e) => updateReceiver("enabled", e.target.checked)}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600" />
          </label>
        </div>
        {receiver.enabled && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              接收邮箱 <span className="text-red-500">*</span>
            </label>
            <input
              type="email"
              value={receiver.email}
              onChange={(e) => updateReceiver("email", e.target.value)}
              className="w-full max-w-md px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
              placeholder="admin@example.com"
            />
            <p className="mt-1 text-xs text-gray-500">
              新的表单提交将通知到此邮箱
            </p>
          </div>
        )}
      </div>

      {/* Auto-Reply Settings */}
      <div className="pt-6 border-t border-gray-200">
        <div className="flex items-center justify-between mb-2">
          <div>
            <h3 className="text-base font-semibold text-gray-900">自动回复</h3>
            <p className="text-sm text-gray-500 mt-0.5">
              提交表单后自动向用户发送确认邮件
            </p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={autoReply.enabled}
              onChange={(e) => updateAutoReply("enabled", e.target.checked)}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600" />
          </label>
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between pt-6 border-t border-gray-200">
        <button
          onClick={onTest}
          disabled={isTesting || !smtp.host || !smtp.from}
          className="px-4 py-2 text-sm font-medium text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 transition-colors"
        >
          {isTesting ? "发送中..." : "发送测试邮件"}
        </button>
        <button
          onClick={onSave}
          disabled={isSaving}
          className="px-6 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          {isSaving ? "保存中..." : "保存配置"}
        </button>
      </div>
    </div>
  );
}
