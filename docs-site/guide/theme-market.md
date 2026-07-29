# Official theme market

Inkless can install **official** themes from a curated catalog without pasting a
UMD URL by hand.

## What you get

- Admin **Theme market** (`/admin/theme-market`): browse catalog cards, install,
  activate, update when a newer catalog version exists.
- Optional **theme auto-update** (default **off**): the server polls the catalog
  and upgrades installed marketplace package pointers (UMD URL + version). It
  does **not** switch which theme is active and is not for major theme redesigns.

## Public catalog

Default public index (product site):

```text
https://inkless.run/marketplace/v1/themes.json
```

Point an instance at it with:

```bash
INKLESS_THEME_CATALOG_URL=https://inkless.run/marketplace/v1/themes.json
```

If unset, the host uses an embedded fallback list. UMD packages are served from
**GitHub Releases** (catalog `umdUrl` fields); host allowlists HTTPS download
hosts for safety.

## Trust model

Only **official** catalog entries are installable in Phase A. External UMD themes
run as trusted frontend scripts — only install sources you control. See
[Extension points](/guide/extension-points).

## Related

- [Theme layout](/guide/theme-layout)
- Repo: `docs/theme-contract.md`, `docs/design-official-extension-store-phase-a.md`
