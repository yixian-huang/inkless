import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  fetchAdminFeatures,
  putAdminFeaturesDraft,
  publishAdminFeatures,
} from "@/api/features";
import { useBootstrap } from "@/contexts/BootstrapContext";
import { normalizeFeatures } from "@/lib/normalizeFeatures";
import {
  SITE_CONFIG_FEATURES_DEFAULT,
  type SiteConfigFeatures,
} from "@/types/siteConfig";

const PUBLIC_PAGE_KEYS: Array<keyof SiteConfigFeatures["publicPages"]> = [
  "home", "blog", "contact",
  "about", "experts", "coreServices", "advantages", "cases",
];

function isFeaturesShape(v: unknown): v is SiteConfigFeatures {
  return !!v && typeof v === "object" && "publicPages" in (v as Record<string, unknown>);
}

function normalizeDraft(raw: SiteConfigFeatures): SiteConfigFeatures {
  return {
    ...SITE_CONFIG_FEATURES_DEFAULT,
    ...raw,
    publicPages: { ...SITE_CONFIG_FEATURES_DEFAULT.publicPages, ...raw.publicPages },
    blog: { ...SITE_CONFIG_FEATURES_DEFAULT.blog, ...raw.blog },
  };
}

export default function AdminFeaturesPage() {
  const { refetch: refetchBootstrap, data: bootstrapData } = useBootstrap();
  const [draft, setDraft] = useState<SiteConfigFeatures>(SITE_CONFIG_FEATURES_DEFAULT);
  const [draftVersion, setDraftVersion] = useState(0);
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchAdminFeatures()
      .then((s) => {
        if (isFeaturesShape(s.draftConfig)) {
          setDraft(normalizeDraft(normalizeFeatures(s.draftConfig) ?? s.draftConfig));
        } else if (isFeaturesShape(s.publishedConfig)) {
          setDraft(normalizeDraft(normalizeFeatures(s.publishedConfig) ?? s.publishedConfig));
        }
        setDraftVersion(s.draftVersion);
      })
      .finally(() => setLoading(false));
  }, []);

  function toggle(key: keyof SiteConfigFeatures["publicPages"]) {
    setDraft((d) => ({
      ...d,
      publicPages: { ...d.publicPages, [key]: !d.publicPages[key] },
    }));
  }

  function toggleBlog(key: keyof SiteConfigFeatures["blog"]) {
    setDraft((d) => ({
      ...d,
      blog: { ...d.blog, [key]: !d.blog[key] },
    }));
  }

  async function save() {
    setStatus("");
    try {
      const r = await putAdminFeaturesDraft(draft, draftVersion);
      setDraftVersion(r.draftVersion);
      setStatus("Draft saved (v" + r.draftVersion + ")");
    } catch (e) {
      setStatus("Save failed: " + (e as Error).message);
    }
  }

  async function publish() {
    setStatus("");
    try {
      const r = await publishAdminFeatures();
      await refetchBootstrap();
      setStatus("Published v" + r.publishedVersion + " — refresh the site to see changes.");
    } catch (e) {
      setStatus("Publish failed: " + (e as Error).message);
    }
  }

  if (loading) return <div className="p-4">Loading…</div>;

  const activeThemeId = bootstrapData?.activeTheme?.themeId ?? "—";

  return (
    <div className="p-4 max-w-2xl">
      <h1 className="text-xl font-semibold mb-4">Features</h1>
      <p className="text-sm text-gray-500 mb-4">Draft v{draftVersion}</p>

      <p className="text-sm text-gray-600 mb-6">
        首页布局与 Header/Footer 由{" "}
        <Link to="/admin/theme" className="text-blue-600 underline">
          主题
        </Link>
        {" "}决定（当前激活：<strong>{activeThemeId}</strong>）。
        本页仅控制公开路由开关与博客功能。
      </p>

      <section className="mb-6">
        <h2 className="text-sm font-medium text-gray-600 mb-2">Blog</h2>
        <ul className="space-y-2">
          <li className="flex items-center gap-3">
            <input
              type="checkbox"
              id="blog-comments"
              checked={draft.blog.comments}
              onChange={() => toggleBlog("comments")}
            />
            <label htmlFor="blog-comments" className="text-sm">Comments on article pages</label>
          </li>
          <li className="flex items-center gap-3">
            <input
              type="checkbox"
              id="blog-rss"
              checked={draft.blog.rss}
              onChange={() => toggleBlog("rss")}
            />
            <label htmlFor="blog-rss" className="text-sm">RSS feed at /feed.xml</label>
          </li>
        </ul>
      </section>

      <section className="mb-6">
        <h2 className="text-sm font-medium text-gray-600 mb-2">Public pages</h2>
        <ul className="space-y-2">
          {PUBLIC_PAGE_KEYS.map((key) => (
            <li key={key} className="flex items-center gap-3">
              <input
                type="checkbox"
                id={`pp-${key}`}
                checked={draft.publicPages[key]}
                onChange={() => toggle(key)}
              />
              <label htmlFor={`pp-${key}`} className="text-sm">/{key}</label>
            </li>
          ))}
        </ul>
      </section>
      <div className="flex gap-2 items-center">
        <button type="button" onClick={save} className="px-4 py-2 bg-blue-600 text-white rounded">Save Draft</button>
        <button type="button" onClick={publish} className="px-4 py-2 bg-green-600 text-white rounded">Publish</button>
        {status && <span className="text-sm text-gray-700">{status}</span>}
      </div>
    </div>
  );
}
