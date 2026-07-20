import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import {
  fetchAdminGlobalConfig,
  putAdminGlobalConfigDraft,
  publishAdminGlobalConfig,
} from "@/api/globalConfig";
import { useBootstrap } from "@/contexts/BootstrapContext";
import {
  AdminButton,
  AdminField,
  AdminFilterChip,
  AdminInput,
  AdminLoading,
  AdminPageHeader,
  AdminTextarea,
  AdminToolbar,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import {
  SITE_CONFIG_GLOBAL_DEFAULT,
  type HeaderBrandMode,
  type SiteConfigGlobal,
  type SocialKind,
} from "@/types/siteConfig";
import MediaUrlField from "./MediaUrlField";
import { normalizeSiteConfig } from "./normalizeSiteConfig";

type TabKey = "basic" | "brand" | "author" | "chrome" | "seo";

const TABS: { key: TabKey; label: string; desc: string }[] = [
  { key: "basic", label: "基本信息", desc: "站点名称、标语与语言" },
  { key: "brand", label: "品牌与图片", desc: "Logo、图标与配色" },
  { key: "author", label: "作者", desc: "署名、头像与社交链接" },
  { key: "chrome", label: "页眉页脚", desc: "顶栏展示与页脚文案" },
  { key: "seo", label: "SEO", desc: "默认标题与描述" },
];

const SOCIAL_KINDS: { value: SocialKind; label: string }[] = [
  { value: "github", label: "GitHub" },
  { value: "twitter", label: "Twitter / X" },
  { value: "linkedin", label: "LinkedIn" },
  { value: "email", label: "邮箱" },
  { value: "rss", label: "RSS" },
  { value: "custom", label: "自定义" },
];

export default function AdminSiteConfigPage() {
  useDocumentTitle("站点配置");
  const { data: bootstrapData, refetch: refetchBootstrap } = useBootstrap();
  const activeThemeId = bootstrapData?.activeTheme?.themeId ?? "—";

  const [cfg, setCfg] = useState<SiteConfigGlobal>(SITE_CONFIG_GLOBAL_DEFAULT);
  const [draftVersion, setDraftVersion] = useState(0);
  const [tab, setTab] = useState<TabKey>("basic");
  const [status, setStatus] = useState<{ type: "ok" | "err"; text: string } | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);

  const bilingual = cfg.identity.localeMode === "bilingual";
  const showZh =
    cfg.identity.localeMode === "mono-zh" ||
    cfg.identity.localeMode === "bilingual" ||
    cfg.identity.defaultLocale === "zh";
  const showEn =
    cfg.identity.localeMode === "mono-en" ||
    cfg.identity.localeMode === "bilingual" ||
    cfg.identity.defaultLocale === "en";

  useEffect(() => {
    fetchAdminGlobalConfig()
      .then((s) => {
        const raw = s.draftConfig ?? s.publishedConfig;
        setCfg(normalizeSiteConfig(raw));
        setDraftVersion(s.draftVersion);
      })
      .catch((e: Error) => setStatus({ type: "err", text: "加载失败：" + e.message }))
      .finally(() => setLoading(false));
  }, []);

  const patch = useCallback((updater: (prev: SiteConfigGlobal) => SiteConfigGlobal) => {
    setCfg((prev) => updater(prev));
  }, []);

  async function save() {
    setStatus(null);
    setSaving(true);
    try {
      const r = await putAdminGlobalConfigDraft(cfg, draftVersion);
      setDraftVersion(r.draftVersion);
      setStatus({ type: "ok", text: `草稿已保存（v${r.draftVersion}）` });
    } catch (e) {
      setStatus({ type: "err", text: "保存失败：" + (e as Error).message });
    } finally {
      setSaving(false);
    }
  }

  async function publish() {
    setStatus(null);
    setPublishing(true);
    try {
      // Always persist latest form before publish
      const saved = await putAdminGlobalConfigDraft(cfg, draftVersion);
      setDraftVersion(saved.draftVersion);
      const r = await publishAdminGlobalConfig();
      setStatus({ type: "ok", text: `已发布到线上（v${r.publishedVersion}）` });
      await refetchBootstrap?.();
    } catch (e) {
      setStatus({ type: "err", text: "发布失败：" + (e as Error).message });
    } finally {
      setPublishing(false);
    }
  }

  const tabMeta = useMemo(() => TABS.find((t) => t.key === tab), [tab]);

  if (loading) {
    return <AdminLoading />;
  }

  return (
    <div className="max-w-3xl">
      <AdminPageHeader
        title="站点配置"
        description="配置全站名称、品牌图、作者与 SEO 默认值。先保存草稿，确认无误后再发布。"
      />

      <AdminToolbar className="mb-4">
        {TABS.map((t) => (
          <AdminFilterChip key={t.key} active={tab === t.key} onClick={() => setTab(t.key)}>
            {t.label}
          </AdminFilterChip>
        ))}
      </AdminToolbar>

      {tabMeta && (
        <p className="mb-4 -mt-1 text-xs text-slate-400">{tabMeta.desc}</p>
      )}

      <div className="rounded-2xl border border-slate-200/80 bg-white p-5 shadow-[0_1px_2px_rgba(15,23,42,0.04),0_4px_16px_rgba(15,23,42,0.03)]">
        {tab === "basic" && (
          <BasicTab
            cfg={cfg}
            patch={patch}
            showZh={showZh}
            showEn={showEn}
            bilingual={bilingual}
          />
        )}
        {tab === "brand" && <BrandTab cfg={cfg} patch={patch} />}
        {tab === "author" && (
          <AuthorTab cfg={cfg} patch={patch} showZh={showZh} showEn={showEn} bilingual={bilingual} />
        )}
        {tab === "chrome" && (
          <ChromeTab
            cfg={cfg}
            patch={patch}
            activeThemeId={activeThemeId}
            showZh={showZh}
            showEn={showEn}
            bilingual={bilingual}
          />
        )}
        {tab === "seo" && (
          <SEOTab cfg={cfg} patch={patch} showZh={showZh} showEn={showEn} bilingual={bilingual} />
        )}
      </div>

      <div className="sticky bottom-4 mt-5 flex flex-wrap items-center gap-3 rounded-2xl border border-slate-200/80 bg-white/95 px-4 py-3 shadow-sm backdrop-blur">
        <AdminButton size="sm" onClick={save} disabled={saving || publishing}>
          {saving ? "保存中…" : "保存草稿"}
        </AdminButton>
        <AdminButton size="sm" variant="soft" className="bg-emerald-50 text-emerald-800 border-emerald-200 hover:bg-emerald-100" onClick={publish} disabled={saving || publishing}>
          {publishing ? "发布中…" : "发布上线"}
        </AdminButton>
        <span className="text-xs text-slate-400">草稿版本 v{draftVersion}</span>
        {status?.type === "ok" ? (
          <span className="text-sm text-emerald-700">{status.text}</span>
        ) : null}
        {status?.type === "err" ? (
          <span className="text-sm text-red-600">{status.text}</span>
        ) : null}
      </div>
    </div>
  );
}

