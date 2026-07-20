/**
 * Inkless admin design tokens — ink wash + print / magazine editorial.
 *
 * Palette intent:
 * - Ink: warm near-black charcoal (not cool navy SaaS)
 * - Paper: ivory / rice-paper for content canvas
 * - Accent: restrained vermillion for rare emphasis; primary actions use ink
 */
export const adminTheme = {
  /* ── Surfaces (paper studio) ──────────────────────────────── */
  pageBg: "bg-[#f5f1ea]",
  shellBg: "bg-[#f5f1ea]",
  surface: "bg-[#fbfaf7]",
  surfaceMuted: "bg-[#f0ebe3]/90",
  surfaceSubtle: "bg-[#f0ebe3]",

  /* ── Sidebar (warm ink rail) ──────────────────────────────── */
  sidebarBg: "bg-[#171512]",
  sidebarBorder: "border-[#2e2924]",
  sidebarText: "text-[#d6d0c6]",
  sidebarTextMuted: "text-[#8a8378]",
  sidebarHover: "hover:bg-[#f4efe6]/[0.07] hover:text-[#f7f3ec]",
  sidebarActive: "bg-[#f4efe6] text-[#171512] shadow-sm",
  sidebarActiveIcon: "text-[#171512]",
  sidebarIcon: "text-[#9a9286] group-hover:text-[#e8e2d8]",
  sidebarSearch:
    "w-full rounded-lg border border-[#3a342e] bg-[#1f1b17] py-2 pl-8 pr-8 text-xs text-[#e8e2d8] placeholder:text-[#7a7368] outline-none transition focus:border-[#8a7f6e] focus:bg-[#24201b] focus:ring-1 focus:ring-[#c4b8a4]/25",

  /* ── Typography (editorial) ───────────────────────────────── */
  pageTitle:
    "text-[1.375rem] font-semibold tracking-[-0.02em] text-[#1a1814] sm:text-[1.65rem]",
  pageDesc: "mt-1.5 text-sm leading-relaxed text-[#6b6560]",
  sectionTitle: "text-base font-semibold tracking-[-0.01em] text-[#1a1814]",
  sectionDesc: "mt-0.5 text-sm text-[#6b6560]",
  label: "text-sm font-medium text-[#3d3832]",
  muted: "text-[#6b6560]",
  caption: "text-xs tracking-wide text-[#8a8378]",
  body: "text-sm text-[#3d3832]",

  /* ── Borders & elevation (print sheet) ────────────────────── */
  border: "border-[#e4ddd2]",
  borderSubtle: "border-[#ece6dc]",
  card:
    "bg-[#fbfaf7] rounded-xl border border-[#e4ddd2] shadow-[0_1px_0_rgba(26,24,20,0.04),0_8px_24px_rgba(26,24,20,0.04)]",
  cardPad: "p-5 sm:p-6",
  cardHeader:
    "flex items-start justify-between gap-3 border-b border-[#ece6dc] px-5 py-4 sm:px-6",
  panel: "rounded-xl border border-[#e4ddd2] bg-[#fbfaf7]",
  dropdown:
    "rounded-xl border border-[#e4ddd2] bg-[#fbfaf7] py-1 shadow-[0_16px_40px_rgba(26,24,20,0.12)]",

  /* ── Interactive ──────────────────────────────────────────── */
  focusRing:
    "focus:outline-none focus-visible:ring-2 focus-visible:ring-[#1a1814]/20 focus-visible:ring-offset-2 focus-visible:ring-offset-[#f5f1ea]",
  focusRingInset:
    "focus:outline-none focus-visible:ring-2 focus-visible:ring-[#1a1814]/25 focus-visible:ring-offset-0",
  transition: "transition-all duration-150 ease-out",

  /* ── Form controls ────────────────────────────────────────── */
  input:
    "w-full rounded-lg border border-[#e4ddd2] bg-[#fbfaf7] px-3 py-2 text-sm text-[#1a1814] placeholder:text-[#a39b90] shadow-[inset_0_1px_2px_rgba(26,24,20,0.03)] transition hover:border-[#d4cbbf] focus:border-[#1a1814]/40 focus:outline-none focus:ring-2 focus:ring-[#1a1814]/10 disabled:cursor-not-allowed disabled:bg-[#f0ebe3] disabled:text-[#a39b90]",
  select:
    "rounded-lg border border-[#e4ddd2] bg-[#fbfaf7] px-3 py-2 text-sm text-[#1a1814] shadow-[inset_0_1px_2px_rgba(26,24,20,0.03)] transition hover:border-[#d4cbbf] focus:border-[#1a1814]/40 focus:outline-none focus:ring-2 focus:ring-[#1a1814]/10 disabled:cursor-not-allowed disabled:bg-[#f0ebe3]",
  textarea:
    "w-full rounded-lg border border-[#e4ddd2] bg-[#fbfaf7] px-3 py-2 text-sm text-[#1a1814] placeholder:text-[#a39b90] shadow-[inset_0_1px_2px_rgba(26,24,20,0.03)] transition hover:border-[#d4cbbf] focus:border-[#1a1814]/40 focus:outline-none focus:ring-2 focus:ring-[#1a1814]/10 disabled:cursor-not-allowed disabled:bg-[#f0ebe3]",
  checkbox:
    "h-4 w-4 rounded border-[#d4cbbf] text-[#1a1814] focus:ring-[#1a1814]/20",

  /* ── Toolbar / filter bars ────────────────────────────────── */
  toolbar:
    "flex flex-wrap items-center gap-2 rounded-xl border border-[#e4ddd2] bg-[#fbfaf7] p-3 shadow-[0_1px_0_rgba(26,24,20,0.03)] sm:gap-3 sm:p-3.5",
  filterChip:
    "inline-flex items-center gap-1.5 rounded-full border border-[#e4ddd2] bg-[#f0ebe3]/70 px-2.5 py-1 text-xs font-medium tracking-wide text-[#5c564f]",

  /* ── Table ────────────────────────────────────────────────── */
  tableHead:
    "bg-[#f0ebe3]/80 text-left text-[11px] font-semibold uppercase tracking-[0.08em] text-[#8a8378]",
  tableRow: "transition-colors hover:bg-[#f5f1ea]/80",
  tableCell: "px-4 py-3 text-[#3d3832]",
  tableCellHead: "px-4 py-3 whitespace-nowrap",

  /* ── Status ───────────────────────────────────────────────── */
  dangerSoft: "bg-[#faf0ee] text-[#8b3a32] border-[#ebd4cf]",
  successSoft: "bg-[#eef5ef] text-[#2f5d3a] border-[#d5e5d8]",
  warningSoft: "bg-[#faf4e8] text-[#7a5b22] border-[#eadfc4]",
  infoSoft: "bg-[#eef2f4] text-[#2f4a5c] border-[#d5e0e6]",

  /* ── Primary accent (ink, not electric blue) ──────────────── */
  primary: "bg-[#1a1814] text-[#f7f3ec] hover:bg-[#2a2622] active:bg-[#0f0e0c]",
  primarySoft:
    "bg-[#f0ebe3] text-[#1a1814] hover:bg-[#e8e2d8] border border-[#d4cbbf]",
  link: "text-[#1a1814] hover:text-[#3d3832] font-medium underline-offset-4 hover:underline decoration-[#d4cbbf]",

  /* ── Cinnabar (rare emphasis only) ────────────────────────── */
  cinnabar: "text-[#9b3b2e]",
  cinnabarSoft: "bg-[#faf0ee] text-[#9b3b2e] border border-[#ebd4cf]",
} as const;

export type AdminTheme = typeof adminTheme;
