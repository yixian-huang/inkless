import { PRODUCT_BRAND } from "@/config/productBrand";

interface ProductLogoProps {
  collapsed?: boolean;
  className?: string;
  /**
   * default — dark wordmark on light UI
   * onDark — light wordmark (generic)
   * ink — admin ink-rail mark (paper glyph on warm charcoal)
   */
  variant?: "default" | "onDark" | "ink";
}

export function ProductLogo({
  collapsed = false,
  className = "",
  variant = "default",
}: ProductLogoProps) {
  const ink = variant === "ink" || variant === "onDark";
  const markSrc = ink ? "/brand/inkless-mark-ink.svg" : "/brand/inkless-mark.svg";

  return (
    <span
      className={`inline-flex items-center gap-2.5 ${className}`}
      aria-label={PRODUCT_BRAND.fullName}
    >
      <img className="h-8 w-8" src={markSrc} alt="Inkless" />
      {!collapsed && (
        <span className="flex flex-col leading-none">
          <span
            className={`text-[1.05rem] font-semibold tracking-[-0.02em] ${
              ink ? "text-[#f4efe6]" : "text-[#1a1814]"
            }`}
          >
            {PRODUCT_BRAND.name}
          </span>
          {ink ? (
            <span className="mt-0.5 text-[9px] font-medium uppercase tracking-[0.18em] text-[#8a8378]">
              Press · CMS
            </span>
          ) : null}
        </span>
      )}
    </span>
  );
}
