import type { SectionProps } from "../types";

export interface RichTextSectionData {
  content?: string;
  alignment?: "left" | "center";
}

export default function RichTextSection({ data }: SectionProps<RichTextSectionData>) {
  const { content, alignment = "left" } = data;

  if (!content) return null;

  const paragraphs = content.split(/\n\n+/).filter(Boolean);
  const alignClass = alignment === "center" ? "text-center" : "text-left";

  return (
    <div className={`max-w-layout mx-auto px-4 md:px-content xl:px-8 ${alignClass}`}>
      {paragraphs.map((para, i) => (
        <p
          key={i}
          className="text-base text-on-surface-muted leading-relaxed mb-4 last:mb-0"
        >
          {para}
        </p>
      ))}
    </div>
  );
}
