import { useState } from "react";
import type { SectionProps } from "../types";

export interface ExpertData {
  id: string;
  name?: string;
  title?: string;
  image?: string;
  bio?: string;
}

export interface TeamGridSectionData {
  sectionTitle?: string;
  experts?: ExpertData[];
}

export default function TeamGridSection({ data }: SectionProps<TeamGridSectionData>) {
  const { sectionTitle, experts = [] } = data;
  const [activeId, setActiveId] = useState<string>(experts[0]?.id || "");

  const activeExpert = experts.find((e) => e.id === activeId) || experts[0];
  const bioParagraphs = activeExpert?.bio
    ? activeExpert.bio.split(/\n\n+/).filter(Boolean)
    : [];

  return (
    <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
      {sectionTitle && (
        <div className="flex items-center mb-10 md:mb-12">
          <div className="w-[26px] h-[26px] bg-accent mr-3 flex-shrink-0 rounded-full" />
          <h2 className="text-2xl md:text-3xl font-bold text-on-surface">
            {sectionTitle}
          </h2>
        </div>
      )}

      {experts.length > 0 && (
        <div className="grid grid-cols-2 gap-8 md:gap-12 xl:gap-14 max-w-2xl mx-auto mb-12 md:mb-16">
          {experts.map((expert) => (
            <div key={expert.id} className="flex flex-col items-center text-center">
              <div className="w-32 h-32 md:w-40 md:h-40 xl:w-44 xl:h-44 rounded-full overflow-hidden border-2 border-border flex-shrink-0 mb-3">
                <img
                  src={expert.image || `/images/expert/${expert.id}.png`}
                  alt={expert.name || expert.id}
                  className="w-full h-full object-cover object-top"
                />
              </div>
              {expert.name && (
                <h3 className="text-lg md:text-xl font-bold text-primary">
                  {expert.name}
                </h3>
              )}
              {expert.title && (
                <p className="text-sm text-on-surface-muted">{expert.title}</p>
              )}
            </div>
          ))}
        </div>
      )}

      {experts.length > 0 && activeExpert && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 lg:gap-8 xl:gap-10 items-start">
          <div className="lg:col-span-3 flex lg:flex-col gap-2">
            {experts.map((expert) => (
              <button
                key={expert.id}
                type="button"
                onClick={() => setActiveId(expert.id)}
                className={`px-4 py-3 text-left rounded-md transition-colors cursor-pointer whitespace-nowrap ${
                  activeId === expert.id
                    ? "bg-primary text-white"
                    : "bg-surface-alt text-on-surface-muted hover:bg-surface-alt/80"
                }`}
              >
                {expert.name || expert.id}
              </button>
            ))}
          </div>
          <div className="lg:col-span-9 bg-surface rounded-lg border border-border p-6 md:p-8">
            {bioParagraphs.map((para, i) => (
              <p
                key={i}
                className="text-base text-on-surface-muted leading-relaxed mb-4 last:mb-0"
              >
                {para}
              </p>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
