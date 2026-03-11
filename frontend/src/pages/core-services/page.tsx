import { useTranslation } from 'react-i18next';
import { PublicLayout } from '@/theme/layouts';
import PageHero from '@/components/feature/PageHero';
import { usePublicContent } from '@/hooks/usePublicContent';
import type { Locale } from '@/api/publicContent';

interface MediaRef {
  url?: string;
  alt?: string;
}

interface HeroConfig {
  label?: string;
  title?: string;
  image?: MediaRef;
}

interface ServiceBlock {
  title?: string;
  description?: string;
  image?: MediaRef;
}

interface CoreServicesPageConfig {
  hero?: HeroConfig;
  services?: ServiceBlock[];
}

/** 服务区块 - 图片 */
function ServiceBlockImage({
  src,
  alt,
  className = '',
}: {
  src: string;
  alt: string;
  className?: string;
}) {
  return (
    <div
      className={`w-full aspect-[4/3] max-h-[400px] lg:max-h-none overflow-hidden rounded-lg ${className}`.trim()}
    >
      <img
        src={src}
        alt={alt}
        className="w-full h-full object-cover object-center"
      />
    </div>
  );
}

/** 服务区块 - 标题与描述 */
function ServiceBlockText({
  title,
  description,
  className = '',
}: {
  title: string;
  description: string;
  className?: string;
}) {
  return (
    <div className={`w-full h-full py-12 px-10 md:px-16 ${className}`.trim()}>
      <div className="mb-4">
        <h2 className="text-xl md:text-2xl font-bold text-primary">
          {title}
        </h2>
      </div>
      <p className="text-base text-gray-700 leading-relaxed">
        {description}
      </p>
    </div>
  );
}

export default function CoreServicesPage() {
  const { i18n } = useTranslation('common');
  const locale = (i18n.language === 'zh' || i18n.language.startsWith('zh') ? 'zh' : 'en') as Locale;

  const { loading, error, config } = usePublicContent('core-services', {
    locale,
    autoNormalize: true,
  });

  if (loading) {
    return (
      <PublicLayout>
        <div className="flex items-center justify-center py-32">
          <div className="text-gray-600">Loading...</div>
        </div>
      </PublicLayout>
    );
  }

  if (error) {
    return (
      <PublicLayout>
        <div className="flex items-center justify-center py-32">
          <div className="text-red-600">Failed to load page content</div>
        </div>
      </PublicLayout>
    );
  }

  const pageConfig = (config as CoreServicesPageConfig) || {};
  const hero = pageConfig.hero || {};
  const services = pageConfig.services || [];

  return (
    <PublicLayout>

      <PageHero
        label={hero.label || ''}
        title={hero.title || ''}
        alt="Core Services Hero"
        imageSrc={hero.image?.url}
      />

      <div className="py-12 md:py-16 lg:py-24 bg-white">
        {services.map((service, index) => {
          if (!service.title || !service.description) return null;
          const isImageLeft = index % 2 === 0;
          const imageSrc = service.image?.url || `/images/service/${index + 1}.png`;
          return (
            <section key={index} className="bg-white">
              <div className="max-w-layout mx-auto px-4 md:px-6">
                <div className="grid grid-cols-1 lg:grid-cols-2 items-center gap-8 lg:gap-12">
                  {isImageLeft ? (
                    <>
                      <ServiceBlockImage
                        src={imageSrc}
                        alt={service.title}
                        className="order-2 lg:order-1"
                      />
                      <ServiceBlockText
                        title={service.title}
                        description={service.description}
                        className="order-1 lg:order-2"
                      />
                    </>
                  ) : (
                    <>
                      <ServiceBlockText title={service.title} description={service.description} />
                      <ServiceBlockImage src={imageSrc} alt={service.title} />
                    </>
                  )}
                </div>
              </div>
            </section>
          );
        })}
      </div>

    </PublicLayout>
  );
}
