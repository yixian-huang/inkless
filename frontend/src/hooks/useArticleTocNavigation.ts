import { useCallback, useEffect, useState } from "react";
import type { RefObject } from "react";
import type { TocHeading } from "@/utils/articleToc";

export function useActiveHeading(contentRef: RefObject<HTMLElement | null>, headings: TocHeading[]) {
  const [activeId, setActiveId] = useState<string | null>(headings[0]?.id ?? null);

  useEffect(() => {
    const root = contentRef.current;
    if (!root || headings.length === 0) return;

    const elements = headings
      .map((heading) => document.getElementById(heading.id))
      .filter((element): element is HTMLElement => element != null);

    if (elements.length === 0) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible.length > 0 && visible[0].target.id) {
          setActiveId(visible[0].target.id);
        }
      },
      { rootMargin: "-20% 0px -55% 0px", threshold: 0 },
    );

    elements.forEach((element) => observer.observe(element));
    return () => observer.disconnect();
  }, [contentRef, headings]);

  return activeId;
}

export function useTocScroll() {
  return useCallback((id: string) => {
    const element = document.getElementById(id);
    if (!element) return;
    element.scrollIntoView({ behavior: "smooth", block: "start" });
    history.replaceState(null, "", `#${id}`);
  }, []);
}
