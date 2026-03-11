import type { SectionProps } from "../types";

export interface CompanyProfileSectionData {
  title?: string;
  description?: string;
}

export default function CompanyProfileSection({
  data,
}: SectionProps<CompanyProfileSectionData>) {
  const { title, description } = data;

  if (!title) return null;

  return (
    <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
      <div className="grid grid-cols-1 lg:grid-cols-12 items-center gap-8 lg:gap-12 xl:gap-16">
        <div className="lg:col-span-3 text-center">
          <h2 className="text-3xl md:text-4xl font-bold text-primary uppercase">
            {title}
          </h2>
        </div>
        {description && (
          <div className="lg:col-span-9">
            <p className="text-2xl md:text-3xl leading-relaxed text-primary-dark">
              {description}
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
