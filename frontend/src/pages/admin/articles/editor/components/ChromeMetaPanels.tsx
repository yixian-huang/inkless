import type { Category, Tag } from "@/api/articles";
import ArticleForm from "../ArticleForm";
import { SeoFieldsPanel, AdvancedSettingsPanel } from "../SeoFields";
import type { EditorMetaPanel } from "../hooks/useEditorShell";

/** Fields + setters for basic / SEO / advanced meta panels. */
export type ChromeMetaFormProps = {
  metaPanel: EditorMetaPanel;
  zenMode: boolean;
  // basic
  slug: string;
  setSlug: (v: string) => void;
  author: string;
  setAuthor: (v: string) => void;
  coverImage: string;
  setCoverImage: (v: string) => void;
  showCoverPicker: boolean;
  setShowCoverPicker: (v: boolean) => void;
  categories: Category[];
  selectedCategoryIds: number[];
  onToggleCategory: (id: number) => void;
  tags: Tag[];
  selectedTagIds: number[];
  onToggleTag: (id: number) => void;
  // seo
  zhSeoTitle: string;
  setZhSeoTitle: (v: string) => void;
  enSeoTitle: string;
  setEnSeoTitle: (v: string) => void;
  zhMetaDescription: string;
  setZhMetaDescription: (v: string) => void;
  enMetaDescription: string;
  setEnMetaDescription: (v: string) => void;
  ogImage: string;
  setOgImage: (v: string) => void;
  // advanced
  visibility: string;
  setVisibility: (v: string) => void;
  autoSummary: boolean;
  setAutoSummary: (v: boolean) => void;
  allowComments: boolean;
  setAllowComments: (v: boolean) => void;
  pinned: boolean;
  setPinned: (v: boolean) => void;
  metadata: Record<string, unknown>;
  setMetadata: (v: Record<string, unknown>) => void;
};

export function ChromeMetaPanels(p: ChromeMetaFormProps) {
  if (p.zenMode || !p.metaPanel) return null;

  if (p.metaPanel === "basic") {
    return (
      <ArticleForm
        slug={p.slug}
        setSlug={p.setSlug}
        author={p.author}
        setAuthor={p.setAuthor}
        coverImage={p.coverImage}
        setCoverImage={p.setCoverImage}
        showCoverPicker={p.showCoverPicker}
        setShowCoverPicker={p.setShowCoverPicker}
        categories={p.categories}
        selectedCategoryIds={p.selectedCategoryIds}
        toggleCategory={p.onToggleCategory}
        tags={p.tags}
        selectedTagIds={p.selectedTagIds}
        toggleTag={p.onToggleTag}
      />
    );
  }

  if (p.metaPanel === "seo") {
    return (
      <SeoFieldsPanel
        zhSeoTitle={p.zhSeoTitle}
        setZhSeoTitle={p.setZhSeoTitle}
        enSeoTitle={p.enSeoTitle}
        setEnSeoTitle={p.setEnSeoTitle}
        zhMetaDescription={p.zhMetaDescription}
        setZhMetaDescription={p.setZhMetaDescription}
        enMetaDescription={p.enMetaDescription}
        setEnMetaDescription={p.setEnMetaDescription}
        ogImage={p.ogImage}
        setOgImage={p.setOgImage}
      />
    );
  }

  return (
    <AdvancedSettingsPanel
      visibility={p.visibility}
      setVisibility={p.setVisibility}
      autoSummary={p.autoSummary}
      setAutoSummary={p.setAutoSummary}
      allowComments={p.allowComments}
      setAllowComments={p.setAllowComments}
      pinned={p.pinned}
      setPinned={p.setPinned}
      metadata={p.metadata}
      setMetadata={p.setMetadata}
    />
  );
}
