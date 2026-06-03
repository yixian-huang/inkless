import { useTranslation } from "react-i18next";
import { resolveLocale } from "@/utils/locale";

interface HeaderLanguageToggleProps {
  variant?: "corporate-bar" | "corporate-mobile" | "inline";
  scrolled?: boolean;
  onAfterToggle?: () => void;
}

export default function HeaderLanguageToggle({
  variant = "inline",
  scrolled = true,
  onAfterToggle,
}: HeaderLanguageToggleProps) {
  const { i18n } = useTranslation("common");

  const toggleLanguage = () => {
    const newLang = resolveLocale(i18n.language) === "zh" ? "en" : "zh";
    i18n.changeLanguage(newLang);
    onAfterToggle?.();
  };

  if (variant === "corporate-bar") {
    return (
      <button
        type="button"
        onClick={toggleLanguage}
        className="text-xs hover:opacity-80 transition-opacity cursor-pointer"
      >
        {resolveLocale(i18n.language) === "zh" ? "English" : "\u4E2D\u6587"}
      </button>
    );
  }

  if (variant === "corporate-mobile") {
    return (
      <button
        type="button"
        onClick={toggleLanguage}
        className="text-sm text-gray-500 hover:text-blue-600 transition-colors cursor-pointer"
      >
        {resolveLocale(i18n.language) === "zh" ? "Switch to English" : "切换到中文"}
      </button>
    );
  }

  return (
    <button
      type="button"
      onClick={toggleLanguage}
      className={`hidden lg:inline text-xs transition-colors cursor-pointer ml-2 ${
        variant === "inline"
          ? "text-on-surface-muted hover:text-primary"
          : scrolled
            ? "text-gray-700 hover:text-blue-600"
            : "text-white/80 hover:text-white"
      }`}
    >
      {resolveLocale(i18n.language) === "zh" ? "EN" : "中文"}
    </button>
  );
}
