import { http } from "@/api/http";

export type ArticleMetaMode = "fill_empty" | "rewrite";

export type ArticleMetaField =
  | "titles"
  | "slug"
  | "seo"
  | "meta"
  | "tags"
  | "excerpts";

export interface ArticleMetaExisting {
  slug?: string;
  zhSeoTitle?: string;
  enSeoTitle?: string;
  zhMetaDescription?: string;
  enMetaDescription?: string;
  zhTitle?: string;
  enTitle?: string;
}

export interface ArticleMetaRequest {
  sourceLang: "zh" | "en";
  zhTitle?: string;
  enTitle?: string;
  zhBody?: string;
  enBody?: string;
  existing?: ArticleMetaExisting;
  fields?: ArticleMetaField[];
  mode?: ArticleMetaMode;
  titleCount?: number;
  existingTags?: string[];
  /** When true (e.g. published article), server skips slug suggestions. */
  slugLocked?: boolean;
}

export interface ArticleMetaSuggested {
  zhTitle?: string;
  enTitle?: string;
  slug?: string;
  zhSeoTitle?: string;
  enSeoTitle?: string;
  zhMetaDescription?: string;
  enMetaDescription?: string;
  zhExcerpt?: string;
  enExcerpt?: string;
  tags?: string[];
}

export interface ArticleMetaWarning {
  code: string;
  field?: string;
  message: string;
  severity: "warn" | "info" | string;
}

export interface ArticleMetaResponse {
  candidates: {
    zhTitles?: string[];
    enTitles?: string[];
  };
  suggested: ArticleMetaSuggested;
  skipped: string[];
  /** Phase 1.5 soft quality signals from the server */
  warnings?: ArticleMetaWarning[];
  model?: string;
  usage?: {
    prompt_tokens?: number;
    output_tokens?: number;
  };
}

export async function generateArticleMeta(
  req: ArticleMetaRequest,
): Promise<ArticleMetaResponse> {
  const response = await http.post<ArticleMetaResponse>("/admin/ai/article-meta", req, {});
  return response.data;
}
