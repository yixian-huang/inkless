import { useCallback, useState } from "react";
import type { Article } from "@/api/articles";
import type { ArticleDraftSnapshot } from "../VersionHistoryPanel";

export type ArticleStatus = "draft" | "published" | "scheduled";

/**
 * All serializable article fields + helpers to hydrate from API / snapshot.
 */
export function useArticleFormState() {
  const [zhTitle, setZhTitle] = useState("");
  const [enTitle, setEnTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [selectedCategoryIds, setSelectedCategoryIds] = useState<number[]>([]);
  const [selectedTagIds, setSelectedTagIds] = useState<number[]>([]);
  const [coverImage, setCoverImage] = useState("");
  const [zhBody, setZhBody] = useState("");
  const [enBody, setEnBody] = useState("");
  const [zhSeoTitle, setZhSeoTitle] = useState("");
  const [enSeoTitle, setEnSeoTitle] = useState("");
  const [zhMetaDescription, setZhMetaDescription] = useState("");
  const [enMetaDescription, setEnMetaDescription] = useState("");
  const [ogImage, setOgImage] = useState("");
  const [author, setAuthor] = useState("");
  const [autoSummary, setAutoSummary] = useState(false);
  const [allowComments, setAllowComments] = useState(true);
  const [pinned, setPinned] = useState(false);
  const [visibility, setVisibility] = useState("public");
  const [metadata, setMetadata] = useState<Record<string, unknown>>({});
  const [articleCreatedAt, setArticleCreatedAt] = useState<string | null>(null);
  const [articlePublishedAt, setArticlePublishedAt] = useState<string | null>(null);
  const [articleStatus, setArticleStatus] = useState<ArticleStatus>("draft");

  const hydrateFromArticle = useCallback((article: Article) => {
    setZhTitle(article.zhTitle || "");
    setEnTitle(article.enTitle || "");
    setSlug(article.slug || "");
    if (article.categoryIds?.length) setSelectedCategoryIds(article.categoryIds);
    else if (article.categories?.length) setSelectedCategoryIds(article.categories.map((c) => c.id));
    else if (article.categoryId) setSelectedCategoryIds([article.categoryId]);
    else setSelectedCategoryIds([]);
    setSelectedTagIds(article.tags?.map((t) => t.id) || []);
    setCoverImage(article.coverImage || "");
    setZhBody(article.zhBody || "");
    setEnBody(article.enBody || "");
    setZhSeoTitle(article.zhSeoTitle || "");
    setEnSeoTitle(article.enSeoTitle || "");
    setZhMetaDescription(article.zhMetaDescription || "");
    setEnMetaDescription(article.enMetaDescription || "");
    setOgImage(article.ogImage || "");
    setAuthor(article.author || "");
    setAutoSummary(article.autoSummary || false);
    setAllowComments(article.allowComments !== false);
    setPinned(article.pinned || false);
    setVisibility(article.visibility || "public");
    setMetadata(article.metadata || {});
    setArticleCreatedAt(article.createdAt || null);
    setArticlePublishedAt(article.publishedAt || null);
    setArticleStatus(article.status || "draft");
  }, []);

  const hydrateFromSnapshot = useCallback((snapshot: ArticleDraftSnapshot, fallback: { slug: string; author: string }) => {
    setZhTitle(typeof snapshot.zhTitle === "string" ? snapshot.zhTitle : "");
    setEnTitle(typeof snapshot.enTitle === "string" ? snapshot.enTitle : "");
    setSlug(typeof snapshot.slug === "string" ? snapshot.slug : fallback.slug);
    setCoverImage(typeof snapshot.coverImage === "string" ? snapshot.coverImage : "");
    setAuthor(typeof snapshot.author === "string" ? snapshot.author : fallback.author);
    if (typeof snapshot.zhSeoTitle === "string") setZhSeoTitle(snapshot.zhSeoTitle);
    if (typeof snapshot.enSeoTitle === "string") setEnSeoTitle(snapshot.enSeoTitle);
    if (typeof snapshot.zhMetaDescription === "string") setZhMetaDescription(snapshot.zhMetaDescription);
    if (typeof snapshot.enMetaDescription === "string") setEnMetaDescription(snapshot.enMetaDescription);
    if (typeof snapshot.ogImage === "string") setOgImage(snapshot.ogImage);
    const zh = typeof snapshot.zhBody === "string" ? snapshot.zhBody : "";
    const en = typeof snapshot.enBody === "string" ? snapshot.enBody : "";
    setZhBody(zh);
    setEnBody(en);
    return { zhBody: zh, enBody: en };
  }, []);

  const toggleCategory = useCallback((catId: number) => {
    setSelectedCategoryIds((prev) =>
      prev.includes(catId) ? prev.filter((i) => i !== catId) : [...prev, catId],
    );
  }, []);

  const toggleTag = useCallback((tagId: number) => {
    setSelectedTagIds((prev) =>
      prev.includes(tagId) ? prev.filter((i) => i !== tagId) : [...prev, tagId],
    );
  }, []);

  return {
    zhTitle, setZhTitle,
    enTitle, setEnTitle,
    slug, setSlug,
    selectedCategoryIds, setSelectedCategoryIds, toggleCategory,
    selectedTagIds, setSelectedTagIds, toggleTag,
    coverImage, setCoverImage,
    zhBody, setZhBody,
    enBody, setEnBody,
    zhSeoTitle, setZhSeoTitle,
    enSeoTitle, setEnSeoTitle,
    zhMetaDescription, setZhMetaDescription,
    enMetaDescription, setEnMetaDescription,
    ogImage, setOgImage,
    author, setAuthor,
    autoSummary, setAutoSummary,
    allowComments, setAllowComments,
    pinned, setPinned,
    visibility, setVisibility,
    metadata, setMetadata,
    articleCreatedAt, setArticleCreatedAt,
    articlePublishedAt, setArticlePublishedAt,
    articleStatus, setArticleStatus,
    hydrateFromArticle,
    hydrateFromSnapshot,
  };
}
