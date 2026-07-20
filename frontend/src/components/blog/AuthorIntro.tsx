import { useIsReadingLayout, useIsThemeHomePath } from "@/plugins/hooks";

interface AuthorIntroProps {
  avatar?: string;
  name: string;
  tagline?: string;
  bio?: string;
  intro: string;
  /** Optional secondary line under the name (e.g. English name). Default: hidden on compact home. */
  subtitle?: string;
  /**
   * Hero social links. Prefer false when Header already shows socials
   * (avoids duplicate GitHub/Twitter on the home first screen).
   */
  showSocials?: boolean;
  /** When false (default on home), bio/intro body is omitted to free first-screen space. */
  showBio?: boolean;
  /** When false (default on home), hide English/alternate subtitle under the name. */
  showSubtitle?: boolean;
}

/**
 * Compact home hero for reading themes: avatar + name + one-line tagline.
 * Socials belong in the header; bio belongs on About or behind a future toggle.
 */
export default function AuthorIntro({
  avatar,
  name,
  tagline,
  bio,
  intro,
  subtitle,
  showBio = false,
  showSubtitle = false,
}: AuthorIntroProps) {
  // showSocials is intentionally unused: home hero must not duplicate Header socials.
  const isReading = useIsReadingLayout();
  const isThemeHome = useIsThemeHomePath();
  const isHomeHero = isReading && isThemeHome;

  const showHeroAvatar = Boolean(avatar) && (!isReading || isThemeHome);
  const bodyIntro =
    bio?.trim() ||
    (intro.trim() && intro.trim() !== tagline?.trim() ? intro.trim() : "");

  if (isHomeHero) {
    return (
      <header className="mb-8 md:mb-10">
        <div className="flex flex-col items-center text-center px-1 pt-2 pb-6 md:pt-3 md:pb-8">
          {showHeroAvatar && (
            <img
              src={avatar}
              alt=""
              className="w-14 h-14 md:w-16 md:h-16 rounded-full object-contain bg-[#141310] mb-4 ring-1 ring-border/80"
            />
          )}
          <h1 className="text-2xl sm:text-[1.75rem] md:text-3xl font-heading font-normal text-on-surface tracking-tight leading-tight">
            {name}
          </h1>
          {showSubtitle && subtitle?.trim() && (
            <p className="mt-1.5 text-xs font-sans tracking-[0.12em] text-on-surface-muted uppercase">
              {subtitle.trim()}
            </p>
          )}
          {tagline?.trim() && (
            <p className="mt-2.5 max-w-sm text-sm md:text-[0.95rem] font-sans font-normal text-on-surface-muted leading-snug">
              {tagline.trim()}
            </p>
          )}
          {showBio && bodyIntro && (
            <p className="mt-3 max-w-md text-sm font-sans text-on-surface-muted/90 leading-relaxed whitespace-pre-wrap">
              {bodyIntro}
            </p>
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
              ? "w-20 h-20 rounded-full object-contain bg-[#141310] mb-5 ring-1 ring-border"
              : "w-20 h-20 rounded-full object-cover mb-4 border border-border"
          }
        />
      )}
      <h1
        className={
          isReading
            ? "text-3xl md:text-4xl font-heading font-normal text-on-surface tracking-tight leading-[1.15]"
            : "text-3xl md:text-4xl font-heading font-bold text-on-surface tracking-tight"
        }
      >
        {name}
      </h1>
      {tagline && (
        <p
          className={
            isReading
              ? "mt-2 text-base text-on-surface-muted font-sans font-normal"
              : "mt-2 text-lg text-on-surface-muted"
          }
        >
          {tagline}
        </p>
      )}
      {(showBio || !isReading) && bodyIntro && (
        <p
          className={
            isReading
              ? "mt-4 text-on-surface text-base leading-relaxed whitespace-pre-wrap"
              : "mt-4 text-on-surface leading-relaxed whitespace-pre-wrap"
          }
        >
          {bodyIntro}
        </p>
      )}
    </header>
  );
}
