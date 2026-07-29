# Plan: Official Theme Store Phase A

**Design:** [`docs/design-official-extension-store-phase-a.md`](../../design-official-extension-store-phase-a.md)  
**Date:** 2026-07-29  
**Status:** ready to implement

## Goal

Admin can browse an **official theme catalog** and **one-click install** (optional activate) into `installed_themes` + UMD load — without pasting URLs. No third-party listing, no plugin zip via marketplace.

## Task board

### Backend

- [x] **A1** Embed fallback catalog JSON + `INKLESS_THEME_CATALOG_URL` + URL host allowlist config  

- [x] **A2** Catalog fetch service (TTL cache, refresh flag, fallback on error)  

- [ ] **A3** `installState` merge (builtin / installed / active / incompatible) + unit tests  
- [ ] **A4** `GET /admin/extensions/themes/catalog`  
- [ ] **A5** `POST /admin/extensions/themes/install` (validate official, contract, allowlist, upsert)  
- [ ] **A6** `activate=true` → existing SetActive + SeedThemePages + cache invalidate  
- [ ] **A7** Persist `source=marketplace`; ensure public bootstrap exposes it  

### Frontend

- [ ] **A8** API client  
- [ ] **A9** Market UI (cards, badges, install buttons)  
- [ ] **A10** Install + activate + bootstrap refetch  
- [ ] **A11** Error / incompatible states  
- [ ] **A12** Nav entry + link from theme gallery  
- [ ] **A7-fe** `ThemeManagerContext`: loadExternal for `source === "marketplace"`  

### Ops / release

- [ ] **A15** Publish official `themes.json` + UMD release URLs for product-first / blog-first / editorial-firm  
- [ ] **A13** Update-to-latest action (optional P1)  
- [ ] **A14** sha256 verify when present (P1)  
- [ ] **A16** Smoke test script (P1)  

## Out of scope

Plugin marketplace install, third-party publish, signatures PKI, auto Features mutation, overwriting user page content.

## Implementation order

```text
A1 → A2 → A3 → A4 → A5 → A6 → A7 → A7-fe → A8 → A9–A12 → A15 → (A13/A14/A16)
```

## Done when

1. Catalog lists official themes with correct installState.  
2. One click installs without manual URL.  
3. Install+activate changes public active theme and theme pages.  
4. Incompatible contract cannot install.  
5. Built-in themes show as builtin (activate only).  
