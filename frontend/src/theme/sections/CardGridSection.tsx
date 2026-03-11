import { useTranslation } from "react-i18next";
import type { SectionProps } from "../types";

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
}

const columnClasses: Record<number, string> = {
  2: "grid-cols-2 md:grid-cols-2",
  3: "grid-cols-2 md:grid-cols-3",
  4: "grid-cols-2 md:grid-cols-4",
};

export default function CardGridSection({ data }: SectionProps<CardGridSectionData>) {
  const { i18n } = useTranslation("common");
  const isZh = i18n.language === "zh" || i18n.language.startsWith("zh");

  const { title, cards, columns = 4 } = data;
  const gridClass = columnClasses[columns] || columnClasses[4];

  return (
    <div className="max-w-layout w-full mx-auto px-4 md:px-content xl:px-8">
      {title && (
        <div className="flex items-center mb-8 sm:mb-12">
          <div className="w-5 h-5 sm:w-[26px] sm:h-[26px] bg-accent mr-2 sm:mr-3 flex-shrink-0 rounded-sm" />
          <h2 className="text-2xl sm:text-3xl md:text-4xl font-bold text-on-surface truncate min-w-0">
            {title}
          </h2>
          <span className="ml-1 sm:ml-2 text-xl sm:text-2xl text-accent flex-shrink-0 cursor-pointer">
            &gt;
          </span>
        </div>
      )}
      {cards && cards.length > 0 && (
        <div className={`grid ${gridClass} gap-0`}>
          {cards.map((card, index) => (
            <div
              key={index}
              className="group relative w-full sm:aspect-auto overflow-hidden"
            >
              <img
                src={card.image || `/images/advantage-${index + 1}.png`}
                alt={card.title || `Card ${index + 1}`}
                className="w-full h-full object-cover object-top transition-transform duration-300 group-hover:scale-105"
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
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
