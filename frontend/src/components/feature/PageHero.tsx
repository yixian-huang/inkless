const DEFAULT_HERO_IMAGE = '/images/about-us-hero-bg.png';

type PageHeroProps = {
  title: string;
  /** 小字，显示在 title 上方（背景图时常用） */
  label?: string;
  /** 小字，显示在 title 下方（纯色 Hero 时常用，如「请关注我们」） */
  subtitle?: string;
  alt?: string;
  /** 背景图；与 backgroundColor 二选一，同时存在时优先背景图 */
  imageSrc?: string;
  /** 纯色背景；指定后不展示图片与遮罩，仅用该色铺满 */
  backgroundColor?: string;
};

/**
 * 企业子页通用 Hero：支持背景图或纯色背景，布局一致。
 * 用于：关于我们、我们的优势、专家团队、案例清单、联系我们等页面。
 */
export default function PageHero({
  title,
  label,
  subtitle,
  alt = 'Page Hero',
  imageSrc = DEFAULT_HERO_IMAGE,
  backgroundColor,
}: PageHeroProps) {
  const useImage = !backgroundColor;
  const src = useImage ? imageSrc : undefined;

  return (
    <section
      data-page-hero
      className={ useImage ?
        `relative h-[280px] sm:h-[360px] md:h-[440px] lg:h-[560px]`
      : `relative h-[200px] sm:h-[300px] md:h-[400px] lg:h-[500px] bg-[${backgroundColor}]`
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
              alt={alt}
              className="w-full h-full object-cover object-center"
            />
            <div className="absolute inset-0 bg-black/40" />
          </>
        )}
      </div>
      <div
        className=
            'absolute left-0 right-0 bottom-[20%] z-10'
        
      >
        <div className="max-w-layout w-full mx-auto px-4 md:px-6">
          {label && (
            <p className="text-white text-sm sm:text-base mb-1">{label}</p>
          )}
          <h1 className="text-white text-2xl md:text-3xl lg:text-4xl font-bold uppercase tracking-wide">
            {title}
          </h1>
          {subtitle && (
            <p className="text-white text-base md:text-lg mt-2 opacity-95">
              {subtitle}
            </p>
          )}
        </div>
      </div>
    </section>
  );
}