// ── Shared form primitives ──────────────────────────────────────────

type Patch = (updater: (prev: SiteConfigGlobal) => SiteConfigGlobal) => void;

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <AdminField label={label} hint={hint}>
      {children}
    </AdminField>
  );
}

function TextInput({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
}) {
  return (
    <AdminInput
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
    />
  );
}

function TextArea({
  value,
  onChange,
  placeholder,
  rows = 3,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  rows?: number;
}) {
  return (
    <AdminTextarea
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      rows={rows}
    />
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-4">
      <h2 className="text-sm font-semibold text-slate-900 border-b border-slate-100 pb-2">{title}</h2>
      <div className="space-y-4">{children}</div>
    </section>
  );
}

// ── Tabs ────────────────────────────────────────────────────────────

function BasicTab({
  cfg,
  patch,
  showZh,
  showEn,
  bilingual,
}: {
  cfg: SiteConfigGlobal;
  patch: Patch;
  showZh: boolean;
  showEn: boolean;
  bilingual: boolean;
}) {
  return (
    <div className="space-y-8">
      <Section title="站点标识">
        {showZh && (
          <Field label={bilingual ? "站点名称（中文）" : "站点名称"} hint="显示在顶栏、页脚与浏览器标题中。">
            <TextInput
              value={cfg.identity.name.zh ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  identity: { ...p.identity, name: { ...p.identity.name, zh: v } },
                }))
              }
              placeholder="例如：一弦"
            />
          </Field>
        )}
        {showEn && (
          <Field label={bilingual ? "站点名称（英文）" : "站点名称（English）"}>
            <TextInput
              value={cfg.identity.name.en ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  identity: { ...p.identity, name: { ...p.identity.name, en: v } },
                }))
              }
              placeholder="e.g. Yixian"
            />
          </Field>
        )}
        {showZh && (
          <Field
            label={bilingual ? "标语 / Slogan（中文）" : "标语 / Slogan"}
            hint="副标题或一句话介绍，可用于首页与关于页。"
          >
            <TextInput
              value={cfg.identity.tagline?.zh ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  identity: {
                    ...p.identity,
                    tagline: { ...(p.identity.tagline ?? {}), zh: v },
                  },
                }))
              }
              placeholder="例如：记录思考与创造"
            />
          </Field>
        )}
        {showEn && (
          <Field label={bilingual ? "标语 / Slogan（英文）" : "Slogan (English)"}>
            <TextInput
              value={cfg.identity.tagline?.en ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  identity: {
                    ...p.identity,
                    tagline: { ...(p.identity.tagline ?? {}), en: v },
                  },
                }))
              }
              placeholder="e.g. Notes on thinking and making"
            />
          </Field>
        )}
      </Section>

      <Section title="语言">
        <Field label="语言模式" hint="决定前台展示哪些语言与表单字段。">
          <select
            value={cfg.identity.localeMode}
            onChange={(e) =>
              patch((p) => ({
                ...p,
                identity: {
                  ...p.identity,
                  localeMode: e.target.value as SiteConfigGlobal["identity"]["localeMode"],
                },
              }))
            }
            className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm transition hover:border-slate-300 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
          >
            <option value="mono-zh">仅中文</option>
            <option value="mono-en">仅英文</option>
            <option value="bilingual">中英双语</option>
          </select>
        </Field>
        <Field label="默认语言" hint="访客首次访问时的首选语言。">
          <select
            value={cfg.identity.defaultLocale}
            onChange={(e) =>
              patch((p) => ({
                ...p,
                identity: {
                  ...p.identity,
                  defaultLocale: e.target.value as "zh" | "en",
                },
              }))
            }
            className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm transition hover:border-slate-300 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
          >
            <option value="zh">中文</option>
            <option value="en">English</option>
          </select>
        </Field>
      </Section>
    </div>
  );
}

