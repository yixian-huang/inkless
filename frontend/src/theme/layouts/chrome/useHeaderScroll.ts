import { useEffect, useState } from "react";

/** Scroll state for corporate hero overlay header (light text until hero scrolls past). */
export function useHeaderScroll(enabled: boolean): boolean {
  const [isScrolled, setIsScrolled] = useState(false);

  useEffect(() => {
    if (!enabled) return;

    const handleScroll = () => {
      const heroEl = document.querySelector("[data-page-hero]");
      if (heroEl) {
        const rect = heroEl.getBoundingClientRect();
        setIsScrolled(rect.bottom <= 80);
      } else {
        setIsScrolled(window.scrollY > 50);
      }
    };

    window.addEventListener("scroll", handleScroll, { passive: true });
    handleScroll();
    return () => window.removeEventListener("scroll", handleScroll);
  }, [enabled]);

  return enabled ? isScrolled : false;
}
