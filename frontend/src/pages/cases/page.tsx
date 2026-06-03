import { useTranslation } from 'react-i18next';
import PageHero from '../../components/feature/PageHero';
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

interface CaseCategory {
  title?: string;
  items?: string[];
}

interface CasesPageConfig {
  hero?: HeroConfig;
  cases?: CaseCategory[];
}

export default function CasesPage() {
  useDocumentTitle("成功案例");
  const { i18n } = useTranslation('common');
  const locale = resolveLocale(i18n.language);

  const { loading, error, config } = usePublicContent('cases', {
    locale,
    autoNormalize: true,
  });

  if (loading) {
    return (
      <>
        <div className="min-h-screen bg-white flex items-center justify-center">
          <div className="text-gray-600">Loading...</div>
        </div>
      </>
    );
  }

  if (error) {
    return (
      <>
        <div className="min-h-screen bg-white flex items-center justify-center">
          <div className="text-red-600">Failed to load page content</div>
        </div>
      </>
    );
  }

  const pageConfig = (config as CasesPageConfig & { cases?: unknown }) || {};
  const hero = pageConfig.hero || {};
  // Accept both legacy [{title, items}] shape and new card-grid {title, cards:[{title, description}]} shape
  const rawCases = pageConfig.cases as unknown;
  let categories: CaseCategory[];
  if (Array.isArray(rawCases)) {
    categories = rawCases as CaseCategory[];
  } else if (rawCases && typeof rawCases === "object" && Array.isArray((rawCases as { cards?: unknown }).cards)) {
    categories = ((rawCases as { cards: Array<{ title?: string; description?: string }> }).cards).map((c) => ({
      title: c.title,
      items: c.description ? [c.description] : [],
    }));
  } else {
    categories = [];
  }

  return (
    <>
      <PageHero
        label={hero.label}
        title={hero.title}
        alt="Case List Hero"
        imageSrc={hero.image?.url || '/images/case-hero-bg.png'}
      />

      <section className="py-12 md:py-16 lg:py-24 bg-white">
        <div className="max-w-layout mx-auto px-6 sm:px-10 md:px-16 lg:px-24">
          <div className="space-y-10 md:space-y-14">
            {categories.map((category, index) => {
              const items = category.items || [];
              if (!category.title && items.length === 0) return null;
              return (
                <div key={index}>
                  {category.title && (
                    <h2 className="text-xl md:text-2xl font-bold text-primary mb-4">
                      {category.title}
                    </h2>
                  )}
                  {items.length > 0 && (
                    <ul className="space-y-3">
                      {items.map((item, i) => (
                        <li
                          key={i}
                          className="text-base text-gray-700 leading-relaxed"
                        >
                          {item}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </section>
    </>
  );
}
