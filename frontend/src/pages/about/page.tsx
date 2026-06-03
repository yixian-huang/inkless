import { useTranslation } from 'react-i18next';
import PageHero from '@/components/feature/PageHero';
import { usePublicContent } from '@/hooks/usePublicContent';
import { useDocumentTitle } from '@/hooks/useDocumentTitle';
import { resolveLocale } from '@/utils/locale';

interface MediaRef {
  url?: string;
  alt?: string;
}

interface HeroConfig {
  label?: string;
  title?: string;
  image?: MediaRef;
}

interface CompanyProfileConfig {
  title?: string;
  description?: string;
}

interface BlockConfig {
  layout?: string;
  title?: string;
  description?: string;
  image?: MediaRef;
}

interface AboutPageConfig {
  hero?: HeroConfig;
  companyProfile?: CompanyProfileConfig;
  blocks?: BlockConfig[];
}

export default function AboutPage() {
  useDocumentTitle("关于我们");
  const { i18n } = useTranslation('common');
  const locale = resolveLocale(i18n.language);

  const { loading, error, config } = usePublicContent('about', {
    locale,
    autoNormalize: true,
  });

  if (loading) {
    return (
      <>
        <div className="flex items-center justify-center py-32">
          <div className="text-gray-600">Loading...</div>
        </div>
      </>
    );
  }

  if (error) {
    return (
      <>
        <div className="flex items-center justify-center py-32">
          <div className="text-red-600">Failed to load page content</div>
        </div>
      </>
    );
  }

  const pageConfig = (config as AboutPageConfig & { blocks?: unknown }) || {};
  const hero = pageConfig.hero || {};
  const companyProfile = pageConfig.companyProfile || {};
  // Accept legacy blocks array OR new card-grid {title, cards:[...]} shape
  const rawBlocks = pageConfig.blocks as unknown;
  let blocks: BlockConfig[];
  if (Array.isArray(rawBlocks)) {
    blocks = rawBlocks as BlockConfig[];
  } else if (rawBlocks && typeof rawBlocks === "object" && Array.isArray((rawBlocks as { cards?: unknown }).cards)) {
    blocks = ((rawBlocks as { cards: BlockConfig[] }).cards);
  } else {
    blocks = [];
  }

  return (
    <>

      <PageHero
        label={hero.label || ''}
        title={hero.title || ''}
        alt="About Us Hero"
        imageSrc={hero.image?.url}
      />

      {/* Section 1: 公司简介 - 标题左、描述右 */}
      {companyProfile.title && (
        <section className="py-12 md:py-16 lg:py-24 bg-white">
          <div className="max-w-layout mx-auto px-4 md:px-6">
            <div className="grid grid-cols-1 lg:grid-cols-12 items-center gap-8 lg:gap-12">
              <div className="lg:col-span-3 text-center">
                <h2 className="text-3xl md:text-4xl font-bold text-primary uppercase">
                  {companyProfile.title}
                </h2>
              </div>
              {companyProfile.description && (
                <div className="lg:col-span-9">
                  <p className="text-2xl  md:text-3xl leading-relaxed text-primary-dark">
                    {companyProfile.description}
                  </p>
                </div>
              )}
            </div>
          </div>
        </section>
      )}

      {/* 内容区块：根据 layout 决定图文左右排布 */}
      {blocks.length > 0 && (
        <div className="py-12 md:py-16 lg:py-24 bg-white">
          {blocks.map((block, index) => {
            if (!block.description) return null;
            const isImageLeft = block.layout === 'imageLeft' || (block.layout == null && index % 2 === 0);
            return (
              <section key={index} className="bg-white">
                <div className="max-w-layout mx-auto px-4 md:px-6 mb-12">
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 lg:gap-12 items-center">
                    {isImageLeft ? (
                      <>
                        <div className="w-full aspect-[4/3] max-h-[400px] lg:max-h-none overflow-hidden rounded-lg order-2 lg:order-1">
                          <img
                            src={block.image?.url || `/images/about-us/about-us-${index + 1}.png`}
                            alt={block.image?.alt || block.title || 'About Us'}
                            className="w-full h-full object-cover object-center"
                          />
                        </div>
                        <div className="w-full py-12 px-10 md:px-16 order-1 lg:order-2">
                          {block.title && (
                            <h2 className="text-3xl md:text-4xl font-bold text-primary mb-4">{block.title}</h2>
                          )}
                          <p className="text-xl md:text-2xl leading-relaxed text-primary-dark">
                            {block.description}
                          </p>
                        </div>
                      </>
                    ) : (
                      <>
                        <div className="w-full py-12 px-10 md:px-16">
                          {block.title && (
                            <h2 className="text-3xl md:text-4xl font-bold text-primary mb-4">{block.title}</h2>
                          )}
                          <p className="text-xl md:text-2xl leading-relaxed text-primary-dark">
                            {block.description}
                          </p>
                        </div>
                        <div className="w-full aspect-[4/3] max-h-[400px] lg:max-h-none overflow-hidden rounded-lg">
                          <img
                            src={block.image?.url || `/images/about-us/about-us-${index + 1}.png`}
                            alt={block.image?.alt || block.title || 'About Us'}
                            className="w-full h-full object-cover object-center"
                          />
                        </div>
                      </>
                    )}
                  </div>
                </div>
              </section>
            );
          })}
        </div>
      )}

    </>
  );
}
