# Responsive Layout Optimization Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Optimize the corporate-classic theme for 1920x1080 and other mainstream resolutions by extending Tailwind breakpoints (xl/2xl) and widening the max-width container from 1200px to 1400px.

**Architecture:** Add xl (1280px) and 2xl (1536px) breakpoint values to existing Tailwind utility classes. Update global layout tokens (maxWidth, section spacing) via CSS variables. Each section component gets `xl:px-8` wrapper padding and component-specific spacing/sizing adjustments.

**Tech Stack:** React + TypeScript + Tailwind CSS 3 + CSS custom properties

**Spec:** `docs/superpowers/specs/2026-03-11-responsive-layout-optimization-design.md`

---

## Chunk 1: Global Tokens & Config

### Task 1: Update global layout tokens

**Files:**
- Modify: `frontend/src/theme/tokens.ts:45` (maxWidth value)
- Modify: `frontend/src/index.css:594` (CSS variable)
- Modify: `frontend/src/index.css:598-599` (add responsive section spacing)

- [ ] **Step 1: Update `tokens.ts` maxWidth**

Change line 45:
```typescript
// old
maxWidth: "1200px",
// new
maxWidth: "1400px",
```

- [ ] **Step 2: Update CSS variable in `index.css`**

Change line 594:
```css
/* old */
--layout-max-width: 1200px;
/* new */
--layout-max-width: 1400px;
```

- [ ] **Step 3: Add responsive section spacing in `index.css`**

After the closing `}` of `@layer base` block (after line 601), add:
```css
@media (min-width: 1280px) {
  :root {
    --layout-section-spacing: 6rem;
  }
}
```

- [ ] **Step 4: Verify no other references to 1200px in token/config files**

Run: `grep -r "1200px" frontend/src/theme/tokens.ts frontend/tailwind.config.ts frontend/src/index.css`
Expected: no matches remaining

- [ ] **Step 5: Commit**

```bash
git add frontend/src/theme/tokens.ts frontend/src/index.css
git commit -m "feat: widen layout max-width to 1400px and add responsive section spacing"
```

### Task 2: Update theme package tokens

**Files:**
- Modify: `frontend/src/theme/packages/modern-dark/index.ts:22`
- Modify: `frontend/src/theme/packages/warm-earth/index.ts:22`

- [ ] **Step 1: Update modern-dark maxWidth**

Change line 22:
```typescript
// old
maxWidth: "1200px",
// new
maxWidth: "1400px",
```

- [ ] **Step 2: Update warm-earth maxWidth**

Change line 22:
```typescript
// old
maxWidth: "1200px",
// new
maxWidth: "1400px",
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/theme/packages/modern-dark/index.ts frontend/src/theme/packages/warm-earth/index.ts
git commit -m "feat: update theme packages maxWidth to 1400px"
```

---

## Chunk 2: HeroSection & TextImageSection

### Task 3: Update HeroSection height and title scaling

**File:**
- Modify: `frontend/src/theme/sections/HeroSection.tsx`

- [ ] **Step 1: Change image branch height (line 20)**

```tsx
// old
? "relative h-[280px] sm:h-[360px] md:h-[440px] lg:h-[560px]"
// new
? "relative min-h-[280px] sm:min-h-[360px] md:min-h-[40vh] lg:min-h-[45vh] max-h-[600px]"
```

- [ ] **Step 2: Change color-background branch height (line 21)**

```tsx
// old
: "relative h-[200px] sm:h-[300px] md:h-[400px] lg:h-[500px]"
// new
: "relative min-h-[200px] sm:min-h-[300px] md:min-h-[35vh] lg:min-h-[40vh] max-h-[540px]"
```

- [ ] **Step 3: Add xl title scaling (line 45)**

```tsx
// old
<h1 className="text-white text-2xl md:text-3xl lg:text-4xl font-bold uppercase tracking-wide">
// new
<h1 className="text-white text-2xl md:text-3xl lg:text-4xl xl:text-5xl font-bold uppercase tracking-wide">
```

- [ ] **Step 4: Add xl:px-8 to content wrapper (line 40)**

```tsx
// old
<div className="max-w-layout w-full mx-auto px-4 md:px-content">
// new
<div className="max-w-layout w-full mx-auto px-4 md:px-content xl:px-8">
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/theme/sections/HeroSection.tsx
git commit -m "feat: HeroSection responsive height (vh units) and xl title scaling"
```

### Task 4: Update TextImageSection wrapper padding

**File:**
- Modify: `frontend/src/theme/sections/TextImageSection.tsx`

