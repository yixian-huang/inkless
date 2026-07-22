import { EfShell } from "./shell";
import { asString, type SectionProps } from "./types";

export interface EfFeatureSplitData {
  title?: string;
  body?: string;
  image?: string;
  imageSide?: "left" | "right" | string;
  caption?: string;
}

export default function EfFeatureSplit({ data }: SectionProps<EfFeatureSplitData>) {
  const title = asString(data.title);
  const body = asString(data.body);
  const image = asString(data.image);
  const caption = asString(data.caption);
  const imageSide = asString(data.imageSide, "left") || "left";
  const imageLeft = imageSide !== "right";

  const imageBlock = (
    <figure className="w-full">
      <div className="aspect-[5/4] overflow-hidden rounded-sm bg-surface-alt">
        {image ? (
          <img
            src={image}
            alt={title || caption || ""}
            className="w-full h-full object-cover object-center"
          />
        ) : null}
      </div>
      {caption ? (
        <figcaption className="mt-3 text-sm text-on-surface-muted">{caption}</figcaption>
      ) : null}
    </figure>
  );

  const textBlock = (
    <div className="flex flex-col justify-center">
      {title ? (
        <h2 className="font-heading text-2xl md:text-3xl lg:text-4xl text-on-surface font-semibold leading-snug mb-5">
          {title}
        </h2>
      ) : null}
      {body
        ? body.split(/\n+/).filter(Boolean).map((para, i) => (
            <p
              key={i}
              className="text-base md:text-lg text-on-surface-muted leading-relaxed mb-4 last:mb-0"
            >
              {para}
            </p>
          ))
        : null}
    </div>
  );

  return (
    <EfShell>
      <div
        className={`grid grid-cols-1 md:grid-cols-12 gap-8 md:gap-12 items-center ${
          imageLeft ? "" : ""
        }`}
      >
        <div className={`md:col-span-5 ${imageLeft ? "order-1" : "order-1 md:order-2"}`}>
          {imageBlock}
        </div>
        <div className={`md:col-span-7 ${imageLeft ? "order-2" : "order-2 md:order-1"}`}>
          {textBlock}
        </div>
      </div>
    </EfShell>
  );
}
