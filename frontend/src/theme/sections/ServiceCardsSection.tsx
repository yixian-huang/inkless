import type { SectionProps } from "../types";

export interface ServiceCard {
  title?: string;
  description?: string;
  image?: string;
  link?: string;
}

export interface ServiceCardsSectionData {
  title?: string;
  services?: ServiceCard[];
}

export default function ServiceCardsSection({ data }: SectionProps<ServiceCardsSectionData>) {
  const { title, services } = data;

  return (
    <div className="max-w-layout w-full h-full mx-auto px-4 md:px-content xl:px-8">
      {title && (
        <div className="flex items-center mb-8 sm:mb-12">
          <div className="w-5 h-5 sm:w-[26px] sm:h-[26px] bg-accent mr-2 sm:mr-3 flex-shrink-0 rounded-sm" />
          <h2 className="text-2xl sm:text-3xl md:text-4xl font-bold text-primary truncate min-w-0">
            {title}
          </h2>
          <span className="ml-1 sm:ml-2 text-xl sm:text-2xl text-accent flex-shrink-0 cursor-pointer">
            &gt;
          </span>
        </div>
      )}
      {services && services.length > 0 && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-8 sm:gap-x-8 sm:gap-y-10 xl:gap-x-10 xl:gap-y-12">
          {services.map((service, index) => (
            <div
              key={index}
              className="flex flex-col sm:flex-row gap-4 sm:gap-5 p-4 sm:p-0 rounded-lg sm:rounded-none bg-surface-alt/80 sm:bg-transparent"
            >
              <div className="w-full h-[180px] sm:h-[220px] lg:h-[260px] xl:h-[280px] flex-shrink-0 rounded-md overflow-hidden bg-surface-alt">
                <img
                  src={service.image || `/images/service-${index + 1}.png`}
                  alt={service.title || `Service ${index + 1}`}
                  className="w-full h-full object-cover object-top sm:object-contain sm:object-center"
                />
              </div>
              <div className="flex flex-col justify-center min-w-0">
                {service.title && (
                  <h3 className="text-base font-bold text-primary mb-1 sm:mb-2">
                    {service.title}
                  </h3>
                )}
                {service.description && (
                  <p className="text-sm text-on-surface-muted leading-relaxed mb-2 sm:mb-3 line-clamp-3 sm:line-clamp-none">
                    {service.description}
                  </p>
                )}
                {service.link && (
                  <a
                    href="#"
                    className="text-sm font-bold text-primary hover:text-accent transition-colors cursor-pointer"
                  >
                    {service.link}
                  </a>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
