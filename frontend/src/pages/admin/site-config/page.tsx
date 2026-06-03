import { useEffect, useState } from "react";
import {
  fetchAdminGlobalConfig,
  putAdminGlobalConfigDraft,
  publishAdminGlobalConfig,
} from "@/api/globalConfig";
import { useBootstrap } from "@/contexts/BootstrapContext";
import { SITE_CONFIG_GLOBAL_DEFAULT, type SiteConfigGlobal, type HeaderBrandMode } from "@/types/siteConfig";

type TabKey = "identity" | "brand" | "author" | "header" | "footer" | "seo";

const TABS: { key: TabKey; label: string }[] = [
  { key: "identity", label: "Identity" },
  { key: "brand",    label: "Brand" },
  { key: "author",   label: "Author" },
  { key: "header",   label: "Header" },
  { key: "footer",   label: "Footer" },
  { key: "seo",      label: "SEO" },
];

export default function AdminSiteConfigPage() {
  const { data: bootstrapData } = useBootstrap();
  const activeThemeId = bootstrapData?.activeTheme?.themeId ?? "—";
  const [cfg, setCfg] = useState<SiteConfigGlobal>(SITE_CONFIG_GLOBAL_DEFAULT);
  const [draftVersion, setDraftVersion] = useState(0);
  const [tab, setTab] = useState<TabKey>("identity");
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchAdminGlobalConfig()
      .then((s) => {
        if (s.draftConfig && (s.draftConfig as unknown as Record<string, unknown>).identity) {
          setCfg(s.draftConfig);
        } else if (s.publishedConfig && (s.publishedConfig as unknown as Record<string, unknown>).identity) {
          setCfg(s.publishedConfig);
        }
        setDraftVersion(s.draftVersion);
      })
      .catch((e: Error) => setStatus("Load failed: " + e.message))
      .finally(() => setLoading(false));
  }, []);

  async function save() {
    setStatus("");
    try {
      const r = await putAdminGlobalConfigDraft(cfg, draftVersion);
      setDraftVersion(r.draftVersion);
      setStatus("Draft saved (v" + r.draftVersion + ")");
    } catch (e) {
      setStatus("Save failed: " + (e as Error).message);
    }
  }

  async function publish() {
    setStatus("");
    try {
      const r = await publishAdminGlobalConfig();
      setStatus("Published (v" + r.publishedVersion + ")");
    } catch (e) {
      setStatus("Publish failed: " + (e as Error).message);
    }
  }

  if (loading) return <div className="p-4">Loading…</div>;

  return (
    <div className="p-4 max-w-3xl">
      <h1 className="text-xl font-semibold mb-4">Site Config</h1>
      <div className="border-b mb-4 flex gap-1">
        {TABS.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 text-sm border-b-2 ${tab === t.key ? "border-blue-600 text-blue-600" : "border-transparent text-gray-600"}`}
          >
            {t.label}
          </button>
        ))}
      </div>
      {tab === "identity" && <IdentityTab cfg={cfg} setCfg={setCfg} />}
      {tab === "brand"    && <BrandTab cfg={cfg} setCfg={setCfg} />}
      {tab === "author"   && <AuthorTab cfg={cfg} setCfg={setCfg} />}
      {tab === "header"   && <HeaderTab cfg={cfg} setCfg={setCfg} activeThemeId={activeThemeId} />}
      {tab === "footer"   && <FooterTab cfg={cfg} setCfg={setCfg} />}
      {tab === "seo"      && <SEOTab cfg={cfg} setCfg={setCfg} />}
      <div className="mt-6 flex gap-2 items-center">
        <button onClick={save} className="px-4 py-2 bg-blue-600 text-white rounded">Save Draft</button>
        <button onClick={publish} className="px-4 py-2 bg-green-600 text-white rounded">Publish</button>
        {status && <span className="text-sm text-gray-700">{status}</span>}
      </div>
    </div>
  );
}

interface TabProps {
  cfg: SiteConfigGlobal;
  setCfg: (cfg: SiteConfigGlobal) => void;
}

function IdentityTab({ cfg, setCfg }: TabProps) {
  return (
    <div className="space-y-3">
      <div>
        <label className="block text-sm font-medium">Site name (zh)</label>
        <input
          type="text"
          value={cfg.identity.name.zh ?? ""}
          onChange={(e) => setCfg({ ...cfg, identity: { ...cfg.identity, name: { ...cfg.identity.name, zh: e.target.value } } })}
          className="border rounded px-2 py-1 w-full"
        />
      </div>
      <div>
        <label className="block text-sm font-medium">Site name (en)</label>
        <input
          type="text"
          value={cfg.identity.name.en ?? ""}
          onChange={(e) => setCfg({ ...cfg, identity: { ...cfg.identity, name: { ...cfg.identity.name, en: e.target.value } } })}
          className="border rounded px-2 py-1 w-full"
        />
      </div>
      <div>
        <label className="block text-sm font-medium">Locale mode</label>
        <select
          value={cfg.identity.localeMode}
          onChange={(e) => setCfg({ ...cfg, identity: { ...cfg.identity, localeMode: e.target.value as SiteConfigGlobal["identity"]["localeMode"] } })}
          className="border rounded px-2 py-1"
        >
          <option value="mono-zh">mono-zh</option>
          <option value="mono-en">mono-en</option>
          <option value="bilingual">bilingual</option>
        </select>
      </div>
      <div>
        <label className="block text-sm font-medium">Default locale</label>
        <select
          value={cfg.identity.defaultLocale}
          onChange={(e) => setCfg({ ...cfg, identity: { ...cfg.identity, defaultLocale: e.target.value as "zh" | "en" } })}
          className="border rounded px-2 py-1"
        >
          <option value="zh">zh</option>
          <option value="en">en</option>
        </select>
      </div>
    </div>
  );
}

function BrandTab({ cfg, setCfg }: TabProps) {
  return (
    <div className="space-y-3">
      <Input label="Logo (light) URL" value={cfg.brand.logo.light} onChange={(v) => setCfg({ ...cfg, brand: { ...cfg.brand, logo: { ...cfg.brand.logo, light: v } } })} />
      <Input label="Logo (dark) URL" value={cfg.brand.logo.dark ?? ""} onChange={(v) => setCfg({ ...cfg, brand: { ...cfg.brand, logo: { ...cfg.brand.logo, dark: v } } })} />
      <Input label="Favicon URL" value={cfg.brand.favicon} onChange={(v) => setCfg({ ...cfg, brand: { ...cfg.brand, favicon: v } })} />
      <Input label="Default OG image URL" value={cfg.brand.ogImage} onChange={(v) => setCfg({ ...cfg, brand: { ...cfg.brand, ogImage: v } })} />
      <Input label="Primary color (hex)" value={cfg.brand.primaryColor} onChange={(v) => setCfg({ ...cfg, brand: { ...cfg.brand, primaryColor: v } })} />
    </div>
  );
}

function AuthorTab({ cfg, setCfg }: TabProps) {
  return (
    <div className="space-y-3">
      <Input label="Name" value={cfg.author.name} onChange={(v) => setCfg({ ...cfg, author: { ...cfg.author, name: v } })} />
      <Input label="Avatar URL" value={cfg.author.avatar ?? ""} onChange={(v) => setCfg({ ...cfg, author: { ...cfg.author, avatar: v } })} />
      <Input label="Location" value={cfg.author.location ?? ""} onChange={(v) => setCfg({ ...cfg, author: { ...cfg.author, location: v } })} />
      <div>
        <label className="block text-sm font-medium mb-1">Socials</label>
        {cfg.author.socials.map((s, i) => (
          <div key={i} className="flex gap-2 mb-1">
            <input
              type="text"
              value={s.kind}
              onChange={(e) => {
                const next = [...cfg.author.socials];
                next[i] = { ...s, kind: e.target.value as typeof s.kind };
                setCfg({ ...cfg, author: { ...cfg.author, socials: next } });
              }}
              placeholder="kind"
              className="border rounded px-2 py-1 w-28"
            />
            <input
              type="text"
              value={s.url}
              onChange={(e) => {
                const next = [...cfg.author.socials];
                next[i] = { ...s, url: e.target.value };
                setCfg({ ...cfg, author: { ...cfg.author, socials: next } });
              }}
              placeholder="url"
              className="border rounded px-2 py-1 flex-1"
            />
            <button
              onClick={() => setCfg({ ...cfg, author: { ...cfg.author, socials: cfg.author.socials.filter((_, j) => j !== i) } })}
              className="px-2 text-red-600"
            >×</button>
          </div>
        ))}
        <button
          onClick={() => setCfg({ ...cfg, author: { ...cfg.author, socials: [...cfg.author.socials, { kind: "github", url: "" }] } })}
          className="text-sm text-blue-600"
        >+ Add social</button>
      </div>
    </div>
  );
}

function HeaderTab({ cfg, setCfg, activeThemeId }: TabProps & { activeThemeId: string }) {
  const header = cfg.header ?? {};
  const setHeader = (patch: Partial<NonNullable<SiteConfigGlobal["header"]>>) => {
    setCfg({ ...cfg, header: { ...header, ...patch } });
  };

  return (
    <div className="space-y-3">
      <p className="text-sm text-gray-500">
        Overrides header defaults for the active theme (<strong>{activeThemeId}</strong>).
        Brand / RSS / social toggles apply when the theme exposes them in its settingSchema.
        Navigation comes from Menus or theme pages.
      </p>
      <div>
        <label className="block text-sm font-medium">Brand mark mode</label>
        <select
          value={header.brandMode ?? ""}
          onChange={(e) => setHeader({ brandMode: (e.target.value || undefined) as HeaderBrandMode | undefined })}
          className="border rounded px-2 py-1 w-full"
        >
          <option value="">Use theme default</option>
          <option value="text">Text (site / author name)</option>
          <option value="logo">Logo image</option>
          <option value="avatar">Avatar + name</option>
          <option value="none">Hidden</option>
        </select>
      </div>
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={header.showRssLink ?? false}
          onChange={(e) => setHeader({ showRssLink: e.target.checked })}
        />
        Show RSS link in header (requires Features → Blog → RSS)
      </label>
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={header.showSocials ?? false}
          onChange={(e) => setHeader({ showSocials: e.target.checked })}
        />
        Show author social links in header
      </label>
    </div>
  );
}

function FooterTab({ cfg, setCfg }: TabProps) {
  return (
    <div className="space-y-3">
      <Input label="Copyright (zh)" value={cfg.footer.copyright?.zh ?? ""} onChange={(v) => setCfg({ ...cfg, footer: { ...cfg.footer, copyright: { ...(cfg.footer.copyright ?? {}), zh: v } } })} />
      <Input label="Copyright (en)" value={cfg.footer.copyright?.en ?? ""} onChange={(v) => setCfg({ ...cfg, footer: { ...cfg.footer, copyright: { ...(cfg.footer.copyright ?? {}), en: v } } })} />
      <Input label="ICP (中国大陆备案号；留空隐藏)" value={cfg.footer.icp ?? ""} onChange={(v) => setCfg({ ...cfg, footer: { ...cfg.footer, icp: v } })} />
    </div>
  );
}

function SEOTab({ cfg, setCfg }: TabProps) {
  return (
    <div className="space-y-3">
      <Input label="Default title template" value={cfg.seo.titleTemplate ?? ""} placeholder="{page} | {site}" onChange={(v) => setCfg({ ...cfg, seo: { ...cfg.seo, titleTemplate: v } })} />
      <Input label="Default description (zh)" value={cfg.seo.defaultDescription?.zh ?? ""} onChange={(v) => setCfg({ ...cfg, seo: { ...cfg.seo, defaultDescription: { ...(cfg.seo.defaultDescription ?? {}), zh: v } } })} />
      <Input label="Default description (en)" value={cfg.seo.defaultDescription?.en ?? ""} onChange={(v) => setCfg({ ...cfg, seo: { ...cfg.seo, defaultDescription: { ...(cfg.seo.defaultDescription ?? {}), en: v } } })} />
      <Input label="Twitter handle" value={cfg.seo.twitterHandle ?? ""} placeholder="@yourhandle" onChange={(v) => setCfg({ ...cfg, seo: { ...cfg.seo, twitterHandle: v } })} />
    </div>
  );
}

function Input({ label, value, onChange, placeholder }: { label: string; value: string; onChange: (v: string) => void; placeholder?: string }) {
  return (
    <div>
      <label className="block text-sm font-medium">{label}</label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="border rounded px-2 py-1 w-full"
      />
    </div>
  );
}
