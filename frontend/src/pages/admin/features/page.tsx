import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  fetchAdminFeatures,
  putAdminFeaturesDraft,
  publishAdminFeatures,
} from "@/api/features";
import { AdminButton, AdminCard, AdminLoading, AdminPageHeader } from "@/components/admin/ui";
import { useBootstrap } from "@/contexts/BootstrapContext";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
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
  useDocumentTitle("功能开关");
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
      setStatus("草稿已保存（v" + r.draftVersion + "）");
    } catch (e) {
      setStatus("保存失败：" + (e as Error).message);
    }
  }

  async function publish() {
    setStatus("");
    try {
      const r = await publishAdminFeatures();
      await refetchBootstrap();
      setStatus("已发布 v" + r.publishedVersion + " — 刷新前台即可看到变更。");
    } catch (e) {
      setStatus("发布失败：" + (e as Error).message);
    }
  }

  if (loading) return <AdminLoading />;

  const activeThemeId = bootstrapData?.activeTheme?.themeId ?? "—";

  return (
    <div className="max-w-2xl">
      <AdminPageHeader
        title="功能开关"
        description={`草稿版本 v${draftVersion} · 控制公开路由与博客能力`}
        actions={
          <>
            <AdminButton variant="secondary" size="sm" onClick={save}>
              保存草稿
            </AdminButton>
            <AdminButton size="sm" onClick={publish}>
              发布
            </AdminButton>
          </>
        }
      />

      <p className="mb-6 text-sm text-slate-600">
        首页布局与 Header/Footer 由{" "}
        <Link to="/admin/theme" className="font-medium text-blue-600 hover:underline">
          主题
        </Link>
        {" "}决定（当前激活：<strong>{activeThemeId}</strong>）。
        本页仅控制公开路由开关与博客功能。
      </p>

      {status ? <p className="mb-4 text-sm text-slate-700">{status}</p> : null}

      <div className="space-y-4">
        <AdminCard title="博客" description="评论、RSS 与阅读信息">
          <ul className="space-y-2">
            <li className="flex items-center gap-3">
              <input
                type="checkbox"
                id="blog-comments"
                checked={draft.blog.comments}
                onChange={() => toggleBlog("comments")}
              />
              <label htmlFor="blog-comments" className="text-sm">
                文章评论区（全站开关，发布后生效）
              </label>
            </li>
            <li className="flex items-center gap-3">
              <input
                type="checkbox"
                id="blog-rss"
                checked={draft.blog.rss}
                onChange={() => toggleBlog("rss")}
              />
              <label htmlFor="blog-rss" className="text-sm">
                RSS 订阅源（/feed.xml）
              </label>
            </li>
            <li className="flex items-center gap-3">
              <input
                type="checkbox"
                id="blog-reading-meta"
                checked={draft.blog.readingMeta}
                onChange={() => toggleBlog("readingMeta")}
              />
              <label htmlFor="blog-reading-meta" className="text-sm">
                显示字数与阅读时间
              </label>
            </li>
          </ul>
          <div className="mt-4">
            <label htmlFor="blog-wpm" className="mb-1 block text-sm text-slate-600">
              阅读速度（词/分钟）
            </label>
            <input
              id="blog-wpm"
              type="number"
              min={100}
              max={600}
              className="w-28 rounded-lg border border-slate-200 px-2 py-1.5 text-sm"
              value={draft.blog.wordsPerMinute ?? 280}
              onChange={(e) => {
                const n = Number(e.target.value);
                setDraft((d) => ({
                  ...d,
                  blog: { ...d.blog, wordsPerMinute: Number.isFinite(n) && n > 0 ? n : 280 },
                }));
              }}
            />
          </div>
        </AdminCard>

        <AdminCard title="公开页面" description="控制前台路由是否开放">
          <ul className="space-y-2">
            {PUBLIC_PAGE_KEYS.map((key) => (
              <li key={key} className="flex items-center gap-3">
                <input
                  type="checkbox"
                  id={`pp-${key}`}
                  checked={draft.publicPages[key]}
                  onChange={() => toggle(key)}
                />
                <label htmlFor={`pp-${key}`} className="text-sm">
                  /{key}
                </label>
              </li>
            ))}
          </ul>
        </AdminCard>
      </div>
    </div>
  );
}