function BrandTab({ cfg, patch }: { cfg: SiteConfigGlobal; patch: Patch }) {
  return (
    <div className="space-y-8">
      <Section title="标志与图标">
        <MediaUrlField
          label="Logo（浅色背景）"
          hint="顶栏等浅色区域使用。支持图库选择、本机上传或粘贴外链 URL。"
          value={cfg.brand.logo.light}
          preview="logo"
          onChange={(v) =>
            patch((p) => ({
              ...p,
              brand: { ...p.brand, logo: { ...p.brand.logo, light: v } },
            }))
          }
        />
        <MediaUrlField
          label="Logo（深色背景，可选）"
          hint="深色顶栏/页脚时使用；留空则回退到浅色 Logo。"
          value={cfg.brand.logo.dark ?? ""}
          preview="logo"
          onChange={(v) =>
            patch((p) => ({
              ...p,
              brand: { ...p.brand, logo: { ...p.brand.logo, dark: v || undefined } },
            }))
          }
        />
        <MediaUrlField
          label="网站图标 Favicon"
          hint="浏览器标签页小图标，建议正方形 PNG/ICO/SVG。"
          value={cfg.brand.favicon}
          preview="square"
          onChange={(v) => patch((p) => ({ ...p, brand: { ...p.brand, favicon: v } }))}
        />
        <MediaUrlField
          label="默认分享图（OG Image）"
          hint="社交平台分享链接时的默认预览图，建议 1200×630。"
          value={cfg.brand.ogImage}
          preview="wide"
          onChange={(v) => patch((p) => ({ ...p, brand: { ...p.brand, ogImage: v } }))}
        />
      </Section>

      <Section title="品牌色">
        <Field label="主色" hint="用于链接、按钮等强调色。">
          <div className="flex items-center gap-3">
            <input
              type="color"
              value={cfg.brand.primaryColor || "#1e40af"}
              onChange={(e) =>
                patch((p) => ({ ...p, brand: { ...p.brand, primaryColor: e.target.value } }))
              }
              className="h-10 w-14 border border-slate-200 rounded cursor-pointer"
            />
            <TextInput
              value={cfg.brand.primaryColor}
              onChange={(v) => patch((p) => ({ ...p, brand: { ...p.brand, primaryColor: v } }))}
              placeholder="#1e40af"
            />
          </div>
        </Field>
        <Field label="强调色（可选）">
          <div className="flex items-center gap-3">
            <input
              type="color"
              value={cfg.brand.accentColor || "#f59e0b"}
              onChange={(e) =>
                patch((p) => ({ ...p, brand: { ...p.brand, accentColor: e.target.value } }))
              }
              className="h-10 w-14 border border-slate-200 rounded cursor-pointer"
            />
            <TextInput
              value={cfg.brand.accentColor ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  brand: { ...p.brand, accentColor: v || undefined },
                }))
              }
              placeholder="#f59e0b"
            />
          </div>
        </Field>
      </Section>
    </div>
  );
}

