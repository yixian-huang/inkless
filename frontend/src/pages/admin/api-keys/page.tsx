import { useCallback, useEffect, useMemo, useState } from "react";
import { Check, Copy, KeyRound, Plus, Trash2 } from "lucide-react";
import {
  API_KEY_SCOPE_OPTIONS,
  API_KEY_SCOPE_PRESETS,
  createAPIKey,
  listAPIKeys,
  revokeAPIKey,
  type APIKey,
} from "@/api/apiKeys";
import {
  AdminButton,
  AdminCard,
  AdminEmptyState,
  AdminErrorBanner,
  AdminField,
  AdminInput,
  AdminLoading,
  AdminModal,
  AdminPageHeader,
  AdminSuccessBanner,
  AdminTable,
  AdminTableBody,
  AdminTableHead,
  AdminTd,
  AdminTh,
  useAdminConfirm,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

function formatWhen(value?: string | null): string {
  if (!value) return "—";
  try {
    return new Date(value).toLocaleString("zh-CN");
  } catch {
    return value;
  }
}

function errMessage(err: unknown, fallback: string): string {
  const anyErr = err as { response?: { data?: { error?: { message?: string }; message?: string } } };
  return (
    anyErr?.response?.data?.error?.message ||
    anyErr?.response?.data?.message ||
    fallback
  );
}

const SCOPE_GROUP_LABEL: Record<string, string> = {
  media: "媒体",
  articles: "文章",
  pages: "页面",
  taxonomy: "分类 / 标签",
};

export default function AdminAPIKeysPage() {
  useDocumentTitle("API Key");
  const { confirm, confirmDialog } = useAdminConfirm();

  const [items, setItems] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [status, setStatus] = useState<{ type: "success" | "error"; message: string } | null>(
    null,
  );

  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [selectedScopes, setSelectedScopes] = useState<string[]>(["media:create"]);
  const [creating, setCreating] = useState(false);

  const [revealedToken, setRevealedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const scopesByGroup = useMemo(() => {
    const groups: Record<string, typeof API_KEY_SCOPE_OPTIONS> = {};
    for (const opt of API_KEY_SCOPE_OPTIONS) {
      (groups[opt.group] ??= []).push(opt);
    }
    return groups;
  }, []);

  const showStatus = (type: "success" | "error", message: string) => {
    setStatus({ type, message });
    window.setTimeout(() => setStatus(null), 4000);
  };

  const fetchKeys = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setItems(await listAPIKeys());
    } catch (err) {
      setError(errMessage(err, "加载 API Key 失败"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchKeys();
  }, [fetchKeys]);

  const openCreate = () => {
    setName("");
    setSelectedScopes(["media:create"]);
    setCreateOpen(true);
  };

  const toggleScope = (scope: string) => {
    setSelectedScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope],
    );
  };

  const applyPreset = (scopes: string[]) => {
    setSelectedScopes([...scopes]);
  };

  const handleCreate = async () => {
    const trimmed = name.trim();
    if (!trimmed) {
      showStatus("error", "请填写名称");
      return;
    }
    if (selectedScopes.length === 0) {
      showStatus("error", "请至少选择一个权限范围");
      return;
    }
    setCreating(true);
    try {
      const res = await createAPIKey({ name: trimmed, scopes: selectedScopes });
      setCreateOpen(false);
      setRevealedToken(res.token);
      setCopied(false);
      await fetchKeys();
    } catch (err) {
      showStatus("error", errMessage(err, "创建失败"));
    } finally {
      setCreating(false);
    }
  };

  const handleCopy = async () => {
    if (!revealedToken) return;
    try {
      await navigator.clipboard.writeText(revealedToken);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      showStatus("error", "复制失败，请手动选中复制");
    }
  };

  const handleRevoke = async (key: APIKey) => {
    const ok = await confirm({
      title: "吊销 API Key",
      message: `确定吊销「${key.name}」吗？使用该密钥的客户端或本地 Agent 将立即失效。`,
      confirmLabel: "吊销",
      danger: true,
    });
    if (!ok) return;
    try {
      await revokeAPIKey(key.id);
      showStatus("success", "已吊销");
      await fetchKeys();
    } catch (err) {
      showStatus("error", errMessage(err, "吊销失败"));
    }
  };

  return (
    <div className="space-y-6">
      {confirmDialog}
      <AdminPageHeader
        title="API Key"
        description="为 PicGo、本地 Agent 等客户端签发长期密钥。明文仅在创建时显示一次；有效权限为用户 RBAC 与 key scope 的交集。"
        actions={
          <AdminButton type="button" onClick={openCreate}>
            <Plus className="mr-1.5 h-4 w-4" />
            新建 Key
          </AdminButton>
        }
      />

      {status?.type === "success" ? <AdminSuccessBanner message={status.message} /> : null}
      {status?.type === "error" ? <AdminErrorBanner message={status.message} /> : null}
      {error ? <AdminErrorBanner message={error} /> : null}

      <AdminCard className="space-y-3 border-blue-100 bg-blue-50/40">
        <div className="flex items-start gap-3">
          <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-blue-100 text-blue-700">
            <KeyRound className="h-4 w-4" />
          </span>
          <div className="min-w-0 space-y-2 text-sm text-slate-600">
            <p className="font-medium text-slate-800">接入方式</p>
            <ul className="list-disc space-y-0.5 pl-4 text-xs leading-relaxed">
              <li>
                Header：<code className="rounded bg-white/80 px-1">Authorization: Bearer ink_…</code>
              </li>
              <li>
                PicGo 上传：<code className="rounded bg-white/80 px-1">POST /admin/media/upload</code>
                ，JSON Path <code className="rounded bg-white/80 px-1">url</code>
              </li>
              <li>
                内容 Agent：Admin API（文章 / 页面 SEO 与草稿），详见仓库{" "}
                <code className="rounded bg-white/80 px-1">docs/agent-access.md</code>
              </li>
            </ul>
          </div>
        </div>
      </AdminCard>

      {loading && items.length === 0 ? (
        <AdminLoading />
      ) : items.length === 0 ? (
        <AdminEmptyState
          title="还没有 API Key"
          description="可选择 PicGo 上传或内容 Agent 预设，一键签发最小权限密钥。"
          action={
            <AdminButton type="button" onClick={openCreate}>
              新建 Key
            </AdminButton>
          }
        />
      ) : (
        <AdminTable>
          <AdminTableHead>
            <tr>
              <AdminTh>名称</AdminTh>
              <AdminTh>前缀</AdminTh>
              <AdminTh>权限</AdminTh>
              <AdminTh>最近使用</AdminTh>
              <AdminTh>创建时间</AdminTh>
              <AdminTh className="text-right">操作</AdminTh>
            </tr>
          </AdminTableHead>
          <AdminTableBody>
            {items.map((key) => (
              <tr key={key.id} className="hover:bg-slate-50/80">
                <AdminTd className="font-medium text-slate-900">{key.name}</AdminTd>
                <AdminTd className="font-mono text-xs text-slate-600">{key.tokenPrefix}…</AdminTd>
                <AdminTd>
                  <div className="flex flex-wrap gap-1">
                    {(key.scopes ?? []).map((s) => (
                      <span
                        key={s}
                        className="inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600"
                      >
                        {s}
                      </span>
                    ))}
                  </div>
                </AdminTd>
                <AdminTd className="whitespace-nowrap text-slate-600">
                  {formatWhen(key.lastUsedAt)}
                </AdminTd>
                <AdminTd className="whitespace-nowrap text-slate-600">
                  {formatWhen(key.createdAt)}
                </AdminTd>
                <AdminTd className="text-right">
                  <AdminButton
                    type="button"
                    size="sm"
                    variant="danger"
                    onClick={() => void handleRevoke(key)}
                  >
                    <Trash2 className="mr-1 h-3.5 w-3.5" />
                    吊销
                  </AdminButton>
                </AdminTd>
              </tr>
            ))}
          </AdminTableBody>
        </AdminTable>
      )}

      <AdminModal
        open={createOpen}
        title="新建 API Key"
        onClose={() => !creating && setCreateOpen(false)}
        footer={
          <>
            <AdminButton
              type="button"
              variant="secondary"
              disabled={creating}
              onClick={() => setCreateOpen(false)}
            >
              取消
            </AdminButton>
            <AdminButton type="button" disabled={creating} onClick={() => void handleCreate()}>
              {creating ? "创建中…" : "创建"}
            </AdminButton>
          </>
        }
      >
        <div className="max-h-[70vh] space-y-4 overflow-y-auto pr-1">
          <AdminField label="名称" hint="便于识别，例如 picgo-mac / content-agent">
            <AdminInput
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="content-agent"
              maxLength={64}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter") void handleCreate();
              }}
            />
          </AdminField>

          <AdminField label="快速预设">
            <div className="flex flex-wrap gap-2">
              {API_KEY_SCOPE_PRESETS.map((preset) => {
                const active =
                  selectedScopes.length === preset.scopes.length &&
                  preset.scopes.every((s) => selectedScopes.includes(s));
                return (
                  <button
                    key={preset.id}
                    type="button"
                    onClick={() => applyPreset(preset.scopes)}
                    className={`rounded-xl border px-3 py-2 text-left text-xs transition ${
                      active
                        ? "border-blue-300 bg-blue-50 text-blue-900"
                        : "border-slate-200 bg-white text-slate-700 hover:border-slate-300"
                    }`}
                  >
                    <div className="font-medium">{preset.label}</div>
                    <div className="mt-0.5 text-[11px] text-slate-500">{preset.description}</div>
                  </button>
                );
              })}
            </div>
          </AdminField>

          <AdminField
            label="权限范围"
            hint="运行时权限 = 你的 RBAC ∩ 下列 scope。发布 / 删除请尽量少开。"
          >
            <div className="space-y-3">
              {Object.entries(scopesByGroup).map(([group, opts]) => (
                <div key={group} className="rounded-xl border border-slate-200 bg-slate-50/80 p-3">
                  <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
                    {SCOPE_GROUP_LABEL[group] ?? group}
                  </div>
                  <div className="grid gap-1.5 sm:grid-cols-2">
                    {opts.map((opt) => {
                      const checked = selectedScopes.includes(opt.value);
                      return (
                        <label
                          key={opt.value}
                          className={`flex cursor-pointer items-start gap-2 rounded-lg border px-2.5 py-2 text-sm ${
                            checked
                              ? "border-blue-200 bg-white"
                              : "border-transparent bg-transparent hover:bg-white/70"
                          }`}
                        >
                          <input
                            type="checkbox"
                            className="mt-0.5"
                            checked={checked}
                            onChange={() => toggleScope(opt.value)}
                          />
                          <span className="min-w-0">
                            <span className="block font-mono text-xs text-slate-800">{opt.value}</span>
                            <span className="block text-xs text-slate-500">
                              {opt.label}
                              {opt.caution ? (
                                <span className="ml-1 text-amber-700">· 慎用</span>
                              ) : null}
                            </span>
                          </span>
                        </label>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          </AdminField>
        </div>
      </AdminModal>

      <AdminModal
        open={Boolean(revealedToken)}
        title="请立即保存密钥"
        onClose={() => setRevealedToken(null)}
        footer={
          <AdminButton type="button" onClick={() => setRevealedToken(null)}>
            我已保存
          </AdminButton>
        }
      >
        <div className="space-y-3">
          <p className="text-sm text-amber-800">
            明文密钥只显示这一次。关闭后无法再查看，只能吊销后重新创建。
          </p>
          <div className="flex items-stretch gap-2">
            <code className="min-w-0 flex-1 break-all rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-xs text-slate-800">
              {revealedToken}
            </code>
            <AdminButton type="button" variant="secondary" onClick={() => void handleCopy()}>
              {copied ? (
                <>
                  <Check className="mr-1 h-4 w-4 text-emerald-600" />
                  已复制
                </>
              ) : (
                <>
                  <Copy className="mr-1 h-4 w-4" />
                  复制
                </>
              )}
            </AdminButton>
          </div>
        </div>
      </AdminModal>
    </div>
  );
}