- [ ] **Step 1: Add xl:px-8 to outer wrapper (line 15)**

```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content mb-12">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 mb-12">
```

- [ ] **Step 2: Add xl:px-20 to text containers (lines 26 and 41)**

Line 26 (image-left, text-right):
```tsx
// old
<div className="w-full h-full py-12 px-10 md:px-16 order-1 lg:order-2">
// new
<div className="w-full h-full py-12 px-10 md:px-16 xl:px-20 order-1 lg:order-2">
```

Line 41 (image-right, text-left):
```tsx
// old
<div className="w-full h-full py-12 px-10 md:px-16">
// new
<div className="w-full h-full py-12 px-10 md:px-16 xl:px-20">
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/theme/sections/TextImageSection.tsx
git commit -m "feat: TextImageSection xl padding for wider screens"
```

---

## Chunk 3: ServiceCards, TeamGrid, CompanyProfile

### Task 5: Update ServiceCardsSection image sizing and gap

**File:**
- Modify: `frontend/src/theme/sections/ServiceCardsSection.tsx`

- [ ] **Step 1: Add xl:px-8 to wrapper (line 19)**

```tsx
// old
<div className="max-w-layout w-full h-full mx-auto px-4 md:px-content">
// new
<div className="max-w-layout w-full h-full mx-auto px-4 md:px-content xl:px-8">
```

- [ ] **Step 2: Add xl gap to grid (line 32)**

```tsx
// old
<div className="grid grid-cols-1 sm:grid-cols-2 gap-8 sm:gap-x-8 sm:gap-y-10">
// new
<div className="grid grid-cols-1 sm:grid-cols-2 gap-8 sm:gap-x-8 sm:gap-y-10 xl:gap-x-10 xl:gap-y-12">
```

- [ ] **Step 3: Change image container sizing (line 38)**

Remove fixed `sm:w-[320px]`, add responsive heights:
```tsx
// old
<div className="w-full h-[180px] sm:w-[320px] sm:h-[240px] flex-shrink-0 rounded-md overflow-hidden bg-surface-alt">
// new
<div className="w-full h-[180px] sm:h-[220px] lg:h-[260px] xl:h-[280px] flex-shrink-0 rounded-md overflow-hidden bg-surface-alt">
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/theme/sections/ServiceCardsSection.tsx
git commit -m "feat: ServiceCardsSection responsive image height and xl gap"
```

### Task 6: Update TeamGridSection avatar and gaps

**File:**
- Modify: `frontend/src/theme/sections/TeamGridSection.tsx`

- [ ] **Step 1: Add xl:px-8 to wrapper (line 27)**

```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
```

- [ ] **Step 2: Add xl gap to top-level grid (line 38)**

```tsx
// old
<div className="grid grid-cols-2 gap-8 md:gap-12 max-w-2xl mx-auto mb-12 md:mb-16">
// new
<div className="grid grid-cols-2 gap-8 md:gap-12 xl:gap-14 max-w-2xl mx-auto mb-12 md:mb-16">
```

- [ ] **Step 3: Add xl avatar size (line 41)**

```tsx
// old
<div className="w-32 h-32 md:w-40 md:h-40 rounded-full overflow-hidden border-2 border-border flex-shrink-0 mb-3">
// new
<div className="w-32 h-32 md:w-40 md:h-40 xl:w-44 xl:h-44 rounded-full overflow-hidden border-2 border-border flex-shrink-0 mb-3">
```

- [ ] **Step 4: Add xl gap to bio grid (line 62)**

```tsx
// old
<div className="grid grid-cols-1 lg:grid-cols-12 gap-6 lg:gap-8 items-start">
// new
<div className="grid grid-cols-1 lg:grid-cols-12 gap-6 lg:gap-8 xl:gap-10 items-start">
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/theme/sections/TeamGridSection.tsx
git commit -m "feat: TeamGridSection xl avatar size and gaps"
```

### Task 7: Update CompanyProfileSection gap

**File:**
- Modify: `frontend/src/theme/sections/CompanyProfileSection.tsx`

- [ ] **Step 1: Add xl:px-8 to wrapper (line 16)**

```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
```

- [ ] **Step 2: Add xl gap to grid (line 17)**

```tsx
// old
<div className="grid grid-cols-1 lg:grid-cols-12 items-center gap-8 lg:gap-12">
// new
<div className="grid grid-cols-1 lg:grid-cols-12 items-center gap-8 lg:gap-12 xl:gap-16">
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/theme/sections/CompanyProfileSection.tsx
git commit -m "feat: CompanyProfileSection xl gap"
```