function AuthorTab({
  cfg,
  patch,
  showZh,
  showEn,
  bilingual,
}: {
  cfg: SiteConfigGlobal;
  patch: Patch;
  showZh: boolean;
  showEn: boolean;
  bilingual: boolean;
}) {
  return (
    <div className="space-y-8">
      <Section title="作者资料">
        <Field label="显示名称" hint="文章署名、评论区等场景使用。">
          <TextInput
            value={cfg.author.name}
            onChange={(v) => patch((p) => ({ ...p, author: { ...p.author, name: v } }))}
            placeholder="作者名"
          />
        </Field>
        <MediaUrlField
          label="头像"
          hint="支持图库选择、上传或外链。"
          value={cfg.author.avatar ?? ""}
          preview="square"
          onChange={(v) =>
            patch((p) => ({
              ...p,
              author: { ...p.author, avatar: v || undefined },
            }))
          }
        />
        <Field label="所在地（可选）">
          <TextInput
            value={cfg.author.location ?? ""}
            onChange={(v) =>
              patch((p) => ({
                ...p,
                author: { ...p.author, location: v || undefined },
              }))
            }
            placeholder="例如：上海"
          />
        </Field>
        {showZh && (
          <Field label={bilingual ? "简介（中文）" : "简介"}>
            <TextArea
              value={cfg.author.bio?.zh ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  author: {
                    ...p.author,
                    bio: { ...(p.author.bio ?? {}), zh: v },
                  },
                }))
              }
              placeholder="一句话介绍自己"
            />
          </Field>
        )}
        {showEn && (
          <Field label={bilingual ? "简介（英文）" : "Bio (English)"}>
            <TextArea
              value={cfg.author.bio?.en ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  author: {
                    ...p.author,
                    bio: { ...(p.author.bio ?? {}), en: v },
                  },
                }))
              }
              placeholder="A short bio"
            />
          </Field>
        )}
      </Section>

      <Section title="社交链接">
        <div className="space-y-2">
          {cfg.author.socials.map((s, i) => (
            <div key={i} className="flex flex-wrap gap-2 items-center">
              <select
                value={s.kind}
                onChange={(e) => {
                  const next = [...cfg.author.socials];
                  next[i] = { ...s, kind: e.target.value as SocialKind };
                  patch((p) => ({ ...p, author: { ...p.author, socials: next } }));
                }}
                className="w-36 rounded-xl border border-slate-200 bg-white px-2 py-2 text-sm shadow-sm"
              >
                {SOCIAL_KINDS.map((k) => (
                  <option key={k.value} value={k.value}>
                    {k.label}
                  </option>
                ))}
              </select>
              <input
                type="text"
                value={s.url}
                onChange={(e) => {
                  const next = [...cfg.author.socials];
                  next[i] = { ...s, url: e.target.value };
                  patch((p) => ({ ...p, author: { ...p.author, socials: next } }));
                }}
                placeholder="https://…"
                className="min-w-[10rem] flex-1 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm shadow-sm"
              />
              <button
                type="button"
                onClick={() =>
                  patch((p) => ({
                    ...p,
                    author: {
                      ...p.author,
                      socials: p.author.socials.filter((_, j) => j !== i),
                    },
                  }))
                }
                className="px-2.5 py-2 text-sm text-red-600 border border-red-200 rounded-lg hover:bg-red-50"
              >
                删除
              </button>
            </div>
          ))}
          <button
            type="button"
            onClick={() =>
              patch((p) => ({
                ...p,
                author: {
                  ...p.author,
                  socials: [...p.author.socials, { kind: "github", url: "" }],
                },
              }))
            }
            className="text-sm text-blue-600 hover:text-blue-800"
          >
            + 添加社交链接
          </button>
        </div>
      </Section>
    </div>
  );
}

