import { useState, useEffect, useCallback, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { getPublicArticle } from "@/api/articles";
import type { Article } from "@/api/articles";
import { PublicLayout } from "@/theme/layouts";
import PageHero from "@/components/feature/PageHero";

export default function BlogDetailPage() {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();

  const [article, setArticle] = useState<Article | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Lightbox state
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
  const contentRef = useRef<HTMLElement>(null);

  useEffect(() => {
    if (!slug) return;

    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await getPublicArticle(slug);
        setArticle(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load article");
      } finally {
        setLoading(false);
      }
    };

    load();
  }, [slug]);

  // Delegate click on images inside rendered content to open lightbox
  const handleContentClick = useCallback((e: React.MouseEvent) => {
    const target = e.target as HTMLElement;
    if (target.tagName === "IMG") {
      const src = (target as HTMLImageElement).src;
      if (src) setLightboxSrc(src);
    }
  }, []);

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return "";
    return new Date(dateStr).toLocaleDateString("zh-CN", {
      year: "numeric",
      month: "long",
      day: "numeric",
    });
  };

  const title = article?.zhTitle || article?.enTitle || "";
  const body = article?.zhBody || article?.enBody || "";

  return (
    <PublicLayout>
      <PageHero
        title={title || "Blog"}
        label="Blog"
        imageSrc={article?.coverImage || undefined}
      />
      <div className="max-w-3xl mx-auto px-4 py-12">
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-gray-600">Loading...</div>
          </div>
        ) : error || !article ? (
          <div className="text-center">
            <p className="text-red-600 mb-4">{error || "Article not found"}</p>
            <button
              onClick={() => navigate("/blog")}
              className="text-blue-600 hover:text-blue-800"
            >
              Back to Blog
            </button>
          </div>
        ) : (
          <>
            <div className="mb-8 flex items-center gap-3 text-sm text-gray-500 flex-wrap">
              <span>{formatDate(article.publishedAt || article.createdAt)}</span>
              {article.category && (
                <>
                  <span>&middot;</span>
                  <button
                    onClick={() => navigate(`/blog?category=${article.category!.slug}`)}
                    className="text-blue-600 hover:text-blue-800"
                  >
                    {article.category.zhName || article.category.enName}
                  </button>
                </>
              )}
              {article.tags && article.tags.length > 0 && (
                <>
                  <span>&middot;</span>
                  {article.tags.map((tag) => (
                    <button
                      key={tag.id}
                      onClick={() => navigate(`/blog?tag=${tag.slug}`)}
                      className="text-xs px-2.5 py-1 bg-gray-100 text-gray-600 rounded-full hover:bg-gray-200"
                    >
                      {tag.zhName || tag.enName}
                    </button>
                  ))}
                </>
              )}
            </div>

            <article
              ref={contentRef}
              className="tiptap ProseMirror prose prose-gray max-w-none article-public-view"
              dangerouslySetInnerHTML={{ __html: body }}
              onClick={handleContentClick}
            />
          </>
        )}
      </div>

      {/* Lightbox */}
      {lightboxSrc && (
        <div
          className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/70 cursor-zoom-out"
          onClick={() => setLightboxSrc(null)}
        >
          <img
            src={lightboxSrc}
            alt=""
            className="max-w-[90vw] max-h-[90vh] object-contain rounded-lg shadow-2xl"
          />
        </div>
      )}
    </PublicLayout>
  );
}
