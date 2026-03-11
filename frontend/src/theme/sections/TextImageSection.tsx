import type { SectionProps } from "../types";

export interface TextImageSectionData {
  title?: string;
  description?: string;
  image?: string;
  imagePosition?: "left" | "right";
}

export default function TextImageSection({ data }: SectionProps<TextImageSectionData>) {
  const { title, description, image, imagePosition = "left" } = data;
  const isImageLeft = imagePosition === "left";

  return (
    <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 mb-12">
      <div className="grid grid-cols-1 lg:grid-cols-2 items-center">
        {isImageLeft ? (
          <>
            <div className="w-full aspect-[4/3] max-h-[400px] lg:max-h-none overflow-hidden rounded-lg order-2 lg:order-1">
              <img
                src={image || "/images/placeholder.png"}
                alt={title || "Section image"}
                className="w-full h-full object-cover object-center"
              />
            </div>
            <div className="w-full h-full py-12 px-10 md:px-16 xl:px-20 order-1 lg:order-2">
              {title && (
                <h2 className="text-3xl md:text-4xl font-bold text-primary mb-4">
                  {title}
                </h2>
              )}
              {description && (
                <p className="text-2xl md:text-3xl leading-relaxed text-primary-dark">
                  {description}
                </p>
              )}
            </div>
          </>
        ) : (
          <>
            <div className="w-full h-full py-12 px-10 md:px-16 xl:px-20">
              {title && (
                <h2 className="text-3xl md:text-4xl font-bold text-primary mb-4">
                  {title}
                </h2>
              )}
              {description && (
                <p className="text-2xl md:text-3xl leading-relaxed text-primary-dark">
                  {description}
                </p>
              )}
            </div>
            <div className="w-full aspect-[4/3] max-h-[400px] lg:max-h-none overflow-hidden rounded-lg">
              <img
                src={image || "/images/placeholder.png"}
                alt={title || "Section image"}
                className="w-full h-full object-cover object-center"
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
}
