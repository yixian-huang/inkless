import { useEffect, useState } from "react";
import {
  fetchAdminGlobalConfig,
  putAdminGlobalConfigDraft,
  publishAdminGlobalConfig,
} from "@/api/globalConfig";
import type { SiteConfigGlobal } from "@/types/siteConfig";

export default function AdminSiteConfigPage() {
  const [draftJson, setDraftJson] = useState("");
  const [draftVersion, setDraftVersion] = useState(0);
  const [publishedVersion, setPublishedVersion] = useState(0);
  const [status, setStatus] = useState<string>("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchAdminGlobalConfig()
      .then((s) => {
        setDraftJson(JSON.stringify(s.draftConfig, null, 2));
        setDraftVersion(s.draftVersion);
        setPublishedVersion(s.publishedVersion);
      })
      .catch((e: Error) => setStatus("Load failed: " + e.message))
      .finally(() => setLoading(false));
  }, []);

  async function saveDraft() {
    setStatus("");
    let parsed: SiteConfigGlobal;
    try {
      parsed = JSON.parse(draftJson) as SiteConfigGlobal;
    } catch (e) {
      setStatus("JSON parse error: " + (e as Error).message);
      return;
    }
    try {
      const r = await putAdminGlobalConfigDraft(parsed, draftVersion);
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
      setPublishedVersion(r.publishedVersion);
      setStatus("Published (v" + r.publishedVersion + ")");
    } catch (e) {
      setStatus("Publish failed: " + (e as Error).message);
    }
  }

  if (loading) return <div className="p-4">Loading…</div>;

  return (
    <div className="p-4 max-w-4xl">
      <h1 className="text-xl font-semibold mb-4">Site Config (raw JSON)</h1>
      <p className="text-sm text-gray-500 mb-2">
        Draft v{draftVersion} · Published v{publishedVersion}
      </p>
      <textarea
        value={draftJson}
        onChange={(e) => setDraftJson(e.target.value)}
        className="w-full h-[60vh] font-mono text-sm border rounded p-2"
      />
      <div className="mt-4 flex gap-2 items-center">
        <button
          onClick={saveDraft}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          Save Draft
        </button>
        <button
          onClick={publish}
          className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
        >
          Publish
        </button>
        {status && <span className="text-sm text-gray-700">{status}</span>}
      </div>
    </div>
  );
}