function ChromeTab({
  cfg,
  patch,
  activeThemeId,
  showZh,
  showEn,
  bilingual,
}: {
  cfg: SiteConfigGlobal;
  patch: Patch;
  activeThemeId: string;
  showZh: boolean;
  showEn: boolean;
  bilingual: boolean;
}) {
  const header = cfg.header ?? {};

  return (
    <div className="space-y-8">
      <Section title="页眉（顶栏）">
        <p className="text-xs text-slate-500 -mt-1">
          覆盖当前主题（<strong>{activeThemeId}</strong>）的顶栏默认行为。导航菜单请到「菜单管理」配置。
        </p>
        <Field label="品牌展示方式">
          <select
            value={header.brandMode ?? ""}
            onChange={(e) =>
              patch((p) => ({
                ...p,
                header: {
                  ...(p.header ?? {}),
                  brandMode: (e.target.value || undefined) as HeaderBrandMode | undefined,
                },
              }))
            }
            className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm transition hover:border-slate-300 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
          >
            <option value="">跟随主题默认</option>
            <option value="text">文字（站点名 / 作者名）</option>
            <option value="logo">Logo 图片</option>
            <option value="avatar">头像 + 名称</option>
            <option value="none">不显示</option>
          </select>
        </Field>
        <label className="flex items-center gap-2 text-sm text-slate-700">
          <input
            type="checkbox"
            checked={header.showRssLink ?? false}
            onChange={(e) =>
              patch((p) => ({
                ...p,
                header: { ...(p.header ?? {}), showRssLink: e.target.checked },
              }))
            }
            className="rounded border-slate-200"
          />
          在顶栏显示 RSS 链接（需在「功能开关」中启用博客 RSS）
        </label>
        <label className="flex items-center gap-2 text-sm text-slate-700">
          <input
            type="checkbox"
            checked={header.showSocials ?? false}
            onChange={(e) =>
              patch((p) => ({
                ...p,
                header: { ...(p.header ?? {}), showSocials: e.target.checked },
              }))
            }
            className="rounded border-slate-200"
          />
          在顶栏显示作者社交链接
        </label>
      </Section>

      <Section title="页脚">
        {showZh && (
          <Field label={bilingual ? "版权文案（中文）" : "版权文案"} hint="留空则自动生成「© 年份 站点名」。">
            <TextInput
              value={cfg.footer.copyright?.zh ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  footer: {
                    ...p.footer,
                    copyright: { ...(p.footer.copyright ?? {}), zh: v },
                  },
                }))
              }
              placeholder="© 2026 一弦"
            />
          </Field>
        )}
        {showEn && (
          <Field label={bilingual ? "版权文案（英文）" : "Copyright (English)"}>
            <TextInput
              value={cfg.footer.copyright?.en ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  footer: {
                    ...p.footer,
                    copyright: { ...(p.footer.copyright ?? {}), en: v },
                  },
                }))
              }
              placeholder="© 2026 Yixian"
            />
          </Field>
        )}
        <Field label="ICP 备案号" hint="中国大陆站点备案号；留空则不显示。">
          <TextInput
            value={cfg.footer.icp ?? ""}
            onChange={(v) =>
              patch((p) => ({
                ...p,
                footer: { ...p.footer, icp: v || undefined },
              }))
            }
            placeholder="京 ICP 备 xxxxxxxx 号"
          />
        </Field>
      </Section>
    </div>
  );
}

