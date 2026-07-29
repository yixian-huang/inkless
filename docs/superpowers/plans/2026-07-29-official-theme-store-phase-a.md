# Plan: Official Theme Store Phase A

**Design:** [`docs/design-official-extension-store-phase-a.md`](../../design-official-extension-store-phase-a.md)  
**Date:** 2026-07-29  
**Status:** P0 implemented (A1–A14, A16); A15 is ops release of real UMD assets

## Goal

Admin can browse an **official theme catalog** and **one-click install** (optional activate) into `installed_themes` + UMD load — without pasting URLs. No third-party listing, no plugin zip via marketplace.

## Task board

### Backend

- [x] **A1** Embed fallback catalog JSON + `INKLESS_THEME_CATALOG_URL` + URL host allowlist config  
- [x] **A2** Catalog fetch service (TTL cache, refresh flag, fallback on error)  
- [x] **A3** `installState` merge (builtin / installed / active / incompatible) + unit tests  
- [x] **A4** `GET /admin/extensions/themes/catalog`  
- [x] **A5** `POST /admin/extensions/themes/install` (validate official, contract, allowlist, upsert)  
- [x] **A6** `activate=true` → existing SetActive + SeedThemePages + cache invalidate  
- [x] **A7** Persist `source=marketplace`; ensure public bootstrap exposes it  
- [x] **A14** sha256 verify when catalog provides `latest.sha256`  

### Frontend

- [x] **A8** API client `extensionsThemes.ts`  
- [x] **A9** Market UI (cards, badges, install buttons)  
- [x] **A10** Install + activate + bootstrap refetch  
- [x] **A11** Error / incompatible states  
- [x] **A12** Nav entry + link from theme gallery  
- [x] **A7-fe** `ThemeManagerContext`: loadExternal for `source === "marketplace"`  
- [x] **A13** Update-to-latest (reuse install with same slug)  

### Ops / release

- [ ] **A15** Publish official `themes.json` + UMD release URLs for product-first / blog-first / editorial-firm  
- [x] **A16** Smoke script `scripts/smoke-theme-catalog.sh`  

## Out of scope

Plugin marketplace install, third-party publish, signatures PKI, auto Features mutation, overwriting user page content.

## Usage

```bash
# Backend with optional remote catalog
export INKLESS_THEME_CATALOG_URL=   # empty → embedded official_themes.json
# Admin UI
#   /admin/theme-market
# Smoke (API up, admin/admin123)
./scripts/smoke-theme-catalog.sh
```

## Done when

1. Catalog lists official themes with correct installState.  
2. One click installs without manual URL.  
3. Install+activate changes public active theme and theme pages.  
4. Incompatible contract cannot install.  
5. Built-in themes show as builtin (activate only).  
