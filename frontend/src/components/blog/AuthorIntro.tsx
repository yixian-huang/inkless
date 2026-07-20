import AuthorSocialLinks from "@/components/blog/AuthorSocialLinks";
import { useIsReadingLayout, useIsThemeHomePath } from "@/plugins/hooks";

interface AuthorIntroProps {
  avatar?: string;
  name: string;
  tagline?: string;
  bio?: string;
  intro: string;
  /** Optional secondary line under the name (e.g. English name). */
  subtitle?: string;
  showSocials?: boolean;
}

/**
 * Home / profile hero. On reading themes, centered lockup with room to breathe
 * so the first screen feels intentional rather than a sparse left column.
 */
export default function AuthorIntro({
  avatar,
  name,
  tagline,
  bio,
  intro,
  subtitle,
  showSocials = true,
}: AuthorIntroProps) {
  const isReading = useIsReadingLayout();
  const isThemeHome = useIsThemeHomePath();
  const isHomeHero = isReading && isThemeHome;

  const showHeroAvatar = Boolean(avatar) && (!isReading || isThemeHome);
  const bodyIntro =
    bio?.trim() ||
    (intro.trim() && intro.trim() !== tagline?.trim() ? intro.trim() : "");

  if (isHomeHero) {
    return (
      <header className="mb-12 md:mb-16">
        <div className="flex flex-col items-center text-center px-1 pt-4 pb-10 md:pt-8 md:pb-14 min-h-[min(52vh,28rem)] justify-center">
          {showHeroAvatar && (
            <img
              src={avatar}
              alt=""
              className="w-[5.5rem] h-[5.5rem] md:w-24 md:h-24 rounded-full object-contain bg-[#141310] mb-7 ring-1 ring-border/80 shadow-sm"
            />
          )}
          <h1 className="text-[2rem] sm:text-4xl md:text-[2.75rem] font-heading font-normal text-on-surface tracking-tight leading-[1.12]">
            {name}
          </h1>
          {subtitle?.trim() && (
            <p className="mt-2 text-sm font-sans tracking-[0.12em] text-on-surface-muted uppercase">
              {subtitle.trim()}
            </p>
          )}
          {tagline?.trim() && (
            <p className="mt-4 max-w-md text-base md:text-lg font-sans font-normal text-on-surface-muted leading-snug">
              {tagline.trim()}
            </p>
          )}
          {bodyIntro && (
            <p className="mt-5 max-w-lg text-[0.95rem] md:text-base font-sans text-on-surface-muted/90 leading-relaxed whitespace-pre-wrap">
              {bodyIntro}
            </p>
          )}
          {showSocials && (
            <div className="mt-8">
              <AuthorSocialLinks
                className="flex flex-wrap items-center justify-center gap-x-6 gap-y-2"
                linkClassName="text-xs font-sans uppercase tracking-[0.16em] text-on-surface-muted hover:text-on-surface transition-colors"
              />
            </div>
          )}
        </div>
        <div className="border-t border-border" aria-hidden="true" />
      </header>
    );
  }

  return (
    <header className={isReading ? "mb-10 pb-8 border-b border-border" : "mb-12"}>
      {showHeroAvatar && (
        <img
          src={avatar}
          alt={name}
          className={
            isReading
              ? "w-24 h-24 rounded-full object-contain bg-[#141310] mb-6 ring-1 ring-border"
              : "w-20 h-20 rounded-full object-cover mb-4 border border-border"
          }
        />
      )}
      <h1
        className={
          isReading
            ? "text-4xl md:text-[2.75rem] font-heading font-normal text-on-surface tracking-tight leading-[1.15]"
            : "text-3xl md:text-4xl font-heading font-bold text-on-surface tracking-tight"
        }
      >
        {name}
      </h1>
      {tagline && (
        <p
          className={
            isReading
              ? "mt-3 text-lg text-on-surface-muted font-sans font-normal"
              : "mt-2 text-lg text-on-surface-muted"
          }
        >
          {tagline}
        </p>
      )}
      {bodyIntro && (
        <p
          className={
            isReading
              ? "mt-6 text-on-surface text-[1.125rem] leading-relaxed whitespace-pre-wrap"
              : "mt-4 text-on-surface leading-relaxed whitespace-pre-wrap"
          }
        >
          {bodyIntro}
        </p>
      )}
    </header>
  );
}
