import { EfShell } from "./shell";
import { asArray, asString, type SectionProps } from "./types";

export interface EfServiceItem {
  title?: string;
  summary?: string;
  href?: string;
}

export interface EfServiceIndexData {
  title?: string;
  intro?: string;
  items?: EfServiceItem[];
}

function padIndex(i: number): string {
  const n = i + 1;
  return n < 10 ? `0${n}` : String(n);
}

export default function EfServiceIndex({ data }: SectionProps<EfServiceIndexData>) {
  const title = asString(data.title);
  const intro = asString(data.intro);
  const items = asArray<EfServiceItem>(data.items);

  return (
    <EfShell>
      {(title || intro) && (
        <header className="mb-10 md:mb-14 max-w-2xl">
          {title ? (
            <h2 className="font-heading text-3xl md:text-4xl text-on-surface font-semibold mb-4">
              {title}
            </h2>
          ) : null}
          {intro ? (
            <p className="text-base md:text-lg text-on-surface-muted leading-relaxed">
              {intro}
            </p>
          ) : null}
        </header>
      )}

      <ol className="list-none m-0 p-0 border-t border-border">
        {items.map((item, i) => {
          const itemTitle = asString(item?.title);
          const summary = asString(item?.summary);
          const href = asString(item?.href);
          if (!itemTitle && !summary) return null;

          const row = (
            <div className="grid grid-cols-[auto_1fr] gap-x-6 md:gap-x-10 gap-y-2 py-6 md:py-8 items-baseline">
              <span className="font-heading text-sm md:text-base text-accent tabular-nums tracking-wider">
                {padIndex(i)}
              </span>
              <div>
                {itemTitle ? (
                  <h3 className="font-heading text-xl md:text-2xl text-on-surface font-semibold">
                    {itemTitle}
                  </h3>
                ) : null}
                {summary ? (
                  <p className="mt-2 text-sm md:text-base text-on-surface-muted leading-relaxed max-w-2xl">
                    {summary}
                  </p>
                ) : null}
              </div>
            </div>
          );

          return (
            <li key={i} className="border-b border-border">
              {href ? (
                <a
                  href={href}
                  className="block hover:bg-surface-alt/60 transition-colors -mx-2 px-2 rounded-sm"
                >
                  {row}
                </a>
              ) : (
                row
              )}
            </li>
          );
        })}
      </ol>
    </EfShell>
  );
}
