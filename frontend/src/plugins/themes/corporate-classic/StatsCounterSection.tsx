import type { SectionProps } from "@/theme/types";

interface StatItem {
  value?: string;
  label?: string;
}

interface StatsCounterData {
  title?: string;
  stats?: StatItem[];
}

export default function StatsCounterSection({ data }: SectionProps<StatsCounterData>) {
  const { title, stats = [] } = data;

  return (
    <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
      {title && (
        <h2 className="text-2xl md:text-3xl font-bold text-on-surface text-center mb-10">
          {title}
        </h2>
      )}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
        {stats.map((stat, i) => (
          <div key={i} className="text-center">
            <div className="text-3xl md:text-4xl font-bold text-primary mb-2">
              {stat.value}
            </div>
            <div className="text-sm text-on-surface-muted">
              {stat.label}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
