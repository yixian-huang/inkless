import type { SectionProps } from "../types";

export interface ChecklistCategory {
  title?: string;
  items?: string[];
}

export interface ChecklistSectionData {
  categories?: ChecklistCategory[];
}

export default function ChecklistSection({ data }: SectionProps<ChecklistSectionData>) {
  const { categories = [] } = data;

  return (
    <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
      <div className="space-y-10 md:space-y-14 xl:space-y-16">
        {categories.map((category, index) => {
          const items = category.items || [];
          if (!category.title && items.length === 0) return null;
          return (
            <div key={index}>
              {category.title && (
                <h2 className="text-xl md:text-2xl font-bold text-primary mb-4">
                  {category.title}
                </h2>
              )}
              {items.length > 0 && (
                <ul className="space-y-2">
                  {items.map((item, i) => (
                    <li
                      key={i}
                      className="flex items-start gap-2 text-base text-on-surface-muted leading-relaxed"
                    >
                      <span className="text-primary flex-shrink-0" aria-hidden>
                        &#x2705;
                      </span>
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
