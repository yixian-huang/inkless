interface MobileMenuButtonProps {
  open: boolean;
  onToggle: () => void;
  className?: string;
  ariaLabel?: string;
}

export default function MobileMenuButton({
  open,
  onToggle,
  className = "text-on-surface",
  ariaLabel = "Toggle menu",
}: MobileMenuButtonProps) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className={`lg:hidden p-2 cursor-pointer transition-colors ${className}`}
      aria-label={ariaLabel}
      aria-expanded={open}
    >
      <div className="w-5 h-4 flex flex-col justify-between">
        <span className={`block h-0.5 w-full bg-current transition-all duration-200 ${open ? "rotate-45 translate-y-[7px]" : ""}`} />
        <span className={`block h-0.5 w-full bg-current transition-all duration-200 ${open ? "opacity-0" : ""}`} />
        <span className={`block h-0.5 w-full bg-current transition-all duration-200 ${open ? "-rotate-45 -translate-y-[7px]" : ""}`} />
      </div>
    </button>
  );
}