---

## Chunk 4: Remaining Sections & Layouts

### Task 8: Update CardGridSection, ChecklistSection, RichTextSection, ContactFormSection wrappers

**Files:**
- Modify: `frontend/src/theme/sections/CardGridSection.tsx:31`
- Modify: `frontend/src/theme/sections/ChecklistSection.tsx:16`
- Modify: `frontend/src/theme/sections/RichTextSection.tsx:17`
- Modify: `frontend/src/theme/sections/ContactFormSection.tsx:65`

- [ ] **Step 1: CardGridSection — add xl:px-8 (line 31)**

```tsx
// old
<div className="max-w-layout w-full mx-auto px-4 md:px-content">
// new
<div className="max-w-layout w-full mx-auto px-4 md:px-content xl:px-8">
```

- [ ] **Step 2: ChecklistSection — add xl:px-8 and xl spacing (line 16-17)**

Wrapper (line 16):
```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
```

Vertical spacing (line 17):
```tsx
// old
<div className="space-y-10 md:space-y-14">
// new
<div className="space-y-10 md:space-y-14 xl:space-y-16">
```

- [ ] **Step 3: RichTextSection — add xl:px-8 (line 17)**

```tsx
// old
<div className={`max-w-layout mx-auto px-4 md:px-content ${alignClass}`}>
// new
<div className={`max-w-layout mx-auto px-4 md:px-content xl:px-8 ${alignClass}`}>
```

- [ ] **Step 4: ContactFormSection — add xl:px-8 (line 65)**

```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/theme/sections/CardGridSection.tsx frontend/src/theme/sections/ChecklistSection.tsx frontend/src/theme/sections/RichTextSection.tsx frontend/src/theme/sections/ContactFormSection.tsx
git commit -m "feat: add xl:px-8 wrapper padding to remaining sections"
```

### Task 9: Update StatsCounterSection wrapper

**File:**
- Modify: `frontend/src/plugins/themes/corporate-classic/StatsCounterSection.tsx:17`

- [ ] **Step 1: Add xl:px-8 (line 17)**

```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/plugins/themes/corporate-classic/StatsCounterSection.tsx
git commit -m "feat: StatsCounterSection xl padding"
```

### Task 10: Update layout components (Header, Footer, PublicLayout)

**Files:**
- Modify: `frontend/src/theme/layouts/ThemedHeader.tsx:229,242`
- Modify: `frontend/src/theme/layouts/ThemedFooter.tsx:33,52,54,94`
- Modify: `frontend/src/theme/layouts/PublicLayout.tsx:22`

- [ ] **Step 1: ThemedHeader — language bar wrapper (line 229)**

```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content flex justify-end">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 flex justify-end">
```

- [ ] **Step 2: ThemedHeader — nav wrapper (line 242)**

```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
```

- [ ] **Step 3: ThemedFooter — minimal style wrapper (line 33)**

```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content py-6">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 py-6">
```

- [ ] **Step 4: ThemedFooter — full style wrapper (line 52)**

```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content py-12">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 py-12">
```

- [ ] **Step 5: ThemedFooter — sections grid gap (line 54)**

```tsx
// old
<div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-8">
// new
<div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-8 xl:gap-10">
```

- [ ] **Step 6: ThemedFooter — fallback layout gap (line 94)**

```tsx
// old
<div className="flex flex-col md:flex-row md:items-start gap-8">
// new
<div className="flex flex-col md:flex-row md:items-start gap-8 xl:gap-10">
```

- [ ] **Step 7: PublicLayout — sidebar wrapper (line 22)**

```tsx
// old
<div className="max-w-layout mx-auto px-4 md:px-content py-8 flex gap-8">
// new
<div className="max-w-layout mx-auto px-4 md:px-content xl:px-8 py-8 flex gap-8">
```

- [ ] **Step 8: Commit**

```bash
git add frontend/src/theme/layouts/ThemedHeader.tsx frontend/src/theme/layouts/ThemedFooter.tsx frontend/src/theme/layouts/PublicLayout.tsx
git commit -m "feat: add xl padding and gap to Header, Footer, and PublicLayout"
```

---

## Chunk 5: Verification

### Task 11: Run lint and type-check

- [ ] **Step 1: Run verification**

```bash
pnpm lint && pnpm type-check
```

Expected: all pass with no errors

- [ ] **Step 2: Fix any issues if found**

- [ ] **Step 3: Build check**

```bash
pnpm build
```

Expected: successful build to `frontend/out/`
