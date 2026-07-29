import type { SectionProps } from "../types";
import ArticleTypographyRoot from "@/components/blog/ArticleTypographyRoot";
import ArticlePostBody from "@/components/blog/ArticlePostBody";

export interface RichTextSectionData {
  /** Plain text or HTML (TipTap / seed content). May be bilingual before SectionRenderer resolve. */
  content?: string;
  alignment?: "left" | "center";
}

function looksLikeHtml(value: string): boolean {
  return /<[a-zA-Z!/?]/.test(value);
}

/**
 * Public rich-text block for unified pages.
 * HTML path reuses article typography + ArticlePostBody (prose, code copy, mermaid).
 */
export default function RichTextSection({
  data,
  settings,
  pageLayout,
}: SectionProps<RichTextSectionData> & { pageLayout?: string }) {
  const { content, alignment = "left" } = data;

  if (!content || typeof content !== "string") return null;

  const alignClass = alignment === "center" ? "text-center" : "text-left";
  // When page already provides reading column, avoid a second max-width shell
  const pageIsReading = pageLayout === "reading";
  const maxWidth = settings?.maxWidth;
  const useInnerLayoutShell =
    !pageIsReading && maxWidth !== "reading" && maxWidth !== "full";

  const shellClass = [
    useInnerLayoutShell ? "max-w-layout mx-auto px-4 md:px-content xl:px-8" : "w-full",
    // reading page / reading section: no extra horizontal pad if parent padded
    pageIsReading || maxWidth === "reading" ? "" : "",
    alignClass,
  ]
    .filter(Boolean)
    .join(" ");

  if (looksLikeHtml(content)) {
    return (
      <div className={shellClass} data-testid="rich-text-section">
        <ArticleTypographyRoot mode="reading" className="article-public-view">
          <ArticlePostBody html={content} />
        </ArticleTypographyRoot>
      </div>
    );
  }

  const paragraphs = content.split(/\n\n+/).filter(Boolean);
  return (
    <div className={shellClass} data-testid="rich-text-section">
      <ArticleTypographyRoot mode="reading" className="article-public-view">
        <div className="tiptap ProseMirror max-w-none article-public-view">
          {paragraphs.map((para, i) => (
            <p key={i} className="whitespace-pre-wrap">
              {para}
            </p>
          ))}
        </div>
      </ArticleTypographyRoot>
    </div>
  );
}
