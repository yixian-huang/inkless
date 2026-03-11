import type { SectionProps } from "../types";

export interface HeroSectionData {
  title?: string;
  subtitle?: string;
  label?: string;
  backgroundImage?: string;
  backgroundColor?: string;
}

export default function HeroSection({ data }: SectionProps<HeroSectionData>) {
  const { title, subtitle, label, backgroundImage, backgroundColor } = data;
  const useImage = !backgroundColor;
  const src = useImage ? (backgroundImage || "/images/hero-bg.png") : undefined;

  return (
    <section
      className={
        useImage
          ? "relative min-h-[280px] sm:min-h-[360px] md:min-h-[40vh] lg:min-h-[45vh] max-h-[600px]"
          : "relative min-h-[200px] sm:min-h-[300px] md:min-h-[35vh] lg:min-h-[40vh] max-h-[540px]"
      }
    >
      <div
        className="absolute inset-0"
        style={backgroundColor ? { backgroundColor } : undefined}
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
      <div className="absolute left-0 right-0 bottom-[20%] z-10">
        <div className="max-w-layout w-full mx-auto px-4 md:px-content xl:px-8">
          {label && (
            <p className="text-white text-sm sm:text-base mb-1">{label}</p>
          )}
          {title && (
            <h1 className="text-white text-2xl md:text-3xl lg:text-4xl xl:text-5xl font-bold uppercase tracking-wide">
              {title}
              {subtitle && (
                <span className="block mt-1 sm:mt-2 text-base sm:text-xl md:text-2xl lg:text-3xl font-normal">
                  {subtitle}
                </span>
              )}
            </h1>
          )}
        </div>
      </div>
    </section>
  );
}
