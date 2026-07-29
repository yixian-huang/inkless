import type { SectionProps } from "../types";

export interface StepItem {
  title?: string;
  description?: string;
}

export interface StepsSectionData {
  title?: string;
  steps?: StepItem[];
}

/** Numbered steps for guides (Host baseline — not theme-branded). */
export default function StepsSection({ data }: SectionProps<StepsSectionData>) {
  const { title, steps = [] } = data;
  const visible = steps.filter((s) => s.title || s.description);
  if (!title && visible.length === 0) return null;

  return (
    <div
      className="max-w-layout w-full mx-auto px-4 md:px-content xl:px-8"
      data-testid="steps-section"
    >
      {title ? (
        <h2 className="text-2xl md:text-3xl font-semibold tracking-tight text-on-surface font-heading mb-8">
          {title}
        </h2>
      ) : null}
      <ol className="grid grid-cols-1 md:grid-cols-3 gap-6 md:gap-8 list-none p-0 m-0">
        {visible.map((step, i) => (
          <li
            key={`${step.title?.slice(0, 24) ?? "step"}-${i}`}
            className="relative rounded-xl border border-border/70 bg-surface p-5 md:p-6"
          >
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary text-sm font-semibold text-white mb-4">
              {i + 1}
            </div>
            {step.title ? (
              <h3 className="text-lg font-semibold text-on-surface mb-2">
                {step.title}
              </h3>
            ) : null}
            {step.description ? (
              <p className="text-sm md:text-base text-on-surface-muted leading-relaxed">
                {step.description}
              </p>
            ) : null}
          </li>
        ))}
      </ol>
    </div>
  );
}
