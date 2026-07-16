import { useMemo } from "react";
import { useIsReadingLayout } from "@/plugins/hooks";
import { useArticleTocEnabled } from "@/hooks/useArticleTocEnabled";
import { useActiveHeading, useTocScroll } from "@/hooks/useArticleTocNavigation";
import { buildArticleToc, type TocLayout } from "@/utils/articleToc";
import { ArticleTocInline, ArticleTocSidebar, ArticleTocLayout } from "@/components/blog/ArticleToc";
import ArticlePostBody from "@/components/blog/ArticlePostBody";
import type { RefObject } from "react";

interface ArticlePostWithTocProps {
  html: string;
  contentRef: RefObject<HTMLElement | null>;
  onClick?: (e: React.MouseEvent) => void;
}

export default function ArticlePostWithToc({ html, contentRef, onClick }: ArticlePostWithTocProps) {
  const isReading = useIsReadingLayout();
  const tocEnabled = useArticleTocEnabled();

  const { headings, htmlWithIds, layout } = useMemo(() => {
    if (!isReading || !tocEnabled || !html.trim()) {
      return { headings: [], htmlWithIds: html, layout: "none" as TocLayout };
    }
    return buildArticleToc(html);
  }, [html, isReading, tocEnabled]);

  const activeId = useActiveHeading(contentRef, headings);
  const scrollTo = useTocScroll();

  return (
    <>
      <ArticleTocInline
        headings={headings}
        layout={layout}
        activeId={activeId}
        onSelect={scrollTo}
      />
      <ArticleTocLayout
        layout={layout}
        sidebar={
          <ArticleTocSidebar
            headings={headings}
            layout={layout}
            activeId={activeId}
            onSelect={scrollTo}
          />
        }
      >
        <ArticlePostBody html={htmlWithIds} contentRef={contentRef} onClick={onClick} />
      </ArticleTocLayout>
    </>
  );
}
