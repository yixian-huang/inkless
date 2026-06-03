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
  useDocumentTitle("核心服务");
  const { i18n } = useTranslation('common');
  const locale = resolveLocale(i18n.language);

  const { loading, error, config } = usePublicContent('core-services', {
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

  const pageConfig = (config as CoreServicesPageConfig & { services?: unknown }) || {};
  const hero = pageConfig.hero || {};
  // Accept legacy services array OR new service-cards {title, items:[{title, description, image}]} shape
  const rawServices = pageConfig.services as unknown;
  let services: ServiceBlock[];
  if (Array.isArray(rawServices)) {
    services = rawServices as ServiceBlock[];
  } else if (rawServices && typeof rawServices === "object" && Array.isArray((rawServices as { items?: unknown }).items)) {
    services = ((rawServices as { items: ServiceBlock[] }).items);
  } else {
    services = [];
  }

  return (
    <>

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

    </>
  );
}
