import { useTranslation } from "react-i18next";
import type { SectionProps } from "../types";
import { resolveLocale } from "@/utils/locale";

export interface CardGridCard {
  title?: string;
  titleEn?: string;
  description?: string;
  image?: string;
}

export interface CardGridSectionData {
  title?: string;
  cards?: CardGridCard[];
  columns?: 2 | 3 | 4;
  /** When true (default), cards without image use text-only layout instead of stock photos */
  preferTextCards?: boolean;
}

const columnClasses: Record<number, string> = {
  2: "grid-cols-1 sm:grid-cols-2",
  3: "grid-cols-1 sm:grid-cols-2 md:grid-cols-3",
  4: "grid-cols-1 sm:grid-cols-2 md:grid-cols-4",
};

function hasAnyImage(cards: CardGridCard[] | undefined): boolean {
  return Boolean(cards?.some((c) => typeof c.image === "string" && c.image.trim()));
}

export default function CardGridSection({ data }: SectionProps<CardGridSectionData>) {
  const { i18n } = useTranslation("common");
  const isZh = resolveLocale(i18n.language) === "zh";

  const { title, cards, columns = 3, preferTextCards = true } = data;
  const gridClass = columnClasses[columns] || columnClasses[3];
  // Default to text cards when no image is set (avoid stock photo placeholders).
  // Set preferTextCards=false to force legacy image grid with fallbacks.
  const textOnly = preferTextCards !== false && !hasAnyImage(cards);
  return (
    <div
      className="max-w-layout w-full mx-auto px-4 md:px-content xl:px-8"
      data-testid="card-grid-section"
      data-variant={textOnly ? "text" : "image"}
    >
      {title && (
        <div className="flex items-center mb-8 sm:mb-10">
          <div className="w-5 h-5 sm:w-[26px] sm:h-[26px] bg-accent mr-2 sm:mr-3 flex-shrink-0 rounded-sm" />
          <h2 className="text-2xl sm:text-3xl md:text-4xl font-bold text-on-surface truncate min-w-0">
            {title}
          </h2>
        </div>
      )}
      {cards && cards.length > 0 && textOnly && (
        <div className={`grid ${gridClass} gap-4 md:gap-6`}>
          {cards.map((card, index) => (
            <div
              key={card.title || String(index)}
              className="rounded-xl border border-border/70 bg-surface p-5 md:p-6 shadow-sm hover:border-primary/30 transition-colors"
            >
              {card.title && (
                <h3 className="text-lg md:text-xl font-semibold text-on-surface mb-2">
                  {card.title}
                </h3>
              )}
              {isZh && card.titleEn && (
                <p className="text-xs text-on-surface-muted mb-2">{card.titleEn}</p>
              )}
              {card.description && (
                <p className="text-sm md:text-base text-on-surface-muted leading-relaxed">
                  {card.description}
                </p>
              )}
            </div>
          ))}
        </div>
      )}
      {cards && cards.length > 0 && !textOnly && (
        <div className={`grid ${gridClass} gap-0`}>
          {cards.map((card, index) => {
            const src =
              (typeof card.image === "string" && card.image.trim()) ||
              `/images/advantage-${(index % 4) + 1}.png`;
            return (
              <div
                key={card.title || String(index)}
                className="group relative w-full min-h-[180px] sm:min-h-[220px] overflow-hidden"
              >
                <img
                  src={src}
                  alt={card.title || `Card ${index + 1}`}
                  className="absolute inset-0 w-full h-full object-cover object-top transition-transform duration-300 group-hover:scale-105"
                />
                <div className="absolute inset-0 bg-surface-alt/95 opacity-0 group-hover:opacity-100 transition-opacity duration-300 flex flex-col justify-center items-center p-4 sm:p-5 text-center">
                  {card.title && (
                    <h3 className="text-primary text-lg sm:text-xl md:text-2xl font-bold mb-3 w-full">
                      {card.title}
                    </h3>
                  )}
                  {isZh && card.titleEn && (
                    <p className="text-primary/80 text-xs sm:text-sm mb-2 w-full">
                      {card.titleEn}
                    </p>
                  )}
                  {card.description && (
                    <p className="text-on-surface text-sm sm:text-base font-normal leading-loose max-w-[92%] sm:max-w-xs line-clamp-4 sm:line-clamp-5 text-left">
                      {card.description}
                    </p>
                  )}
                </div>
                {/* Always-visible title bar for accessibility when not hovering */}
                {card.title && (
                  <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/70 to-transparent p-3 md:p-4 group-hover:opacity-0 transition-opacity">
                    <p className="text-white text-sm md:text-base font-medium truncate">
                      {card.title}
                    </p>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
