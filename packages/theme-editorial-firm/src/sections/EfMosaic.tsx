import { EfShell } from "./shell";
import { asArray, asString, type SectionProps } from "./types";

export interface EfMosaicTile {
  image?: string;
  label?: string;
  href?: string;
}

export interface EfMosaicData {
  title?: string;
  tiles?: EfMosaicTile[];
}

export default function EfMosaic({ data }: SectionProps<EfMosaicData>) {
  const title = asString(data.title);
  const tiles = asArray<EfMosaicTile>(data.tiles).filter((t) => asString(t?.image));

  if (!title && tiles.length === 0) return null;

  return (
    <EfShell>
      {title ? (
        <h2 className="font-heading text-3xl md:text-4xl text-on-surface font-semibold mb-8 md:mb-12">
          {title}
        </h2>
      ) : null}

      <div className="grid grid-cols-2 md:grid-cols-3 gap-3 md:gap-4">
        {tiles.map((tile, i) => {
          const image = asString(tile.image);
          const label = asString(tile.label);
          const href = asString(tile.href);

          const content = (
            <figure className="group relative overflow-hidden rounded-sm bg-surface-alt aspect-square">
              <img
                src={image}
                alt={label || ""}
                className="w-full h-full object-cover object-center transition-transform duration-500 group-hover:scale-[1.03]"
              />
              {label ? (
                <figcaption className="absolute inset-x-0 bottom-0 p-3 md:p-4 bg-gradient-to-t from-black/60 to-transparent">
                  <span className="text-sm text-white font-medium">{label}</span>
                </figcaption>
              ) : null}
            </figure>
          );

          return href ? (
            <a key={i} href={href} className="block focus:outline-none focus-visible:ring-2 focus-visible:ring-accent">
              {content}
            </a>
          ) : (
            <div key={i}>{content}</div>
          );
        })}
      </div>
    </EfShell>
  );
}