function SEOTab({
  cfg,
  patch,
  showZh,
  showEn,
  bilingual,
}: {
  cfg: SiteConfigGlobal;
  patch: Patch;
  showZh: boolean;
  showEn: boolean;
  bilingual: boolean;
}) {
  return (
    <div className="space-y-8">
      <Section title="默认 SEO">
        <p className="text-xs text-slate-500 -mt-1">
          未单独配置 SEO 的页面/文章会使用这里的默认值。单页仍可在页面/文章编辑器中覆盖。
        </p>
        {showZh && (
          <Field
            label={bilingual ? "默认 SEO 标题（中文）" : "默认 SEO 标题"}
            hint="首页或未填标题时的浏览器标题；留空则使用站点名称。"
          >
            <TextInput
              value={cfg.seo.defaultTitle?.zh ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  seo: {
                    ...p.seo,
                    defaultTitle: { ...(p.seo.defaultTitle ?? {}), zh: v },
                  },
                }))
              }
              placeholder="与站点名称相同或更完整的品牌句"
            />
          </Field>
        )}
        {showEn && (
          <Field label={bilingual ? "默认 SEO 标题（英文）" : "Default SEO title (English)"}>
            <TextInput
              value={cfg.seo.defaultTitle?.en ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  seo: {
                    ...p.seo,
                    defaultTitle: { ...(p.seo.defaultTitle ?? {}), en: v },
                  },
                }))
              }
            />
          </Field>
        )}
        <Field
          label="标题模板"
          hint="子页面标题拼接规则。可用占位符：{page} 当前页标题，{site} 站点名。"
        >
          <TextInput
            value={cfg.seo.titleTemplate ?? ""}
            onChange={(v) =>
              patch((p) => ({
                ...p,
                seo: { ...p.seo, titleTemplate: v || undefined },
              }))
            }
            placeholder="{page} | {site}"
          />
        </Field>
        {showZh && (
          <Field label={bilingual ? "默认描述（中文）" : "默认描述"} hint="meta description，建议 50–160 字。">
            <TextArea
              value={cfg.seo.defaultDescription?.zh ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  seo: {
                    ...p.seo,
                    defaultDescription: { ...(p.seo.defaultDescription ?? {}), zh: v },
                  },
                }))
              }
              placeholder="用一两句话介绍这个站点"
            />
          </Field>
        )}
        {showEn && (
          <Field label={bilingual ? "默认描述（英文）" : "Default description (English)"}>
            <TextArea
              value={cfg.seo.defaultDescription?.en ?? ""}
              onChange={(v) =>
                patch((p) => ({
                  ...p,
                  seo: {
                    ...p.seo,
                    defaultDescription: { ...(p.seo.defaultDescription ?? {}), en: v },
                  },
                }))
              }
            />
          </Field>
        )}
        <Field label="Twitter / X 账号" hint="可选，用于 Twitter Card。">
          <TextInput
            value={cfg.seo.twitterHandle ?? ""}
            onChange={(v) =>
              patch((p) => ({
                ...p,
                seo: { ...p.seo, twitterHandle: v || undefined },
              }))
            }
            placeholder="@yourhandle"
          />
        </Field>
      </Section>
    </div>
  );
}
