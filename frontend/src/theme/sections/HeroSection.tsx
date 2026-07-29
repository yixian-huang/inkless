import type { SectionProps } from "../types";

export interface HeroSectionData {
  title?: string;
  subtitle?: string;
  label?: string;
  backgroundImage?: string;
  backgroundColor?: string;
}

export default function HeroSection({
  data,
  variant = "default",
}: SectionProps<HeroSectionData>) {
  const { title, subtitle, label, backgroundImage, backgroundColor } = data;
  const compact = variant === "compact";
  const solidColor = backgroundColor || (compact ? "#0f172a" : undefined);
  const useImage = !solidColor;
  const src = useImage ? (backgroundImage || "/images/hero-bg.png") : undefined;

  const heightClass = compact
    ? "relative min-h-[160px] sm:min-h-[200px] md:min-h-[220px] max-h-[320px]"
    : useImage
      ? "relative min-h-[280px] sm:min-h-[360px] md:min-h-[40vh] lg:min-h-[45vh] max-h-[600px]"
      : "relative min-h-[200px] sm:min-h-[300px] md:min-h-[35vh] lg:min-h-[40vh] max-h-[540px]";

  return (
    <div className={heightClass} data-variant={variant}>
      <div
        className="absolute inset-0"
        style={solidColor ? { backgroundColor: solidColor } : undefined}
      >
        {src && (
          <>
            <img
              src={src}
              alt="Hero Background"
              className="w-full h-full object-cover object-center"
            />
            <div className="absolute inset-0 bg-black/40" />
          </>
        )}
      </div>
      <div
        className={
          compact
            ? "absolute inset-0 z-10 flex items-center"
            : "absolute left-0 right-0 bottom-[20%] z-10"
        }
      >
        <div className="max-w-layout w-full mx-auto px-4 md:px-content xl:px-8">
          {label && (
            <p className="text-white text-sm sm:text-base mb-1">{label}</p>
          )}
          {title && (
            <h1
              className={
                compact
                  ? "text-white text-2xl md:text-3xl lg:text-4xl font-semibold tracking-tight"
                  : "text-white text-2xl md:text-3xl lg:text-4xl xl:text-5xl font-bold uppercase tracking-wide"
              }
            >
              {title}
              {subtitle && (
                <span
                  className={
                    compact
                      ? "block mt-2 text-base sm:text-lg md:text-xl font-normal text-white/85 normal-case tracking-normal"
                      : "block mt-1 sm:mt-2 text-base sm:text-xl md:text-2xl lg:text-3xl font-normal"
                  }
                >
                  {subtitle}
                </span>
              )}
            </h1>
          )}
        </div>
      </div>
    </div>
  );
}
